package dashboard

import (
	"sync"
	"time"
)

// BackgroundError represents an error that occurred in the background.
type BackgroundError struct {
	// Error message.
	Message string `json:"message"`
	// Time when the error occurred.
	Time time.Time `json:"time"`
	mu   *sync.Mutex
}

// lastBackgroundError is the last background error.
// Used to display the error in the dashboard.
var lastBackgroundError = BackgroundError{
	Message: "",
	Time:    time.Time{},
	mu:      &sync.Mutex{},
}

// GetLastBackgroundError returns the last background error.
func GetLastBackgroundError() BackgroundError {
	lastBackgroundError.mu.Lock()
	defer lastBackgroundError.mu.Unlock()
	return lastBackgroundError
}

// SetLastBackgroundError sets the last background error and updates the dashboard.
func SetLastBackgroundError(message string) {
	lastBackgroundError.mu.Lock()
	defer lastBackgroundError.mu.Unlock()
	lastBackgroundError.Message = message
	lastBackgroundError.Time = time.Now()
	UpdateDashboard()
}

// DeleteLastBackgroundError empties the last background error.
func DeleteLastBackgroundError() {
	lastBackgroundError.mu.Lock()
	defer lastBackgroundError.mu.Unlock()
	lastBackgroundError.Message = ""
	lastBackgroundError.Time = time.Time{}
	UpdateDashboard()
}
