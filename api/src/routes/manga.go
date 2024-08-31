// Package routes implements the manga routes
package routes

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AnthonyHewins/gotfy"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/dashboard"
	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/integrations/kaizoku"
	"github.com/diogovalentte/mantium/api/src/integrations/tranga"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/notifications"
	"github.com/diogovalentte/mantium/api/src/sources"
	"github.com/diogovalentte/mantium/api/src/sources/models"
	"github.com/diogovalentte/mantium/api/src/util"
)

// MangaRoutes sets the manga routes
func MangaRoutes(group *gin.RouterGroup) {
	{
		group.POST("/manga/search", SearchManga)
		group.POST("/manga", AddManga)
		group.DELETE("/manga", DeleteManga)
		group.GET("/manga", GetManga)
		group.GET("/mangas", GetMangas)
		group.GET("/mangas/iframe", GetMangasiFrame)
		group.GET("/manga/chapters", GetMangaChapters)
		group.PATCH("/manga/status", UpdateMangaStatus)
		group.PATCH("/manga/last_read_chapter", UpdateMangaLastReadChapter)
		group.PATCH("/manga/cover_img", UpdateMangaCoverImg)
		group.PATCH("/mangas/metadata", UpdateMangasMetadata)
		group.POST("/mangas/add_to_kaizoku", AddMangasToKaizoku)
		group.POST("/mangas/add_to_tranga", AddMangasToTranga)
	}
}

// @Summary Search manga
// @Description Searches a manga in the source. You must provide the source site URL like "https://mangadex.org" and the search query.
// @Accept json
// @Produce json
// @Param manga body SearchMangaRequest true "Search data"
// @Success 200 {object} map[string][]models.MangaSearchResult "{"mangas": [mangaSearchResultObj]}"
// @Router /manga/search [post]
func SearchManga(c *gin.Context) {
	var requestData SearchMangaRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	if requestData.Limit == 0 {
		requestData.Limit = 20
	}
	mangas, err := sources.SearchManga(requestData.Term, requestData.SourceURL, requestData.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	resMap := map[string][]*models.MangaSearchResult{"mangas": mangas}
	c.JSON(http.StatusOK, resMap)
}

// SearchMangaRequest is the request body for the SearchManga route
type SearchMangaRequest struct {
	SourceURL string `json:"source_url" binding:"required,http_url"`
	Term      string `json:"q" binding:"required"`
	Limit     int    `json:"limit"`
}

// @Summary Add manga
// @Description Gets a manga metadata from source and inserts in the database.
// @Accept json
// @Produce json
// @Param manga_has_no_chapters query bool false "If true, assumes the manga has no chapters and sets the last released chapter to null without even checking if the manga really doesn't have released chapters. If false, gets the manga's last released chapter metadata from source. It doesn't do anything with the last read chapter. Defaults to false." Example(true).
// @Param manga body AddMangaRequest true "Manga data"
// @Success 200 {object} responseMessage
// @Router /manga [post]
func AddManga(c *gin.Context) {
	currentTime := time.Now()

	var requestData AddMangaRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	var mangaHasNoChapters bool
	queryMangaHasNoChapters := c.Query("manga_has_no_chapters")
	if queryMangaHasNoChapters != "" {
		switch queryMangaHasNoChapters {
		case "true":
			mangaHasNoChapters = true
		case "false":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"message": "manga_has_no_chapters must be a boolean like true or false"})
			return
		}
	}

	mangaAdd, err := sources.GetMangaMetadata(requestData.URL, !mangaHasNoChapters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	mangaAdd.Status = manga.Status(requestData.Status)

	if requestData.LastReadChapter != "" || requestData.LastReadChapterURL != "" {
		mangaAdd.LastReadChapter, err = sources.GetChapterMetadata(requestData.URL, requestData.LastReadChapter, requestData.LastReadChapterURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		mangaAdd.LastReadChapter.Type = 2
		mangaAdd.LastReadChapter.UpdatedAt = currentTime.Truncate(time.Second)
	}

	_, err = mangaAdd.InsertIntoDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	var integrationsErrors []error
	if config.GlobalConfigs.Kaizoku.Valid {
		kaizoku := kaizoku.Kaizoku{}
		kaizoku.Init()
		err = kaizoku.AddManga(mangaAdd, config.GlobalConfigs.Kaizoku.TryOtherSources)
		if err != nil {
			integrationsErrors = append(integrationsErrors, util.AddErrorContext("manga added to DB, but error while adding it to Kaizoku", err))
		}
	}

	if config.GlobalConfigs.Tranga.Valid {
		tranga := tranga.Tranga{}
		tranga.Init()
		err = tranga.AddManga(mangaAdd)
		if err != nil {
			integrationsErrors = append(integrationsErrors, util.AddErrorContext("manga added to DB, but error while adding it to Tranga", err))
		}
	}

	if len(integrationsErrors) > 0 {
		fullMsg := "manga added to DB, but error executing integrations: "
		for _, err := range integrationsErrors {
			zerolog.Ctx(c.Request.Context()).Error().Err(err).Msg("error while adding manga to integrations")
			fullMsg += err.Error() + " "
		}
		dashboard.UpdateDashboard()
		c.JSON(http.StatusInternalServerError, gin.H{"message": fullMsg})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga added successfully"})
}

// AddMangaRequest is the request body for the AddManga route
type AddMangaRequest struct {
	URL                string `json:"url" binding:"required,http_url"`
	LastReadChapter    string `json:"last_read_chapter"`
	LastReadChapterURL string `json:"last_read_chapter_url"`
	Status             int    `json:"status" binding:"required,gte=0,lte=5"`
}

// @Summary Delete manga
// @Description Deletes a manga from the database. You must provide either the manga ID or the manga URL.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga URL" Example("https://mangadex.org/title/1/one-piece")
// @Success 200 {object} responseMessage
// @Router /manga [delete]
func DeleteManga(c *gin.Context) {
	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	mangaID, mangaURL, err := getMangaIDAndURL(mangaIDStr, mangaURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	mangaDelete, err := manga.GetMangaDB(mangaID, mangaURL)
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	err = mangaDelete.DeleteFromDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga deleted successfully"})
}

// @Summary Get manga
// @Description Gets a manga from the database. You must provide either the manga ID or the manga URL.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga URL" Example("https://mangadex.org/title/1/one-piece")
// @Success 200 {object} manga.Manga "{"manga": mangaObj}"
// @Router /manga [get]
func GetManga(c *gin.Context) {
	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	mangaID, mangaURL, err := getMangaIDAndURL(mangaIDStr, mangaURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	mangaGet, err := manga.GetMangaDB(mangaID, mangaURL)
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	resMap := map[string]manga.Manga{"manga": *mangaGet}
	c.JSON(http.StatusOK, resMap)
}

// @Summary Get mangas
// @Description Gets all mangas from the database.
// @Produce json
// @Success 200 {array} manga.Manga "{"mangas": [mangaObj]}"
// @Router /mangas [get]
func GetMangas(c *gin.Context) {
	mangas, err := manga.GetMangasDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	resMap := map[string][]*manga.Manga{"mangas": mangas}
	c.JSON(http.StatusOK, resMap)
}

// @Summary Mangas iFrame
// @Description Returns an iFrame with mangas. Only mangas with unread chapters, and status reading or completed. Sort by last released chapter date. Designed to be used with [Homarr](https://github.com/ajnart/homarr).
// @Success 200 {string} string "HTML content"
// @Produce html
// @Param api_url query string true "API URL used by your browser. Used for the button that updates the last read chater, as your browser needs to send a request to the API to update the chapter." Example(https://sub.domain.com)
// @Param theme query string false "Homarr theme, defaults to light. If it's different from your Homarr theme, the background turns white" Example(light)
// @Param limit query int false "Limits the number of items in the iFrame." Example(5)
// @Param showBackgroundErrorWarning query bool false "If true, shows a warning in the iFrame if an error occurred in the background. Defaults to true." Example(true)
// @Router /mangas/iframe [get]
func GetMangasiFrame(c *gin.Context) {
	queryLimit := c.Query("limit")
	var limit int
	var err error
	if queryLimit == "" {
		limit = -1
	} else {
		limit, err = strconv.Atoi(queryLimit)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "limit must be a number"})
		}
	}

	theme := c.Query("theme")
	if theme == "" {
		theme = "light"
	} else if theme != "dark" && theme != "light" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "theme must be 'dark' or 'light'"})
		return
	}

	apiURL := c.Query("api_url")
	_, err = url.ParseRequestURI(apiURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "api_url must be a valid URL like 'http://192.168.1.46:8080' or 'https://sub.domain.com'"})
		return
	}

	showBackgroundErrorWarning := true
	showBackgroundErrorWarningStr := c.Query("showBackgroundErrorWarning")
	if showBackgroundErrorWarningStr != "" {
		showBackgroundErrorWarning, err = strconv.ParseBool(showBackgroundErrorWarningStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "showBackgroundErrorWarning must be a boolean"})
			return
		}
	}

	allMangas, err := manga.GetMangasDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	allUnreadMangas := manga.FilterUnreadChapterMangas(allMangas)
	mangas := []*manga.Manga{}
	for _, manga := range allUnreadMangas {
		if manga.Status == 1 || manga.Status == 2 {
			mangas = append(mangas, manga)
		}
	}
	manga.SortMangasByLastReleasedChapterUpdatedAt(mangas)

	if limit >= 0 && limit < len(mangas) {
		mangas = mangas[:limit]
	}

	html, err := getMangasiFrame(mangas, theme, apiURL, showBackgroundErrorWarning)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.Data(http.StatusOK, "text/html", []byte(html))
}

func getMangasiFrame(mangas []*manga.Manga, theme, apiURL string, showBackgroundErrorWarning bool) ([]byte, error) {
	html := `
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="referrer" content="no-referrer"> <!-- If not set, can't load Mangedex images when behind a domain or reverse proxy -->
    <script src="https://kit.fontawesome.com/3f763b063a.js" crossorigin="anonymous"></script>
    <meta name="color-scheme" content="{{ .Theme }}">
    <title>Mantium</title>
    <style>
        body {
            background: transparent !important;
            margin: 0;
            padding: 0;
            width: calc(100% - 3px);
        }

        .mangas-container {
            height: 84px;

            position: relative;
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 14px;

            border-radius: 10px;
            border: 1px solid rgba(56, 58, 64, 1);
        }

        .background-image {
            background-position: center;
            background-size: cover;
            position: absolute;
            filter: brightness(0.3);
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            z-index: -1;
            border-radius: 10px;
        }

        .manga-cover {
            border-radius: 2px;
            margin-left: 20px;
            margin-right: 20px;
            object-fit: cover;
            width: 30px;
            height: 50px;
        }

        .text-wrap {
            flex-grow: 1;
            overflow: hidden;
            white-space: nowrap;
            text-overflow: ellipsis;
            width: 1px !important;
            margin-right: 10px 0px 10px 10px;

            /* if the attributes below are overwritten in the inner elements, this set the ellipsis properties only */
            color: white; 
            font-weight: bold;
        }

        .manga-name {
            font-size: 15px;
            color: white;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji";
            text-decoration: none;
            font-weight: bold;
        }

        .manga-name:hover {
            text-decoration: underline;
        }

        .new-chapter-container {
            display: inline-block;
            padding: 8px 0px;
            margin: 20px 10px;
            background-color: transparent;
            border-radius: 5px;
            width: 162px;
            text-align: center;
        }

        .chapter-label {
            text-decoration: none;
            font-size: 20px;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
              Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji";
        }

        a.chapter-label:hover {
            text-decoration: underline;
        }

        .last-released-chapter-label {
            color: rgb(101, 206, 230);
        }

        .last-read-chapter-label {
            color: rgb(210, 101, 230);
        }

        .chapter-gt-label {
            color: rgb(101, 206, 230);
        }

        .set-last-read-button {
            color: white;
            background-color: #04c9b7;
            padding: 0.25rem 0.75rem;
            border-radius: 0.5rem;
            border: 1px solid rgb(4, 201, 183);
            margin-top: 10px;
            font-weight: bold;
        }

        button.set-last-read-button:hover {
            filter: brightness(0.9)
        }

        .delete-background-error-container {
            display: inline-block;
            padding: 8px 0px;
            margin: 20px 10px;
            background-color: transparent;
            border-radius: 5px;
            width: 162px;
            text-align: center;
        }

        #delete-background-error-button {
            color: red;
            background-color: white;
            padding: 0.25rem 0.75rem;
            border-radius: 0.5rem;
            border: 1px solid rgb(4, 201, 183);
            font-weight: bold;
        }

        button#delete-background-error-button:hover {
            filter: brightness(0.9)
        }

        .info-label {
            text-decoration: none;
            font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont,
              Segoe UI, Roboto, Helvetica Neue, Arial, Noto Sans, sans-serif, Apple Color Emoji,
              Segoe UI Emoji, Segoe UI Symbol, Noto Color Emoji;
            font-feature-settings: normal;
            font-variation-settings: normal;
            font-weight: 600;
            color: #4f6164;
            font-size: 1rem;
            line-height: 1.5rem;
        }

        ::-webkit-scrollbar {
            width: 7px;
        }

        ::-webkit-scrollbar-thumb {
            background-color: {{ .ScrollbarThumbBackgroundColor }};
            border-radius: 2.3px;
        }

        ::-webkit-scrollbar-track {
            background-color: transparent;
        }

        ::-webkit-scrollbar-track:hover {
            background-color: {{ .ScrollbarTrackBackgroundColor }};
        }
    </style>

    <script>
      function setLastReadChapter(mangaId) {
        try {
            var xhr = new XMLHttpRequest();
            var url = '{{ .APIURL }}/v1/manga/last_read_chapter?id=' + encodeURIComponent(mangaId);
            xhr.open('PATCH', url, true);
            xhr.setRequestHeader('Content-Type', 'application/json');

            xhr.onload = function () {
              if (xhr.status >= 200 && xhr.status < 300) {
                console.log('Request to update manga', mangaId, ' last read chapter finished with success:', xhr.responseText);
                location.reload();
              } else {
                console.log('Request to update manga', mangaId, ' last read chapter failed:', xhr.responseText);
                handleSetLastReadChapterError("manga-" + mangaId)
              }
            };

            xhr.onerror = function () {
              console.log('Request to update manga', mangaId, ' last read chapter failed:', xhr.responseText);
              handleSetLastReadChapterError(mangaId)
            };

            var body = {};

            xhr.send(JSON.stringify(body));
        } catch (error) {
            console.log('Request to update manga', mangaId, ' last read chapter failed:', error);
            handleSetLastReadChapterError("manga-" + mangaId)
        }
      }

      function handleSetLastReadChapterError(buttonId) {
        var button = document.getElementById(buttonId);
        button.textContent = "! ERROR !";
        button.style.backgroundColor = "red";
        button.style.borderColor = "red";
      }

      function deleteBackgroundError() {
        try {
            var xhr = new XMLHttpRequest();
            var url = '{{ .APIURL }}/v1/dashboard/last_background_error';
            xhr.open('DELETE', url, true);
            xhr.setRequestHeader('Content-Type', 'application/json');

            xhr.onload = function () {
              if (xhr.status >= 200 && xhr.status < 300) {
                console.log('Request to delete background error finished with success:', xhr.responseText);
                location.reload();
              } else {
                console.log('Request to delete background error failed:', xhr.responseText);
                handleDeleteBackgroundError()
              }
            };

            xhr.onerror = function () {
              console.log('Request to delete background error failed:', xhr.responseText);
              handleDeleteBackgroundError()
            };

            xhr.send();
        } catch (error) {
            console.log('Request to delete background error failed:', xhr.responseText);
            handleDeleteBackgroundError()
        }
      }

      function handleDeleteBackgroundError() {
        var button = document.getElementById('delete-background-error-button');
        button.textContent = "! ERROR !";
      }
    </script>

    <script>
        let lastUpdate = null;

        async function fetchData() {
            try {
                var url = '{{ .APIURL }}/v1/dashboard/last_update';
                const response = await fetch(url);
                const data = await response.json();

                if (lastUpdate === null) {
                    lastUpdate = data.message;
                } else {
                    if (data.message !== lastUpdate) {
                        lastUpdate = data.message;
                        location.reload();
                    }
                }
            } catch (error) {
                console.error('Error getting last update from the API:', error);
            }
        }

        function fetchAndUpdate() {
            fetchData();
            setTimeout(fetchAndUpdate, 5000); // 5 seconds
        }

        fetchAndUpdate();
    </script>

  </head>
<body>
{{ if .ShowBackgroundError }}
<div class="mangas-container" style="background-color: red;">
    <div class="text-wrap" style="margin-left: 20px;">
        <span class="manga-name">An error occured in the background.</span>

        <div>
            <span style="margin-right: 7px;" class="info-label"><i class="fa-solid fa-calendar-days"></i> {{ .BackgroundErrorTime.Format "2006-01-02 15:04:05" }}</span>
        </div>
    </div>
    <div class="delete-background-error-container">
        <button id="delete-background-error-button" onclick="deleteBackgroundError()" onmouseenter="this.style.cursor='pointer';">Delete Error</button>
    </div>
</div>
{{ end }}
{{range .Mangas }}
    <div class="mangas-container">

    <div style="background-image: url('data:image/jpeg;base64,{{ encodeImage .CoverImg }}');" class="background-image"></div>

        <img
            class="manga-cover"
            src="data:image/jpeg;base64,{{ encodeImage .CoverImg }}"
            alt="Manga Cover"
        />

        <div class="text-wrap">
            <a href="{{ .URL }}" target="_blank" class="manga-name">{{ .Name }}</a>
        </div>

        <div class="new-chapter-container">
            {{ if .LastReadChapter }}
                <a href="{{ .LastReadChapter.URL }}" class="chapter-label last-read-chapter-label" target="_blank">{{ .LastReadChapter.Chapter }}</a>
            {{ else }}
                <a href="{{ .URL }}" class="chapter-label last-read-chapter-label" target="_blank">N/A</a>
            {{ end }}
            <span class="chapter-label chapter-gt-label"> &lt; </span>
            {{ if .LastReleasedChapter }}
                <a href="{{ .LastReleasedChapter.URL }}" class="chapter-label last-released-chapter-label" target="_blank">{{ .LastReleasedChapter.Chapter }}</a>
            {{ else }}
                <a href="{{ .URL }}" class="chapter-label last-released-chapter-label" target="_blank">N/A</a>
            {{ end }}

            <div>
                <button id="manga-{{ .ID }}" onclick="setLastReadChapter('{{ .ID }}')" class="set-last-read-button" onmouseenter="this.style.cursor='pointer';">Set last read</button>
            </div>
        </div>

    </div>
{{end}}
</body>
</html>
	`
	// Homarr theme
	scrollbarThumbBackgroundColor := "rgba(209, 219, 227, 1)"
	scrollbarTrackBackgroundColor := "#ffffff"
	if theme == "dark" {
		scrollbarThumbBackgroundColor = "#484d64"
		scrollbarTrackBackgroundColor = "rgba(37, 40, 53, 1)"
	}

	templateData := iframeTemplateData{
		Mangas:                        mangas,
		Theme:                         theme,
		APIURL:                        apiURL,
		ScrollbarThumbBackgroundColor: scrollbarThumbBackgroundColor,
		ScrollbarTrackBackgroundColor: scrollbarTrackBackgroundColor,
	}
	lastBackgroundError := dashboard.GetLastBackgroundError()
	if lastBackgroundError.Message != "" && showBackgroundErrorWarning {
		templateData.ShowBackgroundError = true
		templateData.BackgroundErrorTime = lastBackgroundError.Time
	}

	encodeImageF := template.FuncMap{"encodeImage": func(bytes []byte) string {
		return base64.StdEncoding.EncodeToString(bytes)
	}}

	tmpl := template.Must(template.New("mangas").Funcs(encodeImageF).Parse(html))

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, templateData)
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

type iframeTemplateData struct {
	Mangas                        []*manga.Manga
	Theme                         string
	APIURL                        string
	ScrollbarThumbBackgroundColor string
	ScrollbarTrackBackgroundColor string
	ShowBackgroundError           bool
	BackgroundErrorTime           time.Time
}

// @Summary Get manga chapters
// @Description Get a manga chapters from the source. You must provide either the manga ID or the manga URL.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga URL" Example("https://mangadex.org/title/1/one-piece")
// @Success 200 {array} manga.Chapter "{"chapters": [chapterObj]}"
// @Router /manga/chapters [get]
func GetMangaChapters(c *gin.Context) {
	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	mangaID, mangaURL, err := getMangaIDAndURL(mangaIDStr, mangaURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if mangaURL == "" {
		mangaGet, err := manga.GetMangaDB(mangaID, mangaURL)
		if err != nil {
			if strings.Contains(err.Error(), errordefs.ErrMangaNotFoundDB.Error()) {
				c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		mangaURL = mangaGet.URL
	}

	chapters, err := sources.GetMangaChapters(mangaURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	resMap := map[string][]*manga.Chapter{"chapters": chapters}
	c.JSON(http.StatusOK, resMap)
}

// @Summary Update manga status
// @Description Updates a manga status in the database. You must provide either the manga ID or the manga URL.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga URL" Example("https://mangadex.org/title/1/one-piece")
// @Param status body UpdateMangaStatusRequest true "Manga status"
// @Success 200 {object} responseMessage
// @Router /manga/status [patch]
func UpdateMangaStatus(c *gin.Context) {
	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	mangaID, mangaURL, err := getMangaIDAndURL(mangaIDStr, mangaURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	mangaUpdate, err := manga.GetMangaDB(mangaID, mangaURL)
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	var requestData UpdateMangaStatusRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	err = mangaUpdate.UpdateStatusInDB(requestData.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga status updated successfully"})
}

// UpdateMangaStatusRequest is the request body for the UpdateMangaStatus route
type UpdateMangaStatusRequest struct {
	Status manga.Status `json:"status" binding:"required,gte=0,lte=5"`
}

// @Summary Update manga last read chapter
// @Description Updates a manga last read chapter in the database. If both `chapter` and `chapter_url` are empty strings in the body, set the last read chapter to the last released chapter in the database. You must provide either the manga ID or the manga URL.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga URL" Example("https://mangadex.org/title/1/one-piece")
// @Param status body UpdateMangaChapterRequest true "Manga status"
// @Success 200 {object} responseMessage
// @Router /manga/last_read_chapter [patch]
func UpdateMangaLastReadChapter(c *gin.Context) {
	currentTime := time.Now()

	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	mangaID, mangaURL, err := getMangaIDAndURL(mangaIDStr, mangaURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	var requestData UpdateMangaChapterRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	mangaUpdate, err := manga.GetMangaDB(mangaID, mangaURL)
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	var chapter *manga.Chapter
	if requestData.Chapter == "" && requestData.ChapterURL == "" {
		chapter = mangaUpdate.LastReleasedChapter
	} else {
		chapter, err = sources.GetChapterMetadata(mangaUpdate.URL, requestData.Chapter, requestData.ChapterURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}
	chapter.Type = 2
	chapter.UpdatedAt = currentTime.Truncate(time.Second)

	err = mangaUpdate.UpsertChapterInDB(chapter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga last read chapter updated successfully"})
}

// UpdateMangaChapterRequest is the request body for updating a manga chapter
type UpdateMangaChapterRequest struct {
	Chapter    string `json:"chapter,omitempty"`
	ChapterURL string `json:"chapter_url,omitempty"`
}

// @Summary Update manga cover image
// @Description Updates a manga cover image in the database. You must provide either the manga ID or the manga URL. You must provide only one of the following: cover_img, cover_img_url, get_cover_img_from_source.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga URL" Example("https://mangadex.org/title/1/one-piece")
// @Param cover_img formData file false "Manga cover image file"
// @Param cover_img_url query string false "Manga cover image URL" Example("https://example.com/cover.jpg")
// @Param get_cover_img_from_source query bool false "Let Mantium fetch the cover image from the source site" Example(true)
// @Success 200 {object} responseMessage
// @Router /manga/cover_img [patch]
func UpdateMangaCoverImg(c *gin.Context) {
	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	coverImgURL := c.Query("cover_img_url")
	getCoverImgFromSource := c.Query("get_cover_img_from_source")

	var requestFile multipart.File
	requestFile, _, err := c.Request.FormFile("cover_img")
	if err != nil && err != http.ErrMissingFile {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	defer requestFile.Close()
	coverImg, err := io.ReadAll(requestFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	var score int
	if coverImgURL != "" {
		score++
	}
	if getCoverImgFromSource != "" {
		score++
	}
	if len(coverImg) != 0 {
		score++
	}

	switch score {
	case 0:
		c.JSON(http.StatusBadRequest, gin.H{"message": "you must provide one of the following: cover_img, cover_img_url, get_cover_img_from_source"})
		return
	case 1:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"message": "you must provide only one of the following: cover_img, cover_img_url, get_cover_img_from_source"})
		return
	}

	mangaID, mangaURL, err := getMangaIDAndURL(mangaIDStr, mangaURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	mangaToUpdate, err := manga.GetMangaDB(mangaID, mangaURL)
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	retries := 3
	retryInterval := 3 * time.Second
	if len(coverImg) != 0 {
		if !util.IsImageValid(coverImg) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid image"})
			return
		}

		mangaToUpdate.CoverImgFixed = true

		isImgRezied := false
		resizedCoverImg, err := util.ResizeImage(coverImg, uint(util.DefaultImageWidth), uint(util.DefaultImageHeight))
		if err == nil {
			isImgRezied = true
			err = mangaToUpdate.UpdateCoverImgInDB(resizedCoverImg, isImgRezied, coverImgURL)
		} else {
			err = mangaToUpdate.UpdateCoverImgInDB(coverImg, isImgRezied, coverImgURL)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	} else if coverImgURL != "" {
		coverImg, isImgRezied, err := util.GetImageFromURL(coverImgURL, retries, retryInterval)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		if !util.IsImageValid(coverImg) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid image"})
			return
		}

		mangaToUpdate.CoverImgFixed = true

		err = mangaToUpdate.UpdateCoverImgInDB(coverImg, isImgRezied, coverImgURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	} else if getCoverImgFromSource == "true" {
		var updatedManga *manga.Manga
		for i := 0; i < retries; i++ {
			updatedManga, err = sources.GetMangaMetadata(mangaToUpdate.URL, true)
			if err != nil {
				if i == retries-1 {
					c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
					return
				}
				time.Sleep(retryInterval)
				continue
			}
		}

		mangaToUpdate.CoverImgFixed = false

		err = mangaToUpdate.UpdateCoverImgInDB(updatedManga.CoverImg, updatedManga.CoverImgResized, updatedManga.CoverImgURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga cover image updated successfully"})
}

// @Summary Update mangas metadata
// @Description Get the mangas metadata from the sources and update them in the database.
// @Produce json
// @Param notify query string false "Notify if a new chapter was released for the manga (only of mangas with status reading or completed)."
// @Success 200 {object} responseMessage
// @Router /mangas/metadata [patch]
func UpdateMangasMetadata(c *gin.Context) {
	notifyStr := c.Query("notify")
	var notify bool
	if notifyStr == "true" {
		notify = true
	}

	mangas, err := manga.GetMangasDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	logger := util.GetLogger(zerolog.Level(config.GlobalConfigs.API.LogLevelInt))
	errors := map[string][]string{
		"manga_metadata": {},
		"ntfy":           {},
		"tranga":         {},
		"kaizoku":        {},
	}
	var newMetadata bool
	var trangaInt *tranga.Tranga = nil
	if config.GlobalConfigs.Tranga.Valid {
		trangaInt = &tranga.Tranga{}
		trangaInt.Init()
	}
	retries := 3
	retryInterval := 3 * time.Second
	for _, mangaToUpdate := range mangas {
		var updatedManga *manga.Manga
		for i := 0; i < retries; i++ {
			updatedManga, err = sources.GetMangaMetadata(mangaToUpdate.URL, mangaToUpdate.LastReleasedChapter == nil)
			if err != nil {
				if i != retries-1 {
					logger.Error().Err(err).Str("manga_url", mangaToUpdate.URL).Msgf("Error getting manga metadata, retrying in %.2f seconds...", retryInterval.Seconds())
					time.Sleep(retryInterval)
					continue
				}
			}
			break
		}
		if updatedManga == nil {
			logger.Error().Err(err).Str("manga_url", mangaToUpdate.URL).Msg("Error getting manga metadata, will continue with the next manga...")
			errors["manga_metadata"] = append(errors["manga_metadata"], err.Error())
			continue
		}
		updatedManga.Status = 1

		mangaHasNewReleasedChapter := isNewChapterDifferentFromOld(mangaToUpdate.LastReleasedChapter, updatedManga.LastReleasedChapter)
		if mangaHasNewReleasedChapter || (!mangaToUpdate.CoverImgFixed && (mangaToUpdate.CoverImgURL != updatedManga.CoverImgURL || !bytes.Equal(mangaToUpdate.CoverImg, updatedManga.CoverImg))) || mangaToUpdate.Name != updatedManga.Name {
			newMetadata = true
			updatedManga.CoverImgFixed = mangaToUpdate.CoverImgFixed
			err = manga.UpdateMangaMetadataDB(updatedManga)
			if err != nil {
				logger.Error().Err(err).Str("manga_url", mangaToUpdate.URL).Msg("Error saving manga new metadata to DB, will continue with the next manga...")
				errors["manga_metadata"] = append(errors["manga_metadata"], err.Error())
				continue
			}

			// Notify only if the manga's status is 1 (reading) or 2 (completed)
			if notify && (mangaToUpdate.Status == 1 || mangaToUpdate.Status == 2) {
				if mangaHasNewReleasedChapter {
					for j := 0; j < retries; j++ {
						err = NotifyMangaLastReleasedChapterUpdate(mangaToUpdate, updatedManga)
						if err != nil {
							if j == retries-1 {
								logger.Error().Err(err).Str("manga_url", mangaToUpdate.URL).Msg(fmt.Sprintf("Manga metadata updated in DB, but error while notifying: %s.\nWill continue with the next manga...", err.Error()))
								errors["ntfy"] = append(errors["ntfy"], err.Error())
								break
							}
							logger.Error().Err(err).Str("manga_url", mangaToUpdate.URL).Msgf("Manga metadata updated in DB, but error while notifying: %s.\nRetrying in %.2f seconds...", err.Error(), retryInterval.Seconds())
							continue
						}
						break
					}
				}
			}

			if trangaInt != nil {
				err = trangaInt.StartJob(mangaToUpdate)
				if err != nil {
					logger.Error().Err(err).Str("manga_url", mangaToUpdate.URL).Msg("Manga metadata updated in DB, but error starting job in Tranga.\nWill continue with the next manga...")
					errors["tranga"] = append(errors["tranga"], err.Error())
				}

			}
		}
	}

	if newMetadata {
		dashboard.UpdateDashboard()
	}

	if config.GlobalConfigs.Kaizoku.Valid && newMetadata {
		err = KaizokuTriggerChaptersDownload(logger)
		if err != nil {
			errors["kaizoku"] = append(errors["kaizoku"], err.Error())
		}
	}

	for _, errSlice := range errors {
		if len(errSlice) > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "some errors occured while updating the mangas metadata, check the logs for more information", "errors": errors})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mangas metadata updated successfully"})
}

// @Summary Add mangas to Kaizoku
// @Description Add the mangas in the database to Kaizoku. If it fails to add a manga, it will continue with the next manga. This is a heavy operation depending on the number of mangas in the database.
// @Produce json
// @Param status query []int false "Filter which mangas to add by status. 1=reading, 2=completed, 3=on hold, 4=dropped, 5=plan to read. Example: status=1,2,3,5" Example(1,2,3,5)
// @Success 200 {object} responseMessage
// @Router /mangas/add_to_kaizoku [post]
func AddMangasToKaizoku(c *gin.Context) {
	if !config.GlobalConfigs.Kaizoku.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"message": "kaizoku is not configured in the API"})
		return
	}

	statusFilterStr := c.Query("status")
	var statusFilter []int
	if statusFilterStr != "" {
		statusStrings := strings.Split(statusFilterStr, ",")
		for _, statusStr := range statusStrings {
			status, err := strconv.Atoi(statusStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "status must be a list of numbers"})
				return
			}
			statusFilter = append(statusFilter, status)
		}
	}

	mangas, err := manga.GetMangasDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	kaizoku := kaizoku.Kaizoku{}
	kaizoku.Init()
	logger := util.GetLogger(zerolog.Level(config.GlobalConfigs.API.LogLevelInt))
	var lastError error
	for _, dbManga := range mangas {
		if len(statusFilter) > 0 {
			var found bool
			for _, status := range statusFilter {
				if dbManga.Status == manga.Status(status) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		err = kaizoku.AddManga(dbManga, config.GlobalConfigs.Kaizoku.TryOtherSources)
		if err != nil {
			logger.Error().Err(err).Str("manga_url", dbManga.URL).Msg("error adding manga to Kaizoku, will continue with the next manga...")
			lastError = err
			continue
		}
	}

	if lastError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "some errors occured while adding some mangas to Kaizoku, check the logs for more information. Last error: " + lastError.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mangas added to Kaizoku successfully"})
}

// @Summary Add mangas to Tranga
// @Description Add the mangas in the database to Tranga. If it fails to add a manga, it will continue with the next manga. This is a heavy operation depending on the number of mangas in the database. Currently, only MangaDex mangas can be added to Tranga, but it'll try all mangas anyway.
// @Produce json
// @Param status query []int false "Filter which mangas to add by status. 1=reading, 2=completed, 3=on hold, 4=dropped, 5=plan to read. Example: status=1,2,3,5" Example(1,2,3,5)
// @Success 200 {object} responseMessage
// @Router /mangas/add_to_tranga [post]
func AddMangasToTranga(c *gin.Context) {
	if !config.GlobalConfigs.Tranga.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"message": "tranga is not configured in the API"})
		return
	}

	statusFilterStr := c.Query("status")
	var statusFilter []int
	if statusFilterStr != "" {
		statusStrings := strings.Split(statusFilterStr, ",")
		for _, statusStr := range statusStrings {
			status, err := strconv.Atoi(statusStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "status must be a list of numbers"})
				return
			}
			statusFilter = append(statusFilter, status)
		}
	}

	mangas, err := manga.GetMangasDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	trangaInt := tranga.Tranga{}
	trangaInt.Init()
	logger := util.GetLogger(zerolog.Level(config.GlobalConfigs.API.LogLevelInt))
	var errorSlice []string
	for _, dbManga := range mangas {
		if len(statusFilter) > 0 {
			var found bool
			for _, status := range statusFilter {
				if dbManga.Status == manga.Status(status) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		err = trangaInt.AddManga(dbManga)
		if err != nil {
			logger.Error().Err(err).Str("manga_url", dbManga.URL).Msg("error adding manga to Tranga, will continue with the next manga...")
			errorSlice = append(errorSlice, err.Error())
			continue
		}
	}

	if len(errorSlice) > 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "some errors occured while adding some mangas to Kaizoku, check the logs for more information", "errors": errorSlice})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mangas added to Tranga successfully"})
}

func getMangaIDAndURL(mangaIDStr string, mangaURL string) (manga.ID, string, error) {
	if mangaIDStr == "" && mangaURL == "" {
		err := fmt.Errorf("you must provide either the manga ID or the manga URL")
		return -1, "", err
	}

	mangaID := manga.ID(-1)
	if mangaIDStr != "" {
		mangaIDInt, err := strconv.Atoi(mangaIDStr)
		if err != nil {
			return -1, "", err
		}
		mangaID = manga.ID(mangaIDInt)
	}
	if mangaURL != "" {
		_, err := url.ParseRequestURI(mangaURL)
		if err != nil {
			return -1, "", err
		}
	}

	return mangaID, mangaURL, nil
}

// NotifyMangaLastReleasedChapterUpdate notifies a manga last released chapter update
func NotifyMangaLastReleasedChapterUpdate(oldManga *manga.Manga, newManga *manga.Manga) error {
	publisher, err := notifications.GetNtfyPublisher()
	if err != nil {
		return err
	}

	title := fmt.Sprintf("Mantium - New chapter of manga: %s", newManga.Name)

	var message string
	if oldManga.LastReleasedChapter != nil {
		message = fmt.Sprintf("New chapter: %s\nLast chapter: %s", newManga.LastReleasedChapter.Chapter, oldManga.LastReleasedChapter.Chapter)
	} else {
		message = fmt.Sprintf("New chapter: %s\nLast chapter: N/A", newManga.LastReleasedChapter.Chapter)
	}

	chapterLink, err := url.Parse(newManga.LastReleasedChapter.URL)
	if err != nil {
		return err
	}

	msg := &gotfy.Message{
		Topic:   publisher.Topic,
		Title:   title,
		Message: message,
		Actions: []gotfy.ActionButton{
			&gotfy.ViewAction{
				Label: "Open Chapter",
				Link:  chapterLink,
				Clear: false,
			},
		},
		ClickURL: chapterLink,
	}

	ctx := context.Background()
	err = publisher.SendMessage(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}

func isNewChapterDifferentFromOld(oldChapter *manga.Chapter, newChapter *manga.Chapter) bool {
	if oldChapter == nil && newChapter != nil {
		return true
	}
	if oldChapter != nil && newChapter != nil && oldChapter.Chapter != newChapter.Chapter {
		return true
	}

	return false
}

func KaizokuTriggerChaptersDownload(logger *zerolog.Logger) error {
	kaizoku := kaizoku.Kaizoku{}
	kaizoku.Init()

	waitUntilEmptyQueuesTimeout := config.GlobalConfigs.Kaizoku.WaitUntilEmptyQueuesTimeout
	retryInterval := 5 * time.Second
	maxRetries := 12

	logger.Info().Msg("Adding job to check out of sync chapters to queue in Kaizoku...")
	logger.Info().Msg("Waiting for checkOutOfSyncChaptersQueue and fixOutOfSyncChaptersQueue queues to be empty in Kaizoku...")
	err := waitUntilEmptyCheckFixOutOfSyncChaptersQueues(&kaizoku, waitUntilEmptyQueuesTimeout, retryInterval, logger)
	if err != nil {
		return err
	}
	err = retryKaizokuJob(kaizoku.CheckOutOfSyncChapters, maxRetries, retryInterval, logger, "Error adding job to check out of sync chapters to queue in Kaizoku")
	if err != nil && !util.ErrorContains(err, "there is another active job running") {
		return util.AddErrorContext("error adding job to check out of sync chapters to queue in Kaizoku", err)
	}

	logger.Info().Msg("Adding job to fix out of sync chapters to queue in Kaizoku...")
	logger.Info().Msg("Waiting for checkOutOfSyncChaptersQueue and fixOutOfSyncChaptersQueue queues to be empty in Kaizoku...")
	err = waitUntilEmptyCheckFixOutOfSyncChaptersQueues(&kaizoku, waitUntilEmptyQueuesTimeout, retryInterval, logger)
	if err != nil {
		return err
	}
	err = retryKaizokuJob(kaizoku.FixOutOfSyncChapters, maxRetries, retryInterval, logger, "Error adding job to fix out of sync chapters to queue in Kaizoku")
	if err != nil {
		return util.AddErrorContext("error adding job to fix out of sync chapters to queue in Kaizoku", err)
	}

	logger.Info().Msg("Adding job to retry failed to fix out of sync chapters to queue in Kaizoku...")
	logger.Info().Msg("Waiting for checkOutOfSyncChaptersQueue and fixOutOfSyncChaptersQueue queues to be empty in Kaizoku...")
	err = waitUntilEmptyCheckFixOutOfSyncChaptersQueues(&kaizoku, waitUntilEmptyQueuesTimeout, retryInterval, logger)
	if err != nil {
		return err
	}
	err = retryKaizokuJob(kaizoku.RetryFailedFixOutOfSyncChaptersQueueJobs, maxRetries, retryInterval, logger, "Error adding job to try failed to fix out of sync chapters to queue in Kaizoku")
	if err != nil && !util.ErrorContains(err, "there is another active job running") {
		return util.AddErrorContext("error adding job to try failed to fix out of sync chapters to queue in Kaizoku", err)
	}

	return nil
}

func waitUntilEmptyCheckFixOutOfSyncChaptersQueues(kaizoku *kaizoku.Kaizoku, timeout time.Duration, retryInterval time.Duration, logger *zerolog.Logger) error {
	result := make(chan error)
	go func() {
		for {
			jobsCount, err := getCheckFixOutOfSyncChaptersActiveWaitingJobs(kaizoku)
			if err != nil {
				result <- err
				return
			}
			logger.Debug().Msgf("Jobs in checkOutOfSyncChaptersQueue and fixOutOfSyncChaptersQueue queues: %d", jobsCount)
			if jobsCount == 0 {
				result <- nil
				return
			}
			time.Sleep(retryInterval)
		}
	}()

	select {
	case <-time.After(timeout):
		return fmt.Errorf("timeout while waiting for checkOutOfSyncChaptersQueue and fixOutOfSyncChaptersQueue queues to be empty in Kaizoku. Current timeout is %s, maybe try to increase it?", timeout.String())
	case err := <-result:
		if err != nil {
			return err
		}
	}

	return nil
}

func retryKaizokuJob(jobFunc func() error, maxRetries int, retryInterval time.Duration, logger *zerolog.Logger, errorMessage string) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = jobFunc()
		if err == nil {
			return nil
		}

		if util.ErrorContains(err, "there is another active job running") {
			return err
		}

		logger.Error().Err(err).Msg(errorMessage)
		if i < maxRetries-1 {
			logger.Info().Msgf("Retrying in %s...", retryInterval)
		}
		time.Sleep(retryInterval)
	}

	return err
}

func getCheckFixOutOfSyncChaptersActiveWaitingJobs(kaizoku *kaizoku.Kaizoku) (int, error) {
	queues, err := kaizoku.GetQueues()
	if err != nil {
		return 0, err
	}

	jobsCount := 0
	for _, queue := range queues {
		if queue.Name != "checkOutOfSyncChaptersQueue" && queue.Name != "fixOutOfSyncChaptersQueue" {
			continue
		}
		jobsCount += queue.Counts.Active + queue.Counts.Waiting
	}

	return jobsCount, nil
}
