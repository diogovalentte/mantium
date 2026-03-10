package dashboard

import (
	"sync"
	"time"

	"github.com/diogovalentte/mantium/api/src/config"
)

// BackgroundError represents an error that occurred in the background.
type BackgroundError struct {
	// Error message.
	Message string `json:"message"`
	// Time when the error occurred.
	Time time.Time `json:"time"`
	// Number of consecutive errors. Used to avoid spamming the dashboard with the same error.
	ConsecutiveErrors int `json:"consecutiveErrors"`
	mu                *sync.Mutex
}

// lastBackgroundError is the last background error.
// Used to display the error in the dashboard.
var lastBackgroundError = BackgroundError{
	Message:           "",
	ConsecutiveErrors: 0,
	Time:              time.Time{},
	mu:                &sync.Mutex{},
}

// GetLastBackgroundError returns the last background error.
func GetLastBackgroundError() BackgroundError {
	lastBackgroundError.mu.Lock()
	defer lastBackgroundError.mu.Unlock()

	if lastBackgroundError.ConsecutiveErrors <= config.GlobalConfigs.PeriodicallyUpdateMangas.ConsecutiveErrors {
		return BackgroundError{}
	}

	return lastBackgroundError
}

// SetLastBackgroundError sets the last background error and updates the dashboard.
func SetLastBackgroundError(message string) {
	lastBackgroundError.mu.Lock()
	defer lastBackgroundError.mu.Unlock()
	lastBackgroundError.Message = message
	lastBackgroundError.Time = time.Now()
	lastBackgroundError.ConsecutiveErrors++
	UpdateDashboard()
}

// DeleteLastBackgroundError empties the last background error.
func DeleteLastBackgroundError() {
	lastBackgroundError.mu.Lock()
	defer lastBackgroundError.mu.Unlock()
	lastBackgroundError.Message = ""
	lastBackgroundError.Time = time.Time{}
	lastBackgroundError.ConsecutiveErrors = 0
	UpdateDashboard()
}

// ResetConsecutiveErrors resets the number of consecutive errors to 0. Used to avoid spamming the dashboard with the same error after a successful retry.
func ResetConsecutiveErrors() {
	lastBackgroundError.mu.Lock()
	defer lastBackgroundError.mu.Unlock()
	lastBackgroundError.ConsecutiveErrors = 0
}
