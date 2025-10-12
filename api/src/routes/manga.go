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
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AnthonyHewins/gotfy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/dashboard"
	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/integrations/kaizoku"
	"github.com/diogovalentte/mantium/api/src/integrations/ntfy"
	"github.com/diogovalentte/mantium/api/src/integrations/suwayomi"
	"github.com/diogovalentte/mantium/api/src/integrations/tranga"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources"
	"github.com/diogovalentte/mantium/api/src/sources/models"
	"github.com/diogovalentte/mantium/api/src/util"
)

// MangaRoutes sets the manga routes
func MangaRoutes(group *gin.RouterGroup) {
	{
		// Methods for both normal manga and custom manga
		group.POST("/manga", AddManga)
		group.DELETE("/manga", DeleteManga)
		group.GET("/manga", GetManga)
		group.GET("/manga/metadata", GetMangaMetadata)
		group.GET("/manga/chapters", GetMangaChapters)
		group.PATCH("/manga/status", UpdateMangaStatus)
		group.PATCH("/manga/cover_img", UpdateMangaCoverImg)

		// Methods for custom manga only
		group.POST("/custom_manga", AddCustomManga)
		group.PATCH("/custom_manga/last_read_chapter", UpdateCustomMangaLastReadChapter)
		group.PATCH("/custom_manga/last_released_chapter_selectors", UpdateCustomMangaLastReleasedChapterSelectors)
		group.PATCH("/custom_manga/name", UpdateCustomMangaName)
		group.PATCH("/custom_manga/url", UpdateCustomMangaURL)

		// Methods for multimanga only
		group.POST("/multimanga", AddMultiManga)
		group.DELETE("/multimanga", DeleteMultiManga)
		group.GET("/multimanga", GetMultiManga)
		group.GET("/multimanga/choose_current_manga", ChooseCurrentManga)
		group.GET("/multimanga/chapters", GetMultiMangaChapters)
		group.PATCH("/multimanga/status", UpdateMultiMangaStatus)
		group.PATCH("/multimanga/last_read_chapter", UpdateMultiMangaLastReadChapter)
		group.PATCH("/multimanga/cover_img", UpdateMultiMangaCoverImg)
		group.POST("/multimanga/manga", AddMangaToMultiManga)
		group.DELETE("/multimanga/manga", RemoveMangaFromMultiManga)

		// Methods for manga library
		group.POST("/mangas/search", SearchManga)
		group.GET("/mangas", GetMangas)
		group.GET("/multimangas", GetMultiMangas)
		group.GET("/mangas/iframe", GetMangasiFrame)
		group.PATCH("/mangas/metadata", UpdateMangasMetadata)
		group.POST("/mangas/add_to_kaizoku", AddMangasToKaizoku)
		group.POST("/mangas/add_to_tranga", AddMangasToTranga)
		group.POST("/mangas/add_to_suwayomi", AddMangasToSuwayomi)
		group.GET("/mangas/stats", GetLibraryStats)
	}
}

// @Summary Add manga
// @Description Gets a manga metadata from source and inserts into the database.
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

	mangaAdd, err := sources.GetMangaMetadata(requestData.URL, requestData.MangaInternalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if !slices.Contains(config.GlobalConfigs.DashboardConfigs.Manga.AllowedSources, mangaAdd.Source) {
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("source %s is not allowed", mangaAdd.Source)})
		return
	}

	mangaAdd.Status = manga.Status(requestData.Status)

	if requestData.LastReadChapter != "" || requestData.LastReadChapterURL != "" {
		mangaAdd.LastReadChapter, err = sources.GetChapterMetadata(requestData.URL, requestData.MangaInternalID, requestData.LastReadChapter, requestData.LastReadChapterURL, requestData.LastReadChapterInternalID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		mangaAdd.LastReadChapter.Type = 2
		mangaAdd.LastReadChapter.UpdatedAt = currentTime.Truncate(time.Second)
	}

	if len(mangaAdd.CoverImg) == 0 {
		mangaAdd.CoverImg, err = util.GetDefaultCoverImg()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		mangaAdd.CoverImgResized = true
	}

	if mangaAdd.LastReleasedChapter != nil && mangaAdd.LastReleasedChapter.UpdatedAt.IsZero() {
		mangaAdd.LastReleasedChapter.UpdatedAt = currentTime.Truncate(time.Second)
	}

	err = mangaAdd.InsertIntoDB()
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

	if config.GlobalConfigs.Suwayomi.Valid {
		suwayomi := suwayomi.Suwayomi{}
		suwayomi.Init()

		err = suwayomi.AddManga(mangaAdd, config.GlobalConfigs.DashboardConfigs.Integrations.EnqueueAllSuwayomiChaptersToDownload)
		if err != nil {
			integrationsErrors = append(integrationsErrors, util.AddErrorContext("manga added to DB, but error while adding it to Suwayomi", err))
		}
	}

	if len(integrationsErrors) > 0 {
		fullMsg := "manga added to DB, but error executing integrations: "
		for _, err := range integrationsErrors {
			zerolog.Ctx(c.Request.Context()).Error().Err(err).Msg("error while adding manga to at least one integration")
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
	URL                       string `json:"url" binding:"required,http_url"`
	MangaInternalID           string `json:"manga_internal_id"`
	LastReadChapter           string `json:"last_read_chapter"`
	LastReadChapterURL        string `json:"last_read_chapter_url"`
	LastReadChapterInternalID string `json:"last_read_chapter_internal_id"`
	Status                    int    `json:"status" binding:"required,gte=0,lte=5"`
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

	if mangaGet.Source == manga.CustomMangaSource {
		if strings.HasPrefix(mangaGet.URL, manga.CustomMangaURLPrefix) {
			mangaGet.URL = ""
		}
		if mangaGet.LastReadChapter != nil && strings.HasPrefix(mangaGet.LastReadChapter.URL, manga.CustomMangaURLPrefix) {
			mangaGet.LastReadChapter.URL = ""
		}
		if mangaGet.LastReleasedChapter != nil && strings.HasPrefix(mangaGet.LastReleasedChapter.URL, manga.CustomMangaURLPrefix) {
			mangaGet.LastReadChapter.URL = ""
		}
	}

	resMap := map[string]manga.Manga{"manga": *mangaGet}
	c.JSON(http.StatusOK, resMap)
}

// @Summary Get manga metadata
// @Description Gets the metadata for a manga from the source site.
// @Produce json
// @Param url query string true "Manga URL" Example("https://mangadex.org/title/1/one-piece")
// @Success 200 {object} manga.Manga "{"manga": mangaObj}"
// @Router /manga/metadata [get]
func GetMangaMetadata(c *gin.Context) {
	mangaURL := c.Query("url")
	mangaGet, err := sources.GetMangaMetadata(mangaURL, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	resMap := map[string]manga.Manga{"manga": *mangaGet}
	c.JSON(http.StatusOK, resMap)
}

// @Summary Get manga chapters
// @Description Get a manga chapters from the source. You must provide either the manga ID or the manga URL.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga URL" Example("https://mangadex.org/title/1/one-piece")
// @Param manga_internal_id query string false "Manga Internal ID" Example("1as4fa7")
// @Success 200 {array} manga.Chapter "{"chapters": [chapterObj]}"
// @Router /manga/chapters [get]
func GetMangaChapters(c *gin.Context) {
	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	mangaInternalID := c.Query("manga_internal_id")
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

	chapters, err := sources.GetMangaChapters(mangaURL, mangaInternalID)
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

// @Summary Update custom manga name
// @Description Updates a custom manga name in the database. You must provide either the manga ID or the manga URL.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga current URL" Example("https://mangadex.org/title/1/one-piece")
// @Param name query string true "New manga name" Example("One Piece")
// @Success 200 {object} responseMessage
// @Router /custom_manga/name [patch]
func UpdateCustomMangaName(c *gin.Context) {
	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	newMangaName := c.Query("name")
	mangaID, mangaURL, err := getMangaIDAndURL(mangaIDStr, mangaURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if newMangaName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "you must provide a new manga name"})
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

	if mangaUpdate.Source != manga.CustomMangaSource {
		c.JSON(http.StatusBadRequest, gin.H{"message": "you can only update the name for custom mangas"})
		return
	}

	err = mangaUpdate.UpdateNameInDB(newMangaName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Custom manga name updated successfully"})
}

// @Summary Update custom manga URL
// @Description Updates a custom manga URL in the database. You must provide either the manga ID or the manga current URL.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga current URL" Example("https://mangadex.org/title/1/one-piece")
// @Param new_url query string true "Manga new URL " Example("https://mangadex.org/title/2/two-piece")
// @Success 200 {object} responseMessage
// @Router /manga/url [patch]
func UpdateCustomMangaURL(c *gin.Context) {
	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	newMangaURL := c.Query("new_url")
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

	if mangaUpdate.Source != manga.CustomMangaSource {
		c.JSON(http.StatusBadRequest, gin.H{"message": "you can only update the URL for custom mangas"})
		return
	}

	if newMangaURL == "" {
		newMangaURL = manga.CustomMangaURLPrefix + "/" + uuid.New().String()
	}

	err = mangaUpdate.UpdateCustomMangaURLInDB(newMangaURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Custom manga URL updated successfully"})
}

// @Summary Update custom manga last read chapter
// @Description Updates a custom manga last read chapter in the database. If chapter not in body, set the last read chapter to the last released chapter if it exists, else set it to none. If both `chapter` and `chapter_url` are empty strings in the body, deletes the manga's last read chapter. You can't provide only the chapter_url. You must provide either the manga ID or the manga URL.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga URL" Example("https://mangadex.org/title/1/one-piece")
// @Param chapter body UpdateMangaLastReadChapterRequest true "Chapter"
// @Success 200 {object} responseMessage
// @Router /custom_manga/last_read_chapter [patch]
func UpdateCustomMangaLastReadChapter(c *gin.Context) {
	currentTime := time.Now()

	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	mangaID, mangaURL, err := getMangaIDAndURL(mangaIDStr, mangaURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	var requestData UpdateMangaLastReadChapterRequest
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

	if mangaUpdate.Source != manga.CustomMangaSource {
		c.JSON(http.StatusBadRequest, gin.H{"message": "you can only update the last read chapter for custom mangas. Update the multimanga last read chapter instead"})
		return
	}

	if requestData.Chapter == nil {
		if mangaUpdate.LastReleasedChapter != nil {
			chapter := mangaUpdate.LastReleasedChapter
			chapter.Type = 2
			chapter.UpdatedAt = currentTime.Truncate(time.Second)

			err = mangaUpdate.UpsertChapterIntoDB(chapter)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
		} else {
			err = mangaUpdate.DeleteChaptersFromDB()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
		}
	} else {
		if requestData.Chapter.Chapter == "" && requestData.Chapter.URL != "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "you can't provide only the chapter.url. Set both chapter.chapter and chapter.url to empty strings to delete the last read chapter"})
			return
		}

		if requestData.Chapter.Chapter != "" {
			chapter := &manga.Chapter{
				Chapter:   requestData.Chapter.Chapter,
				Name:      "Chapter " + requestData.Chapter.Chapter,
				URL:       requestData.Chapter.URL,
				Type:      2,
				UpdatedAt: currentTime.Truncate(time.Second),
			}
			if chapter.URL == "" {
				chapter.URL = manga.CustomMangaURLPrefix + "/" + uuid.New().String()
			}
			err = mangaUpdate.UpsertChapterIntoDB(chapter)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
		} else {
			err = mangaUpdate.DeleteLastReadChapterFromDB()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
		}
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga last read chapter updated successfully"})
}

// @Summary Update manga cover image
// @Description Updates a manga/custom manga cover image in the database. You must provide either the manga ID or the manga URL. You must provide only one of the following: cover_img, cover_img_url, get_cover_img_from_source. If it's a custom manga, using get_cover_img_from_source will return an error message.
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga URL" Example("https://mangadex.org/title/1/one-piece")
// @Param manga_internal_id query string false "Manga Internal ID" Example("1as4fa7")
// @Param cover_img formData file false "Manga cover image file. Remember to set the Content-Type header to 'multipart/form-data' when sending the request."
// @Param cover_img_url query string false "Manga cover image URL" Example("https://example.com/cover.jpg")
// @Param get_cover_img_from_source query bool false "Let Mantium fetch the cover image from the source site" Example(true)
// @Param use_mantium_default_img query bool false "Update manga cover image to  Mantium's default cover image" Example(true)
// @Success 200 {object} responseMessage
// @Router /manga/cover_img [patch]
func UpdateMangaCoverImg(c *gin.Context) {
	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")
	mangaInternalID := c.Query("manga_internal_id")
	coverImgURL := c.Query("cover_img_url")
	getCoverImgFromSource := c.Query("get_cover_img_from_source")
	useMantiumDefaultImg := c.Query("use_mantium_default_img")

	var coverImg []byte
	var requestFile multipart.File
	requestFile, _, err := c.Request.FormFile("cover_img")
	if err != nil {
		if err != http.ErrMissingFile && err != http.ErrNotMultipart {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	} else {
		defer requestFile.Close()
		coverImg, err = io.ReadAll(requestFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}

	var score int
	if coverImgURL != "" {
		score++
	}
	if getCoverImgFromSource != "" {
		switch getCoverImgFromSource {
		case "true":
			score++
		case "false":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"message": "get_cover_img_from_source must be a boolean"})
			return
		}
	}
	if useMantiumDefaultImg != "" {
		switch useMantiumDefaultImg {
		case "true":
			score++
		case "false":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"message": "use_mantium_default_img must be a boolean"})
			return
		}
	}
	if len(coverImg) != 0 {
		score++
	}

	switch score {
	case 0:
		c.JSON(http.StatusBadRequest, gin.H{"message": "you must provide one of the following: cover_img, cover_img_url, get_cover_img_from_source, use_mantium_default_img"})
		return
	case 1:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"message": "you must provide only one of the following: cover_img, cover_img_url, get_cover_img_from_source, use_mantium_default_img"})
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
		resizedCoverImg, err := util.ResizeImage(coverImg, uint(util.DefaultImageWidth), uint(util.DefaultImageHeight))
		if err == nil {
			err = mangaToUpdate.UpdateCoverImgInDB(resizedCoverImg, true, coverImgURL)
		} else {
			err = mangaToUpdate.UpdateCoverImgInDB(coverImg, false, coverImgURL)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	} else if coverImgURL != "" {
		coverImg, isImgRezied, err := util.GetImageFromURL(coverImgURL, retries, retryInterval)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "invalid image: " + err.Error()})
			return
		}

		mangaToUpdate.CoverImgFixed = true

		err = mangaToUpdate.UpdateCoverImgInDB(coverImg, isImgRezied, coverImgURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	} else if getCoverImgFromSource != "" {
		if mangaToUpdate.Source == manga.CustomMangaSource {
			c.JSON(http.StatusBadRequest, gin.H{"message": "you can't get the cover image from the source for custom mangas"})
			return
		}

		var updatedManga *manga.Manga
		for i := 0; i < retries; i++ {
			updatedManga, err = sources.GetMangaMetadata(mangaToUpdate.URL, mangaInternalID)
			if err != nil {
				if i == retries-1 {
					c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
					return
				}
				time.Sleep(retryInterval)
				continue
			}
			if len(updatedManga.CoverImg) == 0 {
				continue
			}
		}

		if len(updatedManga.CoverImg) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "cover image not found in source"})
			return
		}

		mangaToUpdate.CoverImgFixed = false

		err = mangaToUpdate.UpdateCoverImgInDB(updatedManga.CoverImg, updatedManga.CoverImgResized, updatedManga.CoverImgURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	} else if useMantiumDefaultImg != "" {
		defaultCoverImg, err := util.GetDefaultCoverImg()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		mangaToUpdate.CoverImgFixed = true

		err = mangaToUpdate.UpdateCoverImgInDB(defaultCoverImg, true, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga cover image updated successfully"})
}

// @Summary Add custom manga
// @Description Inserts a custom manga into the database.
// @Accept json
// @Produce json
// @Param manga body AddCustomMangaRequest true "Manga data"
// @Success 200 {object} responseMessage
// @Router /custom_manga [post]
func AddCustomManga(c *gin.Context) {
	currentTime := time.Now()

	var requestData AddCustomMangaRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	status := manga.Status(requestData.Status)
	err := manga.ValidateStatus(status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	customManga := manga.Manga{
		Status:                                status,
		URL:                                   requestData.URL,
		Name:                                  requestData.Name,
		Source:                                manga.CustomMangaSource,
		LastReleasedChapterSelectorUseBrowser: requestData.LastReleasedChapterSelectorUseBrowser,
	}
	if requestData.LastReleasedChapterNameSelector != nil {
		customManga.LastReleasedChapterNameSelector = (*manga.HTMLSelector)(requestData.LastReleasedChapterNameSelector)
	}
	if requestData.LastReleasedChapterURLSelector != nil {
		customManga.LastReleasedChapterURLSelector = (*manga.HTMLSelector)(requestData.LastReleasedChapterURLSelector)
		customManga.LastReleasedChapterURLSelector.Regex = ""
	}

	if requestData.LastReadChapter != nil {
		customManga.LastReadChapter = &manga.Chapter{
			Chapter:   requestData.LastReadChapter.Chapter,
			Name:      "Chapter " + requestData.LastReadChapter.Chapter,
			URL:       requestData.LastReadChapter.URL,
			Type:      2,
			UpdatedAt: currentTime.Truncate(time.Second),
		}

		if requestData.LastReadChapter.URL == "" {
			customManga.LastReadChapter.URL = manga.CustomMangaURLPrefix + "/" + uuid.New().String()
		}
	}
	if customManga.URL != "" && (customManga.LastReleasedChapterNameSelector != nil || customManga.LastReleasedChapterURLSelector != nil) {
		chapter, err := manga.GetCustomMangaLastReleasedChapter(customManga.URL, customManga.LastReleasedChapterNameSelector, customManga.LastReleasedChapterURLSelector, requestData.LastReleasedChapterSelectorUseBrowser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "error getting last released chapter: " + err.Error()})
			return
		}
		customManga.LastReleasedChapter = chapter
	}

	if customManga.URL == "" {
		customManga.URL = manga.CustomMangaURLPrefix + "/" + uuid.New().String()
	}

	if len(requestData.CoverImg) > 0 && requestData.CoverImgURL != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "you must provide only one of the following: cover_img, cover_img_url"})
		return
	}
	if len(requestData.CoverImg) > 0 {
		if !util.IsImageValid(requestData.CoverImg) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid image"})
			return
		}
		resizedCoverImg, err := util.ResizeImage(requestData.CoverImg, uint(util.DefaultImageWidth), uint(util.DefaultImageHeight))
		if err == nil {
			customManga.CoverImg = resizedCoverImg
			customManga.CoverImgResized = true
		} else {
			customManga.CoverImg = requestData.CoverImg
			customManga.CoverImgResized = false
		}
	} else if requestData.CoverImgURL != "" {
		customManga.CoverImg, customManga.CoverImgResized, err = util.GetImageFromURL(requestData.CoverImgURL, 3, 3*time.Second)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "invalid image: " + err.Error()})
			return
		}

		customManga.CoverImgURL = requestData.CoverImgURL
	} else {
		customManga.CoverImg, err = util.GetDefaultCoverImg()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		customManga.CoverImgResized = true
	}

	err = customManga.InsertIntoDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga added successfully"})
}

// AddCustomMangaRequest is the request body for the AddCustomManga route
type AddCustomMangaRequest struct {
	LastReleasedChapterSelectorUseBrowser bool                 `json:"last_released_chapter_selector_use_browser"`
	Status                                int                  `json:"status" binding:"required,gte=0,lte=5"`
	Name                                  string               `json:"name" binding:"required"`
	URL                                   string               `json:"url" binding:"omitempty,http_url"`
	CoverImgURL                           string               `json:"cover_img_url" binding:"omitempty,http_url"`
	CoverImg                              []byte               `json:"cover_img"`
	LastReleasedChapterNameSelector       *HTMLSelectorRequest `json:"last_released_chapter_name_selector"`
	LastReleasedChapterURLSelector        *HTMLSelectorRequest `json:"last_released_chapter_url_selector"`
	LastReadChapter                       *struct {
		Chapter string `json:"chapter"`
		URL     string `json:"url" binding:"omitempty,http_url"`
	} `json:"last_read_chapter"`
}

type HTMLSelectorRequest struct {
	GetFirst  bool   `json:"get_first"`
	Selector  string `json:"selector" binding:"required"`
	Attribute string `json:"attribute"`
	Regex     string `json:"regex"`
}

// @Summary Update custom manga last released chapter selectors
// @Description Update custom manga last released chapter selectors.
// @Accept json
// @Produce json
// @Param id query int false "Manga ID" Example(1)
// @Param url query string false "Manga current URL" Example("https://mangadex.org/title/1/one-piece")
// @Param manga body UpdateLastReleasedChapterSelectorsRequest true "Selectors data"
// @Success 200 {object} responseMessage
// @Router /custom_manga/last_released_chapter_selectors [patch]
func UpdateCustomMangaLastReleasedChapterSelectors(c *gin.Context) {
	mangaIDStr := c.Query("id")
	mangaURL := c.Query("url")

	var requestData UpdateLastReleasedChapterSelectorsRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
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

	if mangaToUpdate.Source != manga.CustomMangaSource {
		c.JSON(http.StatusBadRequest, gin.H{"message": "you can only update the last released chapter selectors for custom mangas"})
		return
	}

	err = mangaToUpdate.UpdateLastReleasedChapterSelectorsInDB((*manga.HTMLSelector)(requestData.LastReleasedChapterNameSelector), (*manga.HTMLSelector)(requestData.LastReleasedChapterURLSelector), requestData.LastReleasedChapterSelectorUseBrowser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Custom manga updated successfully"})
}

type UpdateLastReleasedChapterSelectorsRequest struct {
	LastReleasedChapterNameSelector       *HTMLSelectorRequest `json:"name_selector"`
	LastReleasedChapterURLSelector        *HTMLSelectorRequest `json:"url_selector"`
	LastReleasedChapterSelectorUseBrowser bool                 `json:"use_browser"`
}

// @Summary Add multimanga
// @Description Gets a manga metadata from source and inserts it as the current manga of a new multimanga into the database.
// @Accept json
// @Produce json
// @Param manga body AddMangaRequest true "Current manga data"
// @Success 200 {object} responseMessage
// @Router /multimanga [post]
func AddMultiManga(c *gin.Context) {
	currentTime := time.Now()

	var requestData AddMangaRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	currentManga, err := sources.GetMangaMetadata(requestData.URL, requestData.MangaInternalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if !slices.Contains(config.GlobalConfigs.DashboardConfigs.Manga.AllowedSources, currentManga.Source) {
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("source %s is not allowed", currentManga.Source)})
		return
	}

	currentManga.Status = manga.Status(requestData.Status)

	if requestData.LastReadChapter != "" || requestData.LastReadChapterURL != "" {
		currentManga.LastReadChapter, err = sources.GetChapterMetadata(requestData.URL, requestData.MangaInternalID, requestData.LastReadChapter, requestData.LastReadChapterURL, requestData.LastReadChapterInternalID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		currentManga.LastReadChapter.Type = 2
		currentManga.LastReadChapter.UpdatedAt = currentTime.Truncate(time.Second)
	}

	if len(currentManga.CoverImg) == 0 {
		currentManga.CoverImg, err = util.GetDefaultCoverImg()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		currentManga.CoverImgURL = models.DefaultCoverImgURL
		currentManga.CoverImgResized = true
	}

	if currentManga.LastReleasedChapter != nil && currentManga.LastReleasedChapter.UpdatedAt.IsZero() {
		currentManga.LastReleasedChapter.UpdatedAt = currentTime.Truncate(time.Second)
	}

	multiManga := &manga.MultiManga{
		CurrentManga:    currentManga,
		LastReadChapter: currentManga.LastReadChapter,
		Mangas:          []*manga.Manga{currentManga},
		Status:          currentManga.Status,
	}

	err = multiManga.InsertIntoDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	var integrationsErrors []error
	if config.GlobalConfigs.Kaizoku.Valid {
		kaizoku := kaizoku.Kaizoku{}
		kaizoku.Init()
		err = kaizoku.AddManga(currentManga, config.GlobalConfigs.Kaizoku.TryOtherSources)
		if err != nil {
			integrationsErrors = append(integrationsErrors, util.AddErrorContext("multimanga added to DB, but error while adding current manga to Kaizoku", err))
		}
	}

	if config.GlobalConfigs.Tranga.Valid {
		tranga := tranga.Tranga{}
		tranga.Init()
		err = tranga.AddManga(currentManga)
		if err != nil {
			integrationsErrors = append(integrationsErrors, util.AddErrorContext("multimanga added to DB, but error while adding current manga to Tranga", err))
		}
	}

	if config.GlobalConfigs.Suwayomi.Valid {
		suwayomi := suwayomi.Suwayomi{}
		suwayomi.Init()

		err = suwayomi.AddManga(currentManga, config.GlobalConfigs.DashboardConfigs.Integrations.EnqueueAllSuwayomiChaptersToDownload)
		if err != nil {
			integrationsErrors = append(integrationsErrors, util.AddErrorContext("multimanga added to DB, but error while adding current manga to Suwayomi", err))
		}
	}

	if len(integrationsErrors) > 0 {
		fullMsg := "multimang added to DB, but error executing integrations: "
		for _, err := range integrationsErrors {
			zerolog.Ctx(c.Request.Context()).Error().Err(err).Msg("error while adding multimanga's current manga to at least one integration")
			fullMsg += err.Error() + " "
		}
		dashboard.UpdateDashboard()
		c.JSON(http.StatusInternalServerError, gin.H{"message": fullMsg})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Multimanga added successfully"})
}

// @Summary Delete multimanga
// @Description Deletes a multimanga from the database.
// @Produce json
// @Param id query int true "Multimanga ID" Example(1)
// @Success 200 {object} responseMessage
// @Router /multimanga [delete]
func DeleteMultiManga(c *gin.Context) {
	multimangaIDStr := c.Query("id")
	if multimangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be provided"})
		return
	}
	multimangaID, err := strconv.Atoi(multimangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be a number"})
		return
	}

	multimangaDelete, err := manga.GetMultiMangaFromDB(manga.ID(multimangaID))
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMultiMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	err = multimangaDelete.DeleteFromDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Multimanga deleted successfully"})
}

// @Summary Get multimanga
// @Description Gets a multimanga from the database.
// @Produce json
// @Param id query int true "Multimanga ID" Example(1)
// @Success 200 {object} manga.MultiManga "{"multimanga": multimangaObj}"
// @Router /multimanga [get]
func GetMultiManga(c *gin.Context) {
	multimangaIDStr := c.Query("id")
	if multimangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be provided"})
		return
	}
	multimangaID, err := strconv.Atoi(multimangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be a number"})
		return
	}

	multimangaGet, err := manga.GetMultiMangaFromDB(manga.ID(multimangaID))
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	resMap := map[string]manga.MultiManga{"multimanga": *multimangaGet}
	c.JSON(http.StatusOK, resMap)
}

// @Summary Choose current manga
// @Description Check a multimanga mangas and returns which manga should be the current manga.
// @Produce json
// @Param id query int true "Multimanga ID" Example(1)
// @Param exclude_manga_ids query string false "Manga IDs to exclude from the check" Example("1,2,3")
// @Success 200 {object} manga.Manga "{"manga": mangaObj}"
// @Router /multimanga/choose_current_manga [get]
func ChooseCurrentManga(c *gin.Context) {
	multimangaIDStr := c.Query("id")
	if multimangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be provided"})
		return
	}
	multimangaID, err := strconv.Atoi(multimangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be a number"})
		return
	}

	multimanga, err := manga.GetMultiMangaFromDB(manga.ID(multimangaID))
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	queryExcludeMangaIDs := c.Query("exclude_manga_ids")
	var excludeMangaIDs []int
	if queryExcludeMangaIDs != "" {
		excludeMangaIDsStr := strings.Split(queryExcludeMangaIDs, ",")
		if len(excludeMangaIDsStr) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "exclude_manga_ids must be a comma separated list of numbers"})
			return
		}
		for _, excludeMangaIDStr := range excludeMangaIDsStr {
			excludeMangaID, err := strconv.Atoi(excludeMangaIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "exclude_manga_ids must be a comma separated list of numbers"})
				return
			}
			excludeMangaIDs = append(excludeMangaIDs, excludeMangaID)
		}
	}

	for _, excludeMangaID := range excludeMangaIDs {
		found := false
		for i, m := range multimanga.Mangas {
			if m.ID == manga.ID(excludeMangaID) {
				multimanga.Mangas = append(multimanga.Mangas[:i], multimanga.Mangas[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusBadRequest, gin.H{"message": "exclude_manga_ids must be a list of multimanga manga IDs"})
			return
		}
	}

	if len(multimanga.Mangas) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "multimanga must have at least one manga after excluding the manga IDs"})
		return
	}

	returnManga, err := manga.GetLatestManga(multimanga.Mangas)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	resMap := map[string]*manga.Manga{"manga": returnManga}
	c.JSON(http.StatusOK, resMap)
}

// @Summary Get multimanga current manga chapters
// @Description Get chapters of the current manga of a multimanga from the source.
// @Produce json
// @Param id query int true "Multimanga ID" Example(1)
// @Success 200 {array} manga.Chapter "{"chapters": [chapterObj]}"
// @Router /multimanga/chapters [get]
func GetMultiMangaChapters(c *gin.Context) {
	multimangaIDStr := c.Query("id")
	if multimangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be provided"})
		return
	}
	multimangaID, err := strconv.Atoi(multimangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be a number"})
		return
	}

	multimanga, err := manga.GetMultiMangaFromDB(manga.ID(multimangaID))
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMultiMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	chapters, err := sources.GetMangaChapters(multimanga.CurrentManga.URL, multimanga.CurrentManga.InternalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	resMap := map[string][]*manga.Chapter{"chapters": chapters}
	c.JSON(http.StatusOK, resMap)
}

// @Summary Update multimanga status
// @Description Updates a multimanga status in the database.
// @Produce json
// @Param id query int true "Multimanga ID" Example(1)
// @Param status body UpdateMangaStatusRequest true "Multimanga status"
// @Success 200 {object} responseMessage
// @Router /multimanga/status [patch]
func UpdateMultiMangaStatus(c *gin.Context) {
	multimangaIDStr := c.Query("id")
	if multimangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be provided"})
		return
	}
	multimangaID, err := strconv.Atoi(multimangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be a number"})
		return
	}

	multimanga, err := manga.GetMultiMangaFromDB(manga.ID(multimangaID))
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMultiMangaNotFoundDB.Error()) {
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

	err = multimanga.UpdateStatusInDB(requestData.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Multimanga status updated successfully"})
}

// UpdateMangaStatusRequest is the request body for the UpdateMangaStatus route
type UpdateMangaStatusRequest struct {
	Status manga.Status `json:"status" binding:"required,gte=0,lte=5"`
}

// @Summary Update multimanga last read chapter
// @Description Updates a multimanga last read chapter in the database. It also needs to know from which manga the chapter is from. If both `chapter` and `chapter_url` are empty strings in the body, set the last read chapter to the last released chapter in the database.
// @Produce json
// @Param id query int true "Multimanga ID" Example(1)
// @Param manga_id query int true "Manga ID" Example(1)
// @Param chapter body UpdateMangaLastReadChapterRequest true "Chapter"
// @Success 200 {object} responseMessage
// @Router /multimanga/last_read_chapter [patch]
func UpdateMultiMangaLastReadChapter(c *gin.Context) {
	currentTime := time.Now()

	multimangaIDStr := c.Query("id")
	if multimangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be provided"})
		return
	}
	multimangaID, err := strconv.Atoi(multimangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be a number"})
		return
	}

	mangaIDStr := c.Query("manga_id")
	if mangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "manga_id must be provided"})
		return
	}
	mangaID, err := strconv.Atoi(mangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "manga_id must be a number"})
		return
	}

	multimanga, err := manga.GetMultiMangaFromDB(manga.ID(multimangaID))
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMultiMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	var mangaGetChapterFrom *manga.Manga
	for _, m := range multimanga.Mangas {
		if m.ID == manga.ID(mangaID) {
			mangaGetChapterFrom = m
			break
		}
	}
	if mangaGetChapterFrom == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": errordefs.ErrMangaNotFoundInMultiManga.Error()})
		return
	}

	var requestData UpdateMangaLastReadChapterRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	var chapter *manga.Chapter
	if requestData.Chapter == nil || (requestData.Chapter.Chapter == "" && requestData.Chapter.URL == "") {
		chapter = mangaGetChapterFrom.LastReleasedChapter
	} else {
		chapter, err = sources.GetChapterMetadata(mangaGetChapterFrom.URL, mangaGetChapterFrom.InternalID, requestData.Chapter.Chapter, requestData.Chapter.URL, requestData.Chapter.InternalID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}
	chapter.Type = 2
	chapter.UpdatedAt = currentTime.Truncate(time.Second)

	err = multimanga.UpsertChapterIntoDB(chapter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Multimanga last read chapter updated successfully"})
}

// UpdateMangaLastReadChapterRequest is the request body for updating a manga chapter
type UpdateMangaLastReadChapterRequest struct {
	Chapter *LastReadChapterRequest `json:"chapter"`
}

type LastReadChapterRequest struct {
	Chapter    string `json:"chapter,omitempty"`
	URL        string `json:"url,omitempty" binding:"omitempty,http_url"`
	InternalID string `json:"internal_id,omitempty"`
}

// @Summary Update multimanga cover image
// @Description Updates a multimanga cover image in the database. You must provide only one of the following: cover_img, cover_img_url, use_current_manga_cover_img.
// @Produce json
// @Param id query int true "Multimanga ID" Example(1)
// @Param cover_img formData file false "Multimanga cover image file. Remember to set the Content-Type header to 'multipart/form-data' when sending the request."
// @Param cover_img_url query string false "Multimanga cover image URL" Example("https://example.com/cover.jpg")
// @Param use_current_manga_cover_img query bool false "Use the multimanga's current manga cover image" Example(true)
// @Success 200 {object} responseMessage
// @Router /multimanga/cover_img [patch]
func UpdateMultiMangaCoverImg(c *gin.Context) {
	coverImgURL := c.Query("cover_img_url")
	useCurrentMangaCoverImg := c.Query("use_current_manga_cover_img")
	multimangaIDStr := c.Query("id")
	if multimangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be provided"})
		return
	}
	multimangaID, err := strconv.Atoi(multimangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be a number"})
		return
	}

	var coverImg []byte
	var requestFile multipart.File
	requestFile, _, err = c.Request.FormFile("cover_img")
	if err != nil {
		if err != http.ErrMissingFile && err != http.ErrNotMultipart {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	} else {
		defer requestFile.Close()
		coverImg, err = io.ReadAll(requestFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}

	var score int
	if coverImgURL != "" {
		score++
	}
	if useCurrentMangaCoverImg != "" {
		switch useCurrentMangaCoverImg {
		case "true":
			score++
		case "false":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"message": "use_current_manga_cover_img must be a boolean"})
			return
		}
	}
	if len(coverImg) != 0 {
		score++
	}

	switch score {
	case 0:
		c.JSON(http.StatusBadRequest, gin.H{"message": "you must provide one of the following: cover_img, cover_img_url, use_current_manga_cover_img"})
		return
	case 1:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"message": "you must provide only one of the following: cover_img, cover_img_url, use_current_manga_cover_img"})
		return
	}

	multimanga, err := manga.GetMultiMangaFromDB(manga.ID(multimangaID))
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMultiMangaNotFoundDB.Error()) {
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

		multimanga.CoverImgFixed = true
		resizedCoverImg, err := util.ResizeImage(coverImg, uint(util.DefaultImageWidth), uint(util.DefaultImageHeight))
		if err == nil {
			err = multimanga.UpdateCoverImgInDB(resizedCoverImg, true, coverImgURL)
		} else {
			err = multimanga.UpdateCoverImgInDB(coverImg, false, coverImgURL)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	} else if coverImgURL != "" {
		coverImg, isImgRezied, err := util.GetImageFromURL(coverImgURL, retries, retryInterval)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "invalid image: " + err.Error()})
			return
		}

		multimanga.CoverImgFixed = true
		err = multimanga.UpdateCoverImgInDB(coverImg, isImgRezied, coverImgURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	} else if useCurrentMangaCoverImg != "" {
		multimanga.CoverImgFixed = false
		err = multimanga.UpdateCoverImgInDB([]byte{}, false, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Multimanga cover image updated successfully"})
}

// @Summary Add manga to multimanga list
// @Description Adds a manga to a multimanga list in the database.
// @Accept json
// @Produce json
// @Param id query int true "Multimanga ID" Example(1)
// @Param manga body AddMangaToMultiMangaRequest true "Manga data"
// @Success 200 {object} responseMessage
// @Router /multimanga/manga [post]
func AddMangaToMultiManga(c *gin.Context) {
	currentTime := time.Now()

	multimangaIDStr := c.Query("id")
	if multimangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be provided"})
		return
	}
	multimangaID, err := strconv.Atoi(multimangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be a number"})
		return
	}

	var requestData AddMangaToMultiMangaRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	mangaAdd, err := sources.GetMangaMetadata(requestData.MangaURL, requestData.MangaInternalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if !slices.Contains(config.GlobalConfigs.DashboardConfigs.Manga.AllowedSources, mangaAdd.Source) {
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("source %s is not allowed", mangaAdd.Source)})
		return
	}

	if len(mangaAdd.CoverImg) == 0 {
		mangaAdd.CoverImg, err = util.GetDefaultCoverImg()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		mangaAdd.CoverImgResized = true
	}

	if mangaAdd.LastReleasedChapter != nil && mangaAdd.LastReleasedChapter.UpdatedAt.IsZero() {
		mangaAdd.LastReleasedChapter.UpdatedAt = currentTime.Truncate(time.Second)
	}

	multimanga, err := manga.GetMultiMangaFromDB(manga.ID(multimangaID))
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMultiMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	mangaAdd.Status = multimanga.Status // Only to make the manga valid to insert into DB, not really used
	err = multimanga.AddManga(mangaAdd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	if config.GlobalConfigs.DashboardConfigs.Integrations.AddAllMultiMangaMangasToDownloadIntegrations {
		var integrationsErrors []error
		if config.GlobalConfigs.Kaizoku.Valid {
			kaizoku := kaizoku.Kaizoku{}
			kaizoku.Init()
			err = kaizoku.AddManga(mangaAdd, config.GlobalConfigs.Kaizoku.TryOtherSources)
			if err != nil {
				integrationsErrors = append(integrationsErrors, util.AddErrorContext("manga added to multimanga, but error while adding current manga to Kaizoku", err))
			}
		}

		if config.GlobalConfigs.Tranga.Valid {
			tranga := tranga.Tranga{}
			tranga.Init()
			err = tranga.AddManga(mangaAdd)
			if err != nil {
				integrationsErrors = append(integrationsErrors, util.AddErrorContext("manga added to multimanga, but error while adding current manga to Tranga", err))
			}
		}

		if config.GlobalConfigs.Suwayomi.Valid {
			suwayomi := suwayomi.Suwayomi{}
			suwayomi.Init()

			err = suwayomi.AddManga(mangaAdd, config.GlobalConfigs.DashboardConfigs.Integrations.EnqueueAllSuwayomiChaptersToDownload)
			if err != nil {
				integrationsErrors = append(integrationsErrors, util.AddErrorContext("manga added to multimanga, but error while adding it to Suwayomi", err))
			}
		}

		if len(integrationsErrors) > 0 {
			fullMsg := "manga added to multimanga, but error executing integrations: "
			for _, err := range integrationsErrors {
				zerolog.Ctx(c.Request.Context()).Error().Err(err).Msg("error while adding manga to at least one integration")
				fullMsg += err.Error() + " "
			}
			dashboard.UpdateDashboard()
			c.JSON(http.StatusInternalServerError, gin.H{"message": fullMsg})
			return
		}
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga added to multimanga successfully"})
}

// AddMangaToMultiMangaRequest is the request body for the AddManga route
type AddMangaToMultiMangaRequest struct {
	MangaURL        string `json:"manga_url" binding:"required,http_url"`
	MangaInternalID string `json:"manga_internal_id"`
}

// @Summary Remove manga from multimanga list
// @Description Removes a manga from a multimanga list in the database.
// @Accept json
// @Produce json
// @Param id query int true "Multimanga ID" Example(1)
// @Param manga_id query int true "Manga ID" Example(1)
// @Success 200 {object} responseMessage
// @Router /multimanga/manga [delete]
func RemoveMangaFromMultiManga(c *gin.Context) {
	multimangaIDStr := c.Query("id")
	if multimangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be provided"})
		return
	}
	multimangaID, err := strconv.Atoi(multimangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be a number"})
		return
	}

	mangaIDStr := c.Query("manga_id")
	if mangaIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "manga_id must be provided"})
		return
	}
	mangaID, err := strconv.Atoi(mangaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "manga_id must be a number"})
		return
	}

	multimanga, err := manga.GetMultiMangaFromDB(manga.ID(multimangaID))
	if err != nil {
		if strings.Contains(err.Error(), errordefs.ErrMultiMangaNotFoundDB.Error()) {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	var mangaToRemove *manga.Manga
	for _, m := range multimanga.Mangas {
		if m.ID == manga.ID(mangaID) {
			mangaToRemove = m
			break
		}
	}
	if mangaToRemove == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": errordefs.ErrMangaNotFoundInMultiManga.Error()})
		return
	}

	err = multimanga.RemoveManga(mangaToRemove)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	dashboard.UpdateDashboard()

	c.JSON(http.StatusOK, gin.H{"message": "Manga removed from multimanga successfully"})
}

// @Summary Search manga
// @Description Searches a manga in the source. You must provide the source name like "mangadex" and the search query.
// @Accept json
// @Produce json
// @Param search body SearchMangaRequest true "Search data"
// @Success 200 {object} map[string][]models.MangaSearchResult "{"mangas": [mangaSearchResultObj]}"
// @Router /mangas/search [post]
func SearchManga(c *gin.Context) {
	var requestData SearchMangaRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	if requestData.Limit == 0 {
		requestData.Limit = 20
	}
	mangas, err := sources.SearchManga(requestData.Term, requestData.Source, requestData.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if len(mangas) > 0 && !slices.Contains(config.GlobalConfigs.DashboardConfigs.Manga.AllowedSources, mangas[0].Source) {
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("source %s is not allowed", mangas[0].Source)})
		return
	}

	resMap := map[string][]*models.MangaSearchResult{"mangas": mangas}
	c.JSON(http.StatusOK, resMap)
}

// SearchMangaRequest is the request body for the SearchManga route
type SearchMangaRequest struct {
	Source string `json:"source" binding:"required"`
	Term   string `json:"q" binding:"required"`
	Limit  int    `json:"limit"`
}

// @Summary Get mangas
// @Description Gets the current manga of multimangas and all custom mangas.
// @Produce json
// @Success 200 {array} manga.Manga "{"mangas": [mangaObj]}"
// @Router /mangas [get]
func GetMangas(c *gin.Context) {
	mangas, err := manga.GetCustomMangasDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	for _, m := range mangas {
		if strings.HasPrefix(m.URL, manga.CustomMangaURLPrefix) {
			m.URL = ""
		}
		if m.LastReadChapter != nil && strings.HasPrefix(m.LastReadChapter.URL, manga.CustomMangaURLPrefix) {
			m.LastReadChapter.URL = ""
		}
	}

	multimangas, err := manga.GetMultiMangasDB(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	for _, multimanga := range multimangas {
		multimanga.CurrentManga.LastReadChapter = multimanga.LastReadChapter
		multimanga.CurrentManga.Status = multimanga.Status
		if multimanga.CoverImgFixed {
			multimanga.CurrentManga.CoverImg = multimanga.CoverImg
			multimanga.CurrentManga.CoverImgURL = multimanga.CoverImgURL
			multimanga.CurrentManga.CoverImgResized = multimanga.CoverImgResized
			multimanga.CurrentManga.CoverImgFixed = true
		}
		mangas = append(mangas, multimanga.CurrentManga)
	}

	resMap := map[string][]*manga.Manga{"mangas": mangas}
	c.JSON(http.StatusOK, resMap)
}

// @Summary Get multimangas
// @Description Gets all multimangas. The multimanga's mangas will have only the current manga. The current manga will have a possible wrong status, so use the multimanga's status.
// @Produce json
// @Success 200 {array} manga.MultiManga "{"multimangas": [multimangaObj]}"
// @Router /multimangas [get]
func GetMultiMangas(c *gin.Context) {
	multimangas, err := manga.GetMultiMangasDB(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	resMap := map[string][]*manga.MultiManga{"multimangas": multimangas}
	c.JSON(http.StatusOK, resMap)
}

// @Summary Mangas iFrame
// @Description Returns an iFrame with mangas. Only mangas with unread chapters, and status reading or completed. Sort by last released chapter date.
// @Success 200 {string} string "HTML content"
// @Produce html
// @Param api_url query string true "API URL used by your browser. Used for the button that updates the last read chater, as your browser needs to send a request to the API to update the chapter." Example(https://sub.domain.com)
// @Param theme query string false "IFrame theme, defaults to light. If it's different from your dashboard theme, the background turns may turn white" Example(light)
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

	allMangas, err := manga.GetCustomMangasDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	multimangas, err := manga.GetMultiMangasDB(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	for _, multimanga := range multimangas {
		multimanga.CurrentManga.LastReadChapter = multimanga.LastReadChapter
		multimanga.CurrentManga.Status = multimanga.Status
		if multimanga.CoverImgFixed {
			multimanga.CurrentManga.CoverImg = multimanga.CoverImg
			multimanga.CurrentManga.CoverImgURL = multimanga.CoverImgURL
			multimanga.CurrentManga.CoverImgResized = multimanga.CoverImgResized
			multimanga.CurrentManga.CoverImgFixed = true
		}
		allMangas = append(allMangas, multimanga.CurrentManga)
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
        }

        .mangas-container {
            height: 84px;

            position: relative;
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin: 8.50px;

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
      function setCustomMangaLastReadChapter(mangaId) {
        try {
            var xhr = new XMLHttpRequest();
            var url = '{{ .APIURL }}/v1/custom_manga/last_read_chapter?id=' + encodeURIComponent(mangaId);
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

      function setMultiMangaLastReadChapter(multimangaId, mangaId) {
        try {
            var xhr = new XMLHttpRequest();
            var url = '{{ .APIURL }}/v1/multimanga/last_read_chapter?id=' + encodeURIComponent(multimangaId) + '&manga_id=' + encodeURIComponent(mangaId);
            xhr.open('PATCH', url, true);
            xhr.setRequestHeader('Content-Type', 'application/json');

            xhr.onload = function () {
              if (xhr.status >= 200 && xhr.status < 300) {
                console.log('Request to update multimanga', multimangaId, ' last read chapter finished with success:', xhr.responseText);
                location.reload();
              } else {
                console.log('Request to update multimanga', multimangaId, ' last read chapter failed:', xhr.responseText);
                handleSetLastReadChapterError("manga-" + mangaId)
              }
            };

            xhr.onerror = function () {
              console.log('Request to update multimanga', multimangaId, ' last read chapter failed:', xhr.responseText);
              handleSetLastReadChapterError(mangaId)
            };

            var body = {};

            xhr.send(JSON.stringify(body));
        } catch (error) {
            console.log('Request to update multimanga', multimangaId, ' last read chapter failed:', error);
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
			{{ if isCustomMangaURL .URL }}
				<span class="manga-name">{{ .Name }}</span>
			{{ else }}
				<a href="{{ .URL }}" target="_blank" class="manga-name">{{ .Name }}</a>
			{{ end }}
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
				<button id="manga-{{ .ID }}" onclick="{{ if isCustomManga .Source }}setCustomMangaLastReadChapter('{{ .ID }}'){{ else }}setMultiMangaLastReadChapter('{{ .MultiMangaID }}', '{{ .ID }}'){{ end }}" class="set-last-read-button" onmouseenter="this.style.cursor='pointer';">Set last read</button>
			</div>
        </div>

    </div>
{{end}}
</body>
</html>
	`
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

	encodeImageF := template.FuncMap{
		"encodeImage": func(bytes []byte) string {
			return base64.StdEncoding.EncodeToString(bytes)
		},
		"isCustomManga": func(source string) bool {
			return source == manga.CustomMangaSource
		},
		"isCustomMangaURL": func(url string) bool {
			return strings.HasPrefix(url, manga.CustomMangaURLPrefix)
		},
	}

	tmpl := template.Must(template.New("mangas").Funcs(encodeImageF).Parse(html))

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, templateData)
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

type iframeTemplateData struct {
	BackgroundErrorTime           time.Time
	Theme                         string
	APIURL                        string
	ScrollbarThumbBackgroundColor string
	ScrollbarTrackBackgroundColor string
	Mangas                        []*manga.Manga
	ShowBackgroundError           bool
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

	var mangasWithNewChapter []*manga.Manga

	logger := util.GetLogger(zerolog.Level(config.GlobalConfigs.API.LogLevelInt))
	errors := map[string][]string{
		"manga_metadata": {},
		"ntfy":           {},
		"tranga":         {},
		"kaizoku":        {},
		"suwayomi":       {},
	}
	var newMetadata bool
	var trangaInt *tranga.Tranga
	if config.GlobalConfigs.Tranga.Valid {
		trangaInt = &tranga.Tranga{}
		trangaInt.Init()
	}
	var suwayomiInt *suwayomi.Suwayomi
	if config.GlobalConfigs.Suwayomi.Valid {
		suwayomiInt = &suwayomi.Suwayomi{}
		suwayomiInt.Init()
	}
	retries := 3
	retryInterval := 3 * time.Second

	mangas, err := manga.GetCustomMangasDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	multimangas, err := manga.GetMultiMangasDB(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	type result struct {
		mangaWithNewChapters *manga.Manga
		multimangaErrors     []string
	}

	results := make(chan result, len(multimangas))
	var wg sync.WaitGroup

	// Custom Mangas
	chunkSize := (len(mangas) + config.GlobalConfigs.PeriodicallyUpdateMangas.ParallelJobs - 1) / config.GlobalConfigs.PeriodicallyUpdateMangas.ParallelJobs
	for i := 0; i < config.GlobalConfigs.PeriodicallyUpdateMangas.ParallelJobs; i++ {
		start := i * chunkSize
		end := start + chunkSize
		end = min(end, len(mangas))
		chunk := mangas[start:end]

		wg.Add(1)
		go func(chunk []*manga.Manga) {
			defer wg.Done()
			for _, mangaToUpdate := range chunk {
				mangaWithNewChapters, multimangaErrors := updateCustomMangaMetadata(mangaToUpdate, retries, retryInterval, logger)
				if mangaWithNewChapters != nil {
					newMetadata = true
				}
				result := result{
					mangaWithNewChapters: mangaWithNewChapters,
					multimangaErrors:     multimangaErrors,
				}
				results <- result
			}
		}(chunk)
	}

	// MultiMangas
	chunkSize = (len(multimangas) + config.GlobalConfigs.PeriodicallyUpdateMangas.ParallelJobs - 1) / config.GlobalConfigs.PeriodicallyUpdateMangas.ParallelJobs
	for i := 0; i < config.GlobalConfigs.PeriodicallyUpdateMangas.ParallelJobs; i++ {
		start := i * chunkSize
		end := start + chunkSize
		end = min(end, len(multimangas))
		chunk := multimangas[start:end]

		wg.Add(1)
		go func(chunk []*manga.MultiManga) {
			defer wg.Done()
			for _, multimangaToUpdate := range chunk {
				mangaWithNewChapters, multimangaNewMetadata, multimangaErrors := updateMultiMangaMetadata(multimangaToUpdate, retries, retryInterval, logger)
				if multimangaNewMetadata {
					newMetadata = true
				}
				result := result{
					mangaWithNewChapters: mangaWithNewChapters,
					multimangaErrors:     multimangaErrors,
				}
				results <- result
			}
		}(chunk)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		if res.mangaWithNewChapters != nil {
			mangasWithNewChapter = append(mangasWithNewChapter, res.mangaWithNewChapters)
		}
		if len(res.multimangaErrors) > 0 {
			errors["manga_metadata"] = append(errors["manga_metadata"], res.multimangaErrors...)
		}
	}

	if newMetadata {
		dashboard.UpdateDashboard()
	}

	for _, m := range mangasWithNewChapter {
		// Notify only if the manga's status is 1 (reading) or 2 (completed)
		if notify && (m.Status == 1 || m.Status == 2) {
			for j := range retries {
				err = NotifyMangaLastReleasedChapterUpdate(m)
				if err != nil {
					if j == retries-1 {
						logger.Error().Err(err).Str("manga_url", m.URL).Msg(fmt.Sprintf("Manga metadata updated in DB, but error while notifying: %s.\nWill continue with the next manga...", err.Error()))
						errors["ntfy"] = append(errors["ntfy"], err.Error())
						break
					}
					logger.Error().Err(err).Str("manga_url", m.URL).Msgf("Manga metadata updated in DB, but error while notifying: %s.\nRetrying in %.2f seconds...", err.Error(), retryInterval.Seconds())
					continue
				}
				break
			}
		}

		if m.Source == manga.CustomMangaSource {
			continue
		}

		if trangaInt != nil {
			err = trangaInt.StartJob(m)
			if err != nil {
				logger.Error().Err(err).Str("manga_url", m.URL).Msg("Manga metadata updated in DB, but error starting job in Tranga.\nWill continue with the next manga...")
				errors["tranga"] = append(errors["tranga"], err.Error())
			}

		}

		if suwayomiInt != nil {
			mangaID, err := suwayomiInt.GetLibraryMangaID(m)
			if err != nil {
				logger.Error().Err(err).Str("manga_url", m.URL).Msg("Manga metadata updated in DB, but error getting manga ID from Suwayomi.\nWill continue with the next manga...")
				errors["suwayomi"] = append(errors["suwayomi"], err.Error())
			} else {
				chapter, err := suwayomiInt.GetChapter(mangaID, m.LastReleasedChapter.URL)
				if err != nil {
					logger.Error().Err(err).Str("manga_url", m.URL).Str("suwayomi_manga_id", strconv.Itoa(mangaID)).Msg("Manga metadata updated in DB, but error getting chapter from Suwayomi.\nWill continue with the next manga...")
					errors["suwayomi"] = append(errors["suwayomi"], err.Error())
				} else {
					err = suwayomiInt.EnqueueChapterDownloads([]int{chapter.ID})
					if err != nil {
						logger.Error().Err(err).Str("manga_url", m.URL).Str("suwayomi_chapter_id", strconv.Itoa(chapter.ID)).Msg("Manga metadata updated in DB, but error updating chapter in Suwayomi.\nWill continue with the next manga...")
						errors["suwayomi"] = append(errors["suwayomi"], err.Error())
					}
				}
			}
		}
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
// @Description Add the multimangas' current manga to Kaizoku. If it fails to add a manga, it will continue with the next manga. This is a heavy operation depending on the number of mangas in the database.
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

	mangas := []*manga.Manga{}
	multimangas, err := manga.GetMultiMangasDB(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	for _, multimanga := range multimangas {
		multimanga.CurrentManga.Status = multimanga.Status
		mangas = append(mangas, multimanga.CurrentManga)
	}

	kaizoku := kaizoku.Kaizoku{}
	kaizoku.Init()
	logger := util.GetLogger(zerolog.Level(config.GlobalConfigs.API.LogLevelInt))
	var lastError error
	for _, dbManga := range mangas {
		if dbManga.Source == manga.CustomMangaSource {
			continue
		}

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
// @Description Add the multimangas' current manga to Tranga. If it fails to add a manga, it will continue with the next manga. This is a heavy operation depending on the number of mangas in the database. Currently, only MangaDex mangas can be added to Tranga, but it'll try all mangas anyway.
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

	mangas := []*manga.Manga{}
	multimangas, err := manga.GetMultiMangasDB(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	for _, multimanga := range multimangas {
		multimanga.CurrentManga.Status = multimanga.Status
		mangas = append(mangas, multimanga.CurrentManga)
	}

	trangaInt := tranga.Tranga{}
	trangaInt.Init()
	logger := util.GetLogger(zerolog.Level(config.GlobalConfigs.API.LogLevelInt))
	var errorSlice []string
	for _, dbManga := range mangas {
		if dbManga.Source == manga.CustomMangaSource {
			continue
		}

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

// @Summary Add mangas to Suwayomi
// @Description Add the multimangas' current manga to Suwayomi. If it fails to add a manga, it will continue with the next manga. This is a heavy operation depending on the number of mangas in the database.
// @Produce json
// @Param status query []int false "Filter which mangas to add by status. 1=reading, 2=completed, 3=on hold, 4=dropped, 5=plan to read. Example: status=1,2,3,5" Example(1,2,3,5)
// @Success 200 {object} responseMessage
// @Router /mangas/add_to_suwayomi [post]
func AddMangasToSuwayomi(c *gin.Context) {
	if !config.GlobalConfigs.Suwayomi.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"message": "suwayomi is not configured in the API"})
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

	mangas := []*manga.Manga{}
	multimangas, err := manga.GetMultiMangasDB(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	for _, multimanga := range multimangas {
		multimanga.CurrentManga.Status = multimanga.Status
		mangas = append(mangas, multimanga.CurrentManga)
	}

	suwayomi := suwayomi.Suwayomi{}
	suwayomi.Init()
	logger := util.GetLogger(zerolog.Level(config.GlobalConfigs.API.LogLevelInt))
	var lastError error

	for _, dbManga := range mangas {
		if dbManga.Source == manga.CustomMangaSource {
			continue
		}

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
		err = suwayomi.AddManga(dbManga, config.GlobalConfigs.DashboardConfigs.Integrations.EnqueueAllSuwayomiChaptersToDownload)
		if err != nil {
			logger.Error().Err(err).Str("manga_url", dbManga.URL).Msg("error adding manga to Suwayomi, will continue with the next manga...")
			lastError = err
			continue
		}
	}

	if lastError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "some errors occured while adding some mangas to Suwayomi, check the logs for more information. Last error: " + lastError.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mangas added to Suwayomi successfully"})
}

// @Summary Get library stats
// @Description Get the library stats from all multimangas and custom mangas.
// @Produce json
// @Success 200 {map} map[string]int "{"property": value}"
// @Router /mangas/stats [get]
func GetLibraryStats(c *gin.Context) {
	stats, err := manga.GetLibraryStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
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
			return -1, "", util.AddErrorContext("invalid manga url", err)
		}
	}

	return mangaID, mangaURL, nil
}

// NotifyMangaLastReleasedChapterUpdate notifies a manga last released chapter update
func NotifyMangaLastReleasedChapterUpdate(m *manga.Manga) error {
	publisher, err := ntfy.GetNtfyPublisher()
	if err != nil {
		return err
	}

	title := fmt.Sprintf("(Mantium) New chapter of manga: %s", m.Name)

	message := fmt.Sprintf("New chapter: %s", m.LastReleasedChapter.Chapter)

	chapterLink, err := url.Parse(m.LastReleasedChapter.URL)
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

func isNewChapterDifferentFromOld(oldChapter, newChapter *manga.Chapter, source string) bool {
	if oldChapter == nil && newChapter != nil {
		return true
	}
	if oldChapter != nil && newChapter != nil && oldChapter.Chapter != newChapter.Chapter {
		return true
	}
	if source == manga.CustomMangaSource {
		if oldChapter != nil && newChapter != nil && oldChapter.URL != newChapter.URL {
			return true
		}
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

// updateMultiMangaMetadata gets the manga metadata from the sources for all the multimanga' mangas and updates it in the database.
// Returns the updated current manga if the current manga has a new released chapter, else nil.
// Also returns a bool indicating if any metadata was updated and a slice of errors.
func updateMultiMangaMetadata(multimanga *manga.MultiManga, retries int, retryInterval time.Duration, logger *zerolog.Logger) (*manga.Manga, bool, []string) {
	var err error
	var errors []string
	var newMetadata bool
	var mangasHaveNewChapter bool

	for _, mangaToUpdate := range multimanga.Mangas {
		var updatedManga *manga.Manga
		for i := 0; i < retries; i++ {
			updatedManga, err = sources.GetMangaMetadata(mangaToUpdate.URL, mangaToUpdate.InternalID)
			if err != nil {
				if i != retries-1 {
					logger.Error().Err(err).Str("manga_url", mangaToUpdate.URL).Msgf("Error getting manga metadata, retrying in %.2f seconds...", retryInterval.Seconds())
					time.Sleep(retryInterval)
					continue
				}
				break
			}
			if len(updatedManga.CoverImg) == 0 {
				continue
			}
			break
		}
		if updatedManga == nil {
			logger.Error().Err(err).Str("manga_url", mangaToUpdate.URL).Msg("Error getting manga metadata, will continue with the next manga...")
			errors = append(errors, err.Error())
			continue
		}

		// Turn manga into valid manga to update DB
		if len(updatedManga.CoverImg) == 0 {
			updatedManga.CoverImg, err = util.GetDefaultCoverImg()
			if err != nil {
				logger.Error().Err(err).Str("manga_url", mangaToUpdate.URL).Msg("Error getting default cover image, will continue with the next manga...")
				errors = append(errors, err.Error())
				continue
			}
			updatedManga.CoverImgResized = true
		}
		updatedManga.Status = multimanga.Status
		updatedManga.ID = mangaToUpdate.ID

		mangaHasNewReleasedChapter := isNewChapterDifferentFromOld(mangaToUpdate.LastReleasedChapter, updatedManga.LastReleasedChapter, mangaToUpdate.Source)
		if mangaHasNewReleasedChapter || (!mangaToUpdate.CoverImgFixed && (mangaToUpdate.CoverImgURL != updatedManga.CoverImgURL || !bytes.Equal(mangaToUpdate.CoverImg, updatedManga.CoverImg))) || mangaToUpdate.Name != updatedManga.Name {
			if mangaHasNewReleasedChapter {
				mangasHaveNewChapter = true
			}
			if updatedManga.LastReleasedChapter != nil && updatedManga.LastReleasedChapter.UpdatedAt.IsZero() {
				if !mangaHasNewReleasedChapter && mangaToUpdate.LastReleasedChapter != nil {
					updatedManga.LastReleasedChapter.UpdatedAt = mangaToUpdate.LastReleasedChapter.UpdatedAt
				} else {
					updatedManga.LastReleasedChapter.UpdatedAt = time.Now().Truncate(time.Second)
				}
			}
			err = manga.UpdateMangaMetadataDB(updatedManga)
			if err != nil {
				logger.Error().Err(err).Str("manga_url", mangaToUpdate.URL).Msg("Error saving manga new metadata to DB, will continue with the next manga...")
				errors = append(errors, err.Error())
				continue
			}
			newMetadata = true
		}
	}
	if mangasHaveNewChapter {
		updatedMultimanga, err := manga.GetMultiMangaFromDB(multimanga.ID)
		if err != nil {
			logger.Error().Err(err).Str("multimanga_id", multimanga.ID.String()).Msg("Error getting multimanga from DB")
			errors = append(errors, err.Error())
			return nil, newMetadata, errors
		}
		err = updatedMultimanga.UpdateCurrentMangaInDB()
		if err != nil {
			logger.Error().Err(err).Str("multimanga_id", multimanga.ID.String()).Msg("Error updating multimanga current manga in DB")
			errors = append(errors, err.Error())
			return nil, newMetadata, errors
		}
		if updatedMultimanga.CurrentManga.LastReleasedChapter != nil {
			if multimanga.CurrentManga.LastReleasedChapter == nil {
				return updatedMultimanga.CurrentManga, newMetadata, errors
			} else if updatedMultimanga.CurrentManga.LastReleasedChapter.Chapter != multimanga.CurrentManga.LastReleasedChapter.Chapter {
				return updatedMultimanga.CurrentManga, newMetadata, errors
			}
		}
	}

	return nil, newMetadata, errors
}

// updateCustomMangaMetadata gets the custom manga last released chapter metadata and updates it in the database.
func updateCustomMangaMetadata(m *manga.Manga, retries int, retryInterval time.Duration, logger *zerolog.Logger) (*manga.Manga, []string) {
	var err error
	var errors []string
	var chapter *manga.Chapter

	if strings.HasPrefix(m.URL, manga.CustomMangaURLPrefix) || (m.LastReleasedChapterNameSelector == nil && m.LastReleasedChapterURLSelector == nil) {
		return nil, errors
	}

	for i := 0; i < retries; i++ {
		chapter, err = manga.GetCustomMangaLastReleasedChapter(m.URL, m.LastReleasedChapterNameSelector, m.LastReleasedChapterURLSelector, m.LastReleasedChapterSelectorUseBrowser)
		if err != nil {
			if i != retries-1 {
				nameSelector, URLSelector := &manga.HTMLSelector{}, &manga.HTMLSelector{}
				if m.LastReleasedChapterNameSelector != nil {
					nameSelector = m.LastReleasedChapterNameSelector
				}
				if m.LastReleasedChapterURLSelector != nil {
					URLSelector = m.LastReleasedChapterURLSelector
				}
				logger.Error().Err(err).Str("manga_url", m.URL).Str("name_selector", nameSelector.String()).Str("url_selector", URLSelector.String()).Msgf("Error getting custom manga last released chapter metadata, retrying in %.2f seconds...", retryInterval.Seconds())
				time.Sleep(retryInterval)
				continue
			}
			break
		}
		break
	}
	if chapter == nil {
		if m.LastReleasedChapterNameSelector == nil {
			m.LastReleasedChapterNameSelector = &manga.HTMLSelector{}
		}
		if m.LastReleasedChapterURLSelector == nil {
			m.LastReleasedChapterURLSelector = &manga.HTMLSelector{}
		}
		logger.Error().Err(err).Str("manga_url", m.URL).Str("name_selector", m.LastReleasedChapterNameSelector.String()).Str("url_selector", m.LastReleasedChapterURLSelector.String()).Msg("Error getting custom manga last released chapter metadata")
		errors = append(errors, err.Error())
		return nil, errors
	}

	mangaHasNewReleasedChapter := isNewChapterDifferentFromOld(m.LastReleasedChapter, chapter, m.Source)
	if mangaHasNewReleasedChapter {
		err = m.UpsertChapterIntoDB(chapter)
		if err != nil {
			if m.LastReleasedChapterNameSelector == nil {
				m.LastReleasedChapterNameSelector = &manga.HTMLSelector{}
			}
			if m.LastReleasedChapterURLSelector == nil {
				m.LastReleasedChapterURLSelector = &manga.HTMLSelector{}
			}
			logger.Error().Err(err).Str("manga_url", m.URL).Str("name_selector", m.LastReleasedChapterNameSelector.String()).Str("url_selector", m.LastReleasedChapterURLSelector.String()).Msg("Error saving custom manga new last released chapter to DB")
			errors = append(errors, err.Error())
			return nil, errors
		}
		return m, errors
	}

	return nil, errors
}
