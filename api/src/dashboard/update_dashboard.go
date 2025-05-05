// Package dashboard contains the structs and functions that are used by routes that interact with the dashboard.
package dashboard

import (
	"sync"
	"time"
)

// whenUpdateDashboard is a struct that holds the last time a resource
// that should trigger a reload of the dashboard was updated.
// Used to update the iframe/dashboard when an event not
// triggered by the user occurs or the dashboard/iframe does something
// that should be reflected in the iframe/dashboard.
type whenUpdateDashboard struct {
	Time time.Time `json:"time"`
	mu   *sync.Mutex
}

var lastUpdate = whenUpdateDashboard{
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
