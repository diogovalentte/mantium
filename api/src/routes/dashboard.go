package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

var configsFilePath = "./configs/configs.json" // relative to main.go

func DashboardRoutes(group *gin.RouterGroup) {
	{
		group.GET("/dashboard/configs", GetDashboardConfigs)
		group.PATCH("/dashboard/configs/columns", UpdateDashboardColumns)
	}
}

// @Summary Get the dashboard configs
// @Description Returns the dashboard configs read from the configs.json file.
// @Success 200 {object} Configs
// @Produce json
// @Router /dashboard/configs [get]
func GetDashboardConfigs(c *gin.Context) {
	configs, err := getConfigsFromFile(configsFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error while loading configs file: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"configs": configs.Dashboard})
}

// @Summary Update dashboard columns
// @Description Update the dashboard columns in the configs.json file.
// @Success 200 {object} responseMessage
// @Produce json
// @Param columns query int true "New number of columns." Example(5)
// @Router /dashboard/configs/columns [patch]
func UpdateDashboardColumns(c *gin.Context) {
	columnsStr := c.Query("columns")
	if columnsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing 'columns' parameter"})
		return
	}
	columns, err := strconv.Atoi(columnsStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to convert 'columns' parameter into number"})
		return
	}

	configs, err := getConfigsFromFile(configsFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error while loading configs file: %s", err.Error())})
		return
	}

	configs.Dashboard.Columns = columns

	updatedConfigs, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error while updating configs file: %s", err.Error())})
		return
	}

	err = os.WriteFile(configsFilePath, updatedConfigs, 0o644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("error while updating configs file: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configs updated successfully"})
}

func getConfigsFromFile(filePath string) (*Configs, error) {
	jsonFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var configs Configs
	err = json.Unmarshal(jsonFile, &configs)
	if err != nil {
		return nil, err
	}

	return &configs, nil
}

type Configs struct {
	Dashboard struct {
		Columns int `json:"columns"`
	} `json:"dashboard"`
}
