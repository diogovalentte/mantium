// Package routes implements the manga routes
package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AnthonyHewins/gotfy"
	"github.com/gin-gonic/gin"

	"github.com/diogovalentte/manga-dashboard-api/src/manga"
	"github.com/diogovalentte/manga-dashboard-api/src/sources"
	"github.com/diogovalentte/manga-dashboard-api/src/util"
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

	if requestData.LastReadChapterNumber != 0 || requestData.LastReadChapterURL != "" {
		mangaAdd.LastReadChapter, err = sources.GetChapterMetadata(requestData.URL, requestData.LastReadChapterNumber, requestData.LastReadChapterURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		mangaAdd.LastReadChapter.Type = 2
		mangaAdd.LastReadChapter.UpdatedAt = currentTime
	}

	_, err = mangaAdd.InsertDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Manga added successfully"})
}

// AddMangaRequest is the request body for the AddManga route
type AddMangaRequest struct {
	URL                   string       `json:"url" binding:"required,http_url"`
	Status                int          `json:"status" binding:"required,gte=0,lte=5"`
	LastReadChapterNumber manga.Number `json:"last_read_chapter_number"`
	LastReadChapterURL    string       `json:"last_read_chapter_url"`
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

// UpdateMangaLastReadChapter updates the manga last read chapter in the database
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

	chapter, err := sources.GetChapterMetadata(mangaUpdate.URL, requestData.ChapterNumber, requestData.ChapterURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	chapter.Type = 2
	chapter.UpdatedAt = currentTime

	err = mangaUpdate.UpdateChapter(chapter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Manga last read chapter updated successfully"})
}

// UpdateMangaChapterRequest is the request body for updating a manga chapter
type UpdateMangaChapterRequest struct {
	ChapterNumber manga.Number `json:"chapter_number" binding:"required"`
	ChapterURL    string       `json:"chapter_url"`
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

		if mangaToUpdate.LastUploadChapter.Number != updatedManga.LastUploadChapter.Number || mangaToUpdate.CoverImgURL != updatedManga.CoverImgURL || !bytes.Equal(mangaToUpdate.CoverImg, updatedManga.CoverImg) || mangaToUpdate.Name != updatedManga.Name {
			err = manga.UpdateMangaMetadataDB(updatedManga)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}

			if notify {
				if mangaToUpdate.LastUploadChapter.Number != updatedManga.LastUploadChapter.Number {
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
		Message: fmt.Sprintf("Last chapter: %.2f\nNew chapter: %.2f", oldManga.LastUploadChapter.Number, newManga.LastUploadChapter.Number),
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
