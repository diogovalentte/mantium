// Package dashboard contains the structs and functions that are used by routes that interact with the dashboard.
package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

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
		Columns                    int  `json:"columns"`
		ShowBackgroundErrorWarning bool `json:"showBackgroundErrorWarning"`
	} `json:"dashboard"`
}

// GetConfigsFromFile reads a file and unmarshal it into a Configs struct.
// Used to get the configurations from a JSON file.
func GetConfigsFromFile(filePath string) (*Configs, error) {
	jsonFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf("error reading configs from file '%s'", filePath), err)
	}

	var configs Configs
	err = json.Unmarshal(jsonFile, &configs)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf("error umarshaling configs from file '%s'", filePath), err)
	}

	return &configs, nil
}
