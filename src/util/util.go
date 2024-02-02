// Package util implements utility functions
package util

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

var logger *zerolog.Logger

// GetLogger returns the zerolog logger instance
func GetLogger() *zerolog.Logger {
	if logger == nil {
		logLevelStr := os.Getenv("LOG_LEVEL")
		logLevel, err := zerolog.ParseLevel(logLevelStr)
		if err != nil {
			logLevel = zerolog.InfoLevel
		}

		l := zerolog.New(os.Stdout).Level(logLevel).With().Timestamp().Logger()
		logger = &l
	}

	return logger
}

// AddErrorContext adds context to an error, like:
// "error downloading image: Get "https://example.com/image.jpg": dial tcp: lookup example.com: no such host".
// Should be used to add context to errors that are
// returned to the user, mostly in exported functions
// and methods
func AddErrorContext(err error, context string) error {
	return fmt.Errorf("%s: %w", context, err)
}

// RemoveLastOccurrence removes the last occurrence of a string from another string
func RemoveLastOccurrence(s, old string) string {
	if old == "" {
		return s
	}

	lastIndex := strings.LastIndex(s, old)
	modifiedString := s
	if lastIndex != -1 {
		modifiedString = s[:lastIndex] + s[lastIndex+len(old):]
	}

	return modifiedString
}

// GetImageFromURL downloads an image from a URL and returns the image bytes
func GetImageFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		err = fmt.Errorf("error downloading image: %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("failed to download image. Status code: %d", resp.StatusCode)
		return nil, err
	}

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf(`error reading image data at url "%s": %s`, url, err)
		return nil, err
	}

	return imageBytes, nil
}
