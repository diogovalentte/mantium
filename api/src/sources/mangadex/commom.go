package mangadex

import (
	"fmt"
	"strings"
	"time"
)

type genericRelationship struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
}

type localisedStrings map[string]string

type tag struct {
	ID            string                `json:"id"`
	Type          string                `json:"type"`
	Attributes    tagAttributes         `json:"attributes"`
	Relationships []genericRelationship `json:"relationships"`
}

type tagAttributes struct {
	Name        localisedStrings `json:"name"`
	Description localisedStrings `json:"description"`
	Group       string           `json:"group"`
	Version     int              `json:"version"`
}

// ErrorResponse is typical response for errored requests.
type ErrorResponse struct {
	Result string  `json:"result"`
	Errors []Error `json:"errors"`
}

// GetResult get the result for this particular request.
func (er *ErrorResponse) GetResult() string {
	return er.Result
}

// GetErrors get the errors for this particular request.
func (er *ErrorResponse) GetErrors() string {
	var errors strings.Builder
	for _, err := range er.Errors {
		errors.WriteString(fmt.Sprintf("%s: %s\n", err.Title, err.Detail))
	}
	return errors.String()
}

// Error contains details of an error.
type Error struct {
	ID string `json:"id"`

	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func getDatetime(date string) (time.Time, error) {
	parsedDate, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return time.Time{}, err
	}
	parsedDate = parsedDate.In(time.UTC)

	return parsedDate, err
}
