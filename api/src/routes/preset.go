package routes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
)

// @Summary Get custom manga presets
// @Description Gets all custom manga presets, the ones that come with Mantium first.
// @Produce json
// @Success 200 {array} manga.Preset "{"presets": [presetObj]}"
// @Router /custom_manga/presets [get]
func GetCustomMangaPresets(c *gin.Context) {
	presets, err := manga.GetPresetsDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"presets": presets})
}

// AddCustomMangaPresetRequest is the request body for the AddCustomMangaPreset route
type AddCustomMangaPresetRequest struct {
	NameSelector *HTMLSelectorRequest `json:"name_selector"`
	URLSelector  *HTMLSelectorRequest `json:"url_selector"`
	Name         string               `json:"name" binding:"required"`
	Description  string               `json:"description"`
	Domains      []string             `json:"domains"`
	UseBrowser   bool                 `json:"use_browser"`
}

// @Summary Add custom manga preset
// @Description Saves custom manga selectors as a preset. If the error is caused by the request, the response has a "user_message" field explaining it.
// @Accept json
// @Produce json
// @Param preset body AddCustomMangaPresetRequest true "Preset"
// @Success 200 {object} manga.Preset "{"preset": presetObj}"
// @Router /custom_manga/presets [post]
func AddCustomMangaPreset(c *gin.Context) {
	var requestData AddCustomMangaPresetRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	preset := &manga.Preset{
		Name:         requestData.Name,
		Description:  requestData.Description,
		Domains:      requestData.Domains,
		UseBrowser:   requestData.UseBrowser,
		NameSelector: (*manga.HTMLSelector)(requestData.NameSelector),
		URLSelector:  (*manga.HTMLSelector)(requestData.URLSelector),
	}
	if err := manga.InsertPresetDB(preset); err != nil {
		c.JSON(userErrorStatus(err), errorResponse(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"preset": preset})
}

// @Summary Delete custom manga preset
// @Description Deletes a preset created by the user. Presets that come with Mantium can't be deleted.
// @Produce json
// @Param id query int true "Preset ID"
// @Success 200 {object} responseMessage
// @Router /custom_manga/presets [delete]
func DeleteCustomMangaPreset(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be a number"})
		return
	}

	if err := manga.DeletePresetDB(id); err != nil {
		c.JSON(userErrorStatus(err), errorResponse(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preset deleted successfully"})
}

// DetectCustomMangaPresetsRequest is the request body for the DetectCustomMangaPresets route
type DetectCustomMangaPresetsRequest struct {
	URL string `json:"url" binding:"required,http_url"`
}

// @Summary Detect custom manga presets
// @Description Tells which presets work for the manga page. "domain_presets" are the presets whose domains match the URL, which don't need the page to be fetched. "detections" has the result of applying every preset to the page.
// @Accept json
// @Produce json
// @Param request body DetectCustomMangaPresetsRequest true "Manga page URL"
// @Success 200 {object} manga.PresetDetection "{"domain_presets": [presetObj], "detections": [presetDetectionObj]}"
// @Router /custom_manga/presets/detect [post]
func DetectCustomMangaPresets(c *gin.Context) {
	var requestData DetectCustomMangaPresetsRequest
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON fields, refer to the API documentation"})
		return
	}

	presets, err := manga.GetPresetsDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	detections, err := manga.DetectPresets(requestData.URL, presets)
	if err != nil {
		c.JSON(userErrorStatus(err), errorResponse(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"domain_presets": manga.PresetsForURL(requestData.URL, presets), "detections": detections})
}

func userErrorStatus(err error) int {
	var userErr *errordefs.UserError
	if errors.As(err, &userErr) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
