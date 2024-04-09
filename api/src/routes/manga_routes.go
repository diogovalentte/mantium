// Package routes implements the manga routes
package routes

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AnthonyHewins/gotfy"
	"github.com/gin-gonic/gin"

	"github.com/diogovalentte/manga-dashboard-api/api/src/manga"
	"github.com/diogovalentte/manga-dashboard-api/api/src/sources"
	"github.com/diogovalentte/manga-dashboard-api/api/src/util"
)

// MangaRoutes sets the manga routes
func MangaRoutes(group *gin.RouterGroup) {
	{
		group.POST("/manga", AddManga)
		group.GET("/manga", GetManga)
		group.DELETE("/manga", DeleteManga)
		group.GET("/manga/chapters", GetMangaChapters)
		group.PATCH("/manga/status", UpdateMangaStatus)
		group.PATCH("/manga/last_read_chapter", UpdateMangaLastReadChapter)
		group.GET("/mangas", GetMangas)
		group.GET("/mangas/iframe", GetMangasiFrame)
		group.PATCH("/mangas/metadata", UpdateMangasMetadata)
	}
}

// AddManga scrapes the manga page and inserts the manga data into the database
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

	_, err = mangaAdd.InsertDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Manga added successfully"})
}

// AddMangaRequest is the request body for the AddManga route
type AddMangaRequest struct {
	URL                string `json:"url" binding:"required,http_url"`
	Status             int    `json:"status" binding:"required,gte=0,lte=5"`
	LastReadChapter    string `json:"last_read_chapter"`
	LastReadChapterURL string `json:"last_read_chapter_url"`
}

// GetManga gets the manga from the database
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

// GetMangas gets mangas from the database
func GetMangas(c *gin.Context) {
	mangas, err := manga.GetMangasDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	resMap := map[string][]*manga.Manga{"mangas": mangas}
	c.JSON(http.StatusOK, resMap)
}

// GetMangasiFrame gets mangas from the database and returns a HTML code designed
// be used in an iFrame in Homarr
func GetMangasiFrame(c *gin.Context) {
	queryLimit := c.Query("limit")
	var limit int
	var err error
	if queryLimit == "" {
		limit = -1
	} else {
		limit, err = strconv.Atoi(queryLimit)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a number"})
		}
	}

	theme := c.Query("theme")
	if theme == "" {
		theme = "light"
	} else if theme != "dark" && theme != "light" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "theme must be 'dark' or 'light'"})
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

	html, err := getMangasiFrame(mangas, theme)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.Data(http.StatusOK, "text/html", []byte(html))
}

func getMangasiFrame(mangas []*manga.Manga, theme string) ([]byte, error) {
	html := `
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="referrer" content="no-referrer"> <!-- If not set, can't load Mangedex images when behind a domain or reverse proxy -->
    <title>Movie Display Template</title>
    <style>
      body {
        background-color: MANGAS-CONTAINER-BACKGROUND-COLOR;
        margin: 0;
        padding: 0;
      }

      .manga-container {
        width: calc(100% - MANGAS-CONTAINER-WIDTHpx);
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
        object-fit: cover;
        width: 30px;
        height: 50px;
      }

      .manga-details {
        flex: 1;
        padding: 0 20px;
      }

      .manga-name {
        font-size: 15px;
        font-weight: bold;
        color: white;

        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji";

        text-decoration: none;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .manga-name:hover {
        text-decoration: underline;
      }

      .new-chapter-container {
        display: inline-block;
        padding: 8px 10px;
        margin: 20px;
        background-color: rgb(109, 139, 150, 0.5);
        border-radius: 5px;
        width: 140px;
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
      function setLastReadChapter(ID, chapter) {
        var xhr = new XMLHttpRequest();
        var url = 'http://localhost:8080/v1/manga/last_read_chapter?id=' + encodeURIComponent(ID);
        xhr.open('PATCH', url, true);
        xhr.setRequestHeader('Content-Type', 'application/json');

        xhr.onload = function () {
          if (xhr.status >= 200 && xhr.status < 300) {
            console.log('Request to update manga', ID, ' last read chapter finished with success:', xhr.responseText);
            location.reload();
          } else {
            console.log('Request to update manga', ID, ' last read chapter failed:', xhr.responseText);
          }
        };

        xhr.onerror = function () {
          console.log('Request to update manga', ID, ' last read chapter failed:', xhr.responseText);
        };

        var body = {
            chapter: chapter
        };

        xhr.send(JSON.stringify(body));
      }
    </script>

  </head>
  <body>
    {{range .}}
        <div class="manga-container">

          <div style="background-image: url('{{ .CoverImgURL }}');" class="background-image"></div>

          <img
            class="manga-cover"
            src="{{ .CoverImgURL }}"
            alt="Manga Cover"
          />

          <div class="manga-details">
            <a href="{{ .URL }}" target="_blank" class="manga-name">{{ .Name }}</a>
          </div>

          <div class="new-chapter-container">
            <a href="{{ .LastReadChapter.URL }}" class="chapter-label last-read-chapter-label" target="_blank">{{ .LastReadChapter.Chapter }}</a>
            <span class="chapter-label chapter-gt-label"> &lt; </span>
            <a href="{{ .LastUploadChapter.URL }}" class="chapter-label last-upload-chapter-label" target="_blank">{{ .LastUploadChapter.Chapter }}</a>

            <button onclick="setLastReadChapter('{{ .ID }}', {{ .LastUploadChapter.Chapter }})" class="set-last-read-button" onmouseenter="this.style.cursor='pointer';">Set last read</button>
          </div>

        </div>
    {{end}}
  </body>
</html>
	`
	// Set the container width based on the number of mangas for better fitting with Homarr
	containerWidth := "1.6"
	if len(mangas) > 3 {
		containerWidth = "8"
	}

	// Homarr theme
	containerBackgroundColor := "#ffffff"
	scrollbarThumbBackgroundColor := "rgba(209, 219, 227, 1)"
	scrollbarTrackBackgroundColor := "#ffffff"
	if theme == "dark" {
		containerBackgroundColor = "#25262b"
		scrollbarThumbBackgroundColor = "#484d64"
		scrollbarTrackBackgroundColor = "rgba(37, 40, 53, 1)"
	}

	html = strings.Replace(html, "MANGAS-CONTAINER-WIDTH", containerWidth, -1)
	html = strings.Replace(html, "MANGAS-CONTAINER-BACKGROUND-COLOR", containerBackgroundColor, -1)
	html = strings.Replace(html, "SCROLLBAR-THUMB-BACKGROUND-COLOR", scrollbarThumbBackgroundColor, -1)
	html = strings.Replace(html, "SCROLLBAR-TRACK-BACKGROUND-COLOR", scrollbarTrackBackgroundColor, -1)

	tmpl := template.Must(template.New("mangas").Parse(html))

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, mangas)
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

// GetMangaChapters gets the manga chapters from the source
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

// DeleteManga delestes the from the database
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

	err = mangaDelete.DeleteDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Manga deleted successfully"})
}

// UpdateMangaStatus updates the manga status in the database
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

	err = mangaUpdate.UpdateStatus(requestData.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Manga status updated successfully"})
}

// UpdateMangaStatusRequest is the request body for the UpdateMangaStatus route
type UpdateMangaStatusRequest struct {
	Status manga.Status `json:"status" binding:"required,gte=0,lte=5"`
}

// UpdateMangaLastReadChapter updates the manga last read chapter.
// If not chapter number or URL is set on the body, set the last read
// chapter to the last upload chapter
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

	err = mangaUpdate.UpsertChapter(chapter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Manga last read chapter updated successfully"})
}

// UpdateMangaChapterRequest is the request body for updating a manga chapter
type UpdateMangaChapterRequest struct {
	Chapter    string `json:"chapter"`
	ChapterURL string `json:"chapter_url"`
}

// UpdateMangasMetadata updates the mangas metadata in the database
// It updates: the last upload chapter (and its metadata), the manga name and cover image
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
	publisher, err := util.GetNtfyPublisher()
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
