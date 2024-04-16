// Package routes implements the manga routes
package routes

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AnthonyHewins/gotfy"
	"github.com/gin-gonic/gin"

	"github.com/diogovalentte/mantium/api/src/dashboard"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/notifications"
	"github.com/diogovalentte/mantium/api/src/sources"
)

// MangaRoutes sets the manga routes
func MangaRoutes(group *gin.RouterGroup) {
	{
		group.POST("/manga", AddManga)
		group.DELETE("/manga", DeleteManga)
		group.GET("/manga", GetManga)
		group.GET("/mangas", GetMangas)
		group.GET("/mangas/iframe", GetMangasiFrame)
		group.GET("/manga/chapters", GetMangaChapters)
		group.PATCH("/manga/status", UpdateMangaStatus)
		group.PATCH("/manga/last_read_chapter", UpdateMangaLastReadChapter)
		group.PATCH("/mangas/metadata", UpdateMangasMetadata)
	}
}

// @Summary Add manga
// @Description Gets a manga metadata from source and inserts in the database.
// @Accept json
// @Produce json
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

	mangaAdd, err := sources.GetMangaMetadata(requestData.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	mangaAdd.Status = manga.Status(requestData.Status)

	// Last read chapter is not optional
	mangaAdd.LastReadChapter, err = sources.GetChapterMetadata(requestData.URL, requestData.LastReadChapter, requestData.LastReadChapterURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	mangaAdd.LastReadChapter.Type = 2
	mangaAdd.LastReadChapter.UpdatedAt = currentTime

	_, err = mangaAdd.InsertIntoDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga added successfully"})
}

// AddMangaRequest is the request body for the AddManga route
type AddMangaRequest struct {
	URL                string `json:"url" binding:"required,http_url"`
	Status             int    `json:"status" binding:"required,gte=0,lte=5"`
	LastReadChapter    string `json:"last_read_chapter"`
	LastReadChapterURL string `json:"last_read_chapter_url"`
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
		if strings.Contains(err.Error(), "manga not found in DB") {
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
		if strings.Contains(err.Error(), "manga not found in DB") {
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
// @Description Returns an iFrame with mangas. Only mangas with unread chapters, and status reading or completed. Sort by last upload chapter date. Designed to be used with [Homarr](https://github.com/ajnart/homarr).
// @Success 200 {string} string "HTML content"
// @Produce html
// @Param api_url query string true "API URL used by your browser. Used for the button that updates the last read chater, as your browser needs to send a request to the API to update the chapter." Example(https://sub.domain.com)
// @Param theme query string false "Homarr theme, defaults to light. If it's different from your Homarr theme, the background turns white" Example(light)
// @Param limit query int false "Limits the number of items in the iFrame." Example(5)
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
	manga.SortMangasByLastUploadChapterUpdatedAt(mangas)

	if limit >= 0 && limit < len(mangas) {
		mangas = mangas[:limit]
	}

	html, err := getMangasiFrame(mangas, theme, apiURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.Data(http.StatusOK, "text/html", []byte(html))
}

func getMangasiFrame(mangas []*manga.Manga, theme, apiURL string) ([]byte, error) {
	html := `
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="referrer" content="no-referrer"> <!-- If not set, can't load Mangedex images when behind a domain or reverse proxy -->
    <script src="https://kit.fontawesome.com/3f763b063a.js" crossorigin="anonymous"></script>
    <meta name="color-scheme" content="MANGAS-CONTAINER-BACKGROUND-COLOR">
    <title>Movie Display Template</title>
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

        .last-upload-chapter-label {
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
            background-color: SCROLLBAR-THUMB-BACKGROUND-COLOR;
            border-radius: 2.3px;
        }

        ::-webkit-scrollbar-track {
            background-color: transparent;
        }

        ::-webkit-scrollbar-track:hover {
            background-color: SCROLLBAR-TRACK-BACKGROUND-COLOR;
        }
    </style>

    <script>
      function setLastReadChapter(mangaId, chapter) {
        try {
            var xhr = new XMLHttpRequest();
            var url = 'API-URL/v1/manga/last_read_chapter?id=' + encodeURIComponent(mangaId);
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

            var body = {
                chapter: chapter
            };

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
    </script>

    <script>
        let lastUpdate = null;

        async function fetchData() {
            try {
                var url = 'API-URL/v1/dashboard/last_update';
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
BACKGROUND-ERROR-HTML
{{range .}}
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
            <a href="{{ .LastReadChapter.URL }}" class="chapter-label last-read-chapter-label" target="_blank">{{ .LastReadChapter.Chapter }}</a>
                <span class="chapter-label chapter-gt-label"> &lt; </span>
            <a href="{{ .LastUploadChapter.URL }}" class="chapter-label last-upload-chapter-label" target="_blank">{{ .LastUploadChapter.Chapter }}</a>

            <div>
                <button id="manga-{{ .ID }}" onclick="setLastReadChapter('{{ .ID }}', {{ .LastUploadChapter.Chapter }})" class="set-last-read-button" onmouseenter="this.style.cursor='pointer';">Set last read</button>
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

	html = strings.Replace(html, "API-URL", apiURL, -1)
	html = strings.Replace(html, "MANGAS-CONTAINER-BACKGROUND-COLOR", theme, -1)
	html = strings.Replace(html, "SCROLLBAR-THUMB-BACKGROUND-COLOR", scrollbarThumbBackgroundColor, -1)
	html = strings.Replace(html, "SCROLLBAR-TRACK-BACKGROUND-COLOR", scrollbarTrackBackgroundColor, -1)

	lastBackgroundError := dashboard.GetLastBackgroundError()
	if lastBackgroundError.Message != "" {
		backgroundErrorHTML := `
<div class="mangas-container" style="background-color: red;">
    <div class="text-wrap" style="margin-left: 20px;">
        <span class="manga-name">An error occured in the background. Check the dashboard and your logs.</span>

        <div>
            <span style="margin-right: 7px;" class="info-label"><i class="fa-solid fa-calendar-days"></i> ERROR-TIME</span>
        </div>
    </div>
</div>
        `
		backgroundErrorHTML = strings.Replace(backgroundErrorHTML, "ERROR-TIME", lastBackgroundError.Time.Format("2006-01-02 15:04:05"), -1)
		html = strings.Replace(html, "BACKGROUND-ERROR-HTML", backgroundErrorHTML, -1)
	} else {
		html = strings.Replace(html, "BACKGROUND-ERROR-HTML", "", -1)
	}

	encodeImageF := template.FuncMap{"encodeImage": func(bytes []byte) string {
		return base64.StdEncoding.EncodeToString(bytes)
	}}

	tmpl := template.Must(template.New("mangas").Funcs(encodeImageF).Parse(html))

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, mangas)
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
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
			if strings.Contains(err.Error(), "manga not found in DB") {
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
		if strings.Contains(err.Error(), "manga not found in DB") {
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
// @Description Updates a manga last read chapter in the database. If both `chapter` and `chapter_url` are empty strings in the body, set the last read chapter to the last upload chapter in the database. You must provide either the manga ID or the manga URL.
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

	mangaUpdate, err := manga.GetMangaDB(mangaID, mangaURL)
	if err != nil {
		if strings.Contains(err.Error(), "manga not found in DB") {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	var requestData UpdateMangaChapterRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	var chapter *manga.Chapter
	if requestData.Chapter == "" || requestData.ChapterURL == "" {
		chapter = mangaUpdate.LastUploadChapter
	} else {
		chapter, err = sources.GetChapterMetadata(mangaUpdate.URL, requestData.Chapter, requestData.ChapterURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}
	chapter.Type = 2
	chapter.UpdatedAt = currentTime

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
	Chapter    string `json:"chapter"`
	ChapterURL string `json:"chapter_url"`
}

// UpdateMangasMetadata updates the mangas metadata in the database
// It updates: the last upload chapter (and its metadata), the manga name and cover image

// @Summary Update mangas metadata
// @Description Get the mangas metadata from the sources and update them in the database.
// @Produce json
// @Param notify query string false "Notify if a new chapter was upload for the manga (only of mangas with status reading or completed)."
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

	for _, mangaToUpdate := range mangas {
		updatedManga, err := sources.GetMangaMetadata(mangaToUpdate.URL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		updatedManga.Status = 1

		if mangaToUpdate.LastUploadChapter.Chapter != updatedManga.LastUploadChapter.Chapter || mangaToUpdate.CoverImgURL != updatedManga.CoverImgURL || !bytes.Equal(mangaToUpdate.CoverImg, updatedManga.CoverImg) || mangaToUpdate.Name != updatedManga.Name {
			err = manga.UpdateMangaMetadataDB(updatedManga)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}

			// Notify only if the manga's status is 1 (reading) or 2 (completed)
			if notify && (mangaToUpdate.Status == 1 || mangaToUpdate.Status == 2) {
				if mangaToUpdate.LastUploadChapter.Chapter != updatedManga.LastUploadChapter.Chapter {
					err = NotifyMangaLastUploadChapterUpdate(mangaToUpdate, updatedManga)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf(`manga "%s" metadata updated, error while notifying: %s`, mangaToUpdate.URL, err.Error())})
						return
					}
				}
			}
		}

	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Mangas metadata updated successfully"})
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

// NotifyMangaLastUploadChapterUpdate notifies a manga last upload chapter update
func NotifyMangaLastUploadChapterUpdate(oldManga *manga.Manga, newManga *manga.Manga) error {
	publisher, err := notifications.GetNtfyPublisher()
	if err != nil {
		return err
	}

	chapterLink, err := url.Parse(newManga.LastUploadChapter.URL)
	if err != nil {
		return err
	}

	msg := &gotfy.Message{
		Topic:   publisher.Topic,
		Title:   fmt.Sprintf("New chapter of manga: %s", newManga.Name),
		Message: fmt.Sprintf("Last chapter: %s\nNew chapter: %s", oldManga.LastUploadChapter.Chapter, newManga.LastUploadChapter.Chapter),
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
