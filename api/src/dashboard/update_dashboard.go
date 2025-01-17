// Package dashboard contains the structs and functions that are used by routes that interact with the dashboard.
package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/util"
)

// WhenUpdateDashboard is a struct that holds the last time a resource
// that should trigger a reload of the dashboard was updated.
// Usually used to update the iframe/dashboard when an event not
// triggered by the user occurs or the dashboard/iframe does something
// that should be reflected in the iframe/dashboard.
type WhenUpdateDashboard struct {
	Time time.Time `json:"time"`
	mu   *sync.Mutex
}

var lastUpdate = WhenUpdateDashboard{
	Time: time.Now(),
	mu:   &sync.Mutex{},
}

// UpdateDashboard updates the last time a resource that should
// trigger a reload of the iframe/dashboard was updated.
func UpdateDashboard() {
	lastUpdate.mu.Lock()
	defer lastUpdate.mu.Unlock()

	lastUpdate.Time = time.Now()
}

// GetLastUpdateDashboard returns the last time a resource that should
// trigger a reload of the iframe/dashboard was updated.
func GetLastUpdateDashboard() time.Time {
	lastUpdate.mu.Lock()
	defer lastUpdate.mu.Unlock()

	return lastUpdate.Time
}

// Configs is a struct that holds the configuration of the dashboard.
// Usually something that need to be persisted when the application
// restart and can be updated.
type Configs struct {
	Dashboard struct {
		Columns                    int    `json:"columns"`
		ShowBackgroundErrorWarning bool   `json:"showBackgroundErrorWarning"`
		SearchResultsLimit         int    `json:"searchResultsLimit"`
		DisplayMode                string `json:"displayMode"`
	} `json:"dashboard"`
}

var ValidDisplayModeValues = []string{"Grid View", "List View"}

// Note: Also add default values to the SetDefaultConfigsFile function and the default file defaults/configs.json

// SetDefaultConfigsFile copies the default configs file
// to the default configs path if it doesn't exist.
// Also add default configs if they don't exist in the current file.
func SetDefaultConfigsFile() error {
	configsFilePath := config.GlobalConfigs.ConfigsFilePath
	if _, err := os.Stat(configsFilePath); os.IsNotExist(err) {
		err := copyDefaultConfigsFile(config.GlobalConfigs.DefaultConfigsFilePath, configsFilePath)
		if err != nil {
			return err
		}
	} else {
		var configs map[string]interface{}
		err := GetConfigsFromFile(&configs)
		if err != nil {
			return err
		}

		dashboardMap, ok := configs["dashboard"]
		if !ok {
			configs["dashboard"] = make(map[string]interface{})
			dashboardMap = configs["dashboard"]
		}
		dashboard, ok := dashboardMap.(map[string]interface{})
		if !ok {
			return fmt.Errorf("error while loading configs file")
		}
		_, ok = dashboard["columns"]
		if !ok {
			dashboard["columns"] = 5
		}
		_, ok = dashboard["showBackgroundErrorWarning"]
		if !ok {
			dashboard["showBackgroundErrorWarning"] = true
		}
		_, ok = dashboard["searchResultsLimit"]
		if !ok {
			dashboard["searchResultsLimit"] = 20
		}
		_, ok = dashboard["displayMode"]
		if !ok {
			dashboard["displayMode"] = "Grid View"
		}

		updatedConfigs, err := json.MarshalIndent(configs, "", "  ")
		if err != nil {
			return err
		}

		err = os.WriteFile(config.GlobalConfigs.ConfigsFilePath, updatedConfigs, 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyDefaultConfigsFile(srcPath, dstPath string) error {
	srcFile, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	err = os.WriteFile(dstPath, srcFile, 0o644)
	if err != nil {
		return err
	}

	return nil
}

// GetConfigsFromFile reads a file and unmarshal it into a Configs struct.
// Used to get the configurations from a JSON file.
func GetConfigsFromFile(target interface{}) error {
	jsonFile, err := os.ReadFile(config.GlobalConfigs.ConfigsFilePath)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf("error reading configs from file '%s'", config.GlobalConfigs.ConfigsFilePath), err)
	}

	err = json.Unmarshal(jsonFile, target)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf("error umarshaling configs from file '%s'", config.GlobalConfigs.ConfigsFilePath), err)
	}

	return nil
}
