package mangadex

import (
	"fmt"
	"strings"
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

type mangaAttributes struct {
	Title                          localisedStrings   `json:"title"`
	AltTitles                      []localisedStrings `json:"altTitles"`
	Description                    localisedStrings   `json:"description"`
	IsLocked                       bool               `json:"isLocked"`
	Links                          localisedStrings   `json:"links"`
	OriginalLanguage               string             `json:"originalLanguage"`
	LastVolume                     string             `json:"lastVolume"`
	LastChapter                    string             `json:"lastChapter"`
	PublicationDemographic         string             `json:"publicationDemographic"`
	Status                         string             `json:"status"`
	Year                           int                `json:"year"`
	ContentRating                  string             `json:"contentRating"`
	Tags                           []tag              `json:"tags"`
	State                          string             `json:"state"`
	ChapterNumbersResetOnNewVolume bool               `json:"chapterNumbersResetOnNewVolume"`
	CreatedAt                      string             `json:"createdAt"`
	UpdatedAt                      string             `json:"updatedAt"`
	Version                        int                `json:"version"`
	AvailableTranslatedLanguages   []string           `json:"availableTranslatedLanguages"`
	LatestUploadedChapter          string             `json:"latestUploadedChapter"`
}

type coverAttributes map[string]interface{}

// type coverAttributes struct {
// 	Description string  `json:"description"`
// 	Volume      string  `json:"volume"`
// 	FileName    string  `json:"fileName"`
// 	Locale      string  `json:"locale"`
// 	CreatedAt   string  `json:"createdAt"`
// 	UpdatedAt   string  `json:"updatedAt"`
// 	Version     float64 `json:"version"`
// }

type chapterAttributes struct {
	Title              string `json:"title"`
	Volume             string `json:"volume"`
	Chapter            string `json:"chapter"`
	Pages              int    `json:"pages"`
	TranslatedLanguage string `json:"translatedLanguage"`
	Uploader           string `json:"uploader"`
	ExternalURL        string `json:"externalURL"`
	Version            int    `json:"version"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
	PublishAt          string `json:"publishAt"`
	ReadableAt         string `json:"readableAt"`
}
