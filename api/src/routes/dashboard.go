package routes

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"

	"github.com/mendoncart/mantium/api/src/config"
	"github.com/mendoncart/mantium/api/src/dashboard"
)

// DashboardRoutes sets the routes for the dashboard.
func DashboardRoutes(group *gin.RouterGroup) {
	{
		group.GET("/dashboard/configs", GetDashboardConfigs)
		group.POST("/dashboard/configs", UpdateDashboardConfigs)
		group.GET("/dashboard/last_update", GetLastUpdate)
		group.GET("/dashboard/last_background_error", GetLastBackgroundError)
		group.DELETE("/dashboard/last_background_error", DeleteLastBackgroundError)
		group.GET("/dashboard/updated_message", GetUpdatedMessage)
	}
}

// @Summary Get the dashboard configs
// @Description Returns the dashboard configs
// @Success 200 {object} config.DashboardConfigs
// @Produce json
// @Router /dashboard/configs [get]
func GetDashboardConfigs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"configs": config.GlobalConfigs.DashboardConfigs})
}

// @Summary Update dashboard configs
// @Description Update the dashboard configs in the DB. Cannot update version.
// @Success 200 {object} responseMessage
// @Accept json
// @Produce json
// @Param configs body config.DashboardConfigs true "Dashboard configs"
// @Router /dashboard/configs [post]
func UpdateDashboardConfigs(c *gin.Context) {
	var newConfigs config.DashboardConfigs
	err := c.ShouldBindJSON(&newConfigs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("error while binding configs: %s", err.Error())})
		return
	}

	if newConfigs.Display.Columns < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "columns must be greater than 0"})
		return
	}

	if newConfigs.Display.SearchResultsLimit < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "searchResultsLimit must be greater than 0"})
		return
	}

	if !slices.Contains(config.ValidDisplayModeValues, newConfigs.Display.DisplayMode) {
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("displayMode must be one of the following values: %v", config.ValidDisplayModeValues)})
		return
	}

	err = config.SaveConfigsToDB(&newConfigs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error while saving configs: %s", err.Error())})
		return
	}

	err = config.LoadConfigsFromDB(config.GlobalConfigs.DashboardConfigs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("configs saved to DB, but error while loading new configs from DB: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configs updated successfully"})
}

// @Summary Get the last update date
// @Description Returns the last time a resource that should trigger an update in the iframe/dashboard was updated. Usually used to update the dashboard when an event not triggered by the user occurs.
// @Success 200 {object} responseMessage
// @Produce json
// @Router /dashboard/last_update [get]
func GetLastUpdate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": dashboard.GetLastUpdateDashboard(),
	})
}

// @Summary Get the last background error
// @Description Returns the last error that happened in the background. Usually used to display the error in the dashboard.
// @Success 200 {object} dashboard.BackgroundError
// @Produce json
// @Router /dashboard/last_background_error [get]
func GetLastBackgroundError(c *gin.Context) {
	lastError := dashboard.GetLastBackgroundError()
	c.JSON(http.StatusOK, gin.H{"message": lastError.Message, "time": lastError.Time.Format("2006-01-02 15:04:05")})
}

// @Summary Delete the last background error
// @Description Deletes the last error that happened in the background. Usually used to clear the error in the dashboard.
// @Success 200 {object} responseMessage
// @Produce json
// @Router /dashboard/last_background_error [delete]
func DeleteLastBackgroundError(c *gin.Context) {
	dashboard.DeleteLastBackgroundError()
	c.JSON(http.StatusOK, gin.H{"message": "Last background error deleted"})
}

// @Summary Get's the updated message for this version
// @Description Get's the updated message for this version and deletes so it won't be shown again. Returns the message and the updated version.
// @Success 200 {object} responseMessage
// @Produce json
// @Router /dashboard/updated_message [get]
func GetUpdatedMessage(c *gin.Context) {
	message := dashboard.UpdatedMessageToShow
	version := dashboard.UpdatedMessageVersion
	dashboard.UpdatedMessageToShow = ""
	dashboard.UpdatedMessageVersion = ""
	c.JSON(http.StatusOK, gin.H{"message": message, "version": version})
}
