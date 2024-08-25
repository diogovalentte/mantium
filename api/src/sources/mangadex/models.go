package mangadex

import (
	"fmt"
	"strings"
)

type genericRelationship struct {
	Attributes map[string]interface{} `json:"attributes"`
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
}

type localisedStrings map[string]string

func (ls localisedStrings) get() string {
	if val, ok := ls["en"]; ok {
		return val
	}
	if val, ok := ls["ja"]; ok {
		return val
	}
	if val, ok := ls["ja-ro"]; ok {
		return val
	}
	for _, val := range ls {
		return val
	}

	return ""
}

type tag struct {
	Relationships []genericRelationship `json:"relationships"`
	ID            string                `json:"id"`
	Type          string                `json:"type"`
	Attributes    tagAttributes         `json:"attributes"`
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

	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}

type mangaAttributes struct {
	Title       localisedStrings   `json:"title"`
	Description localisedStrings   `json:"description"`
	Links       localisedStrings   `json:"links"`
	LastChapter string             `json:"lastChapter"`
	Status      string             `json:"status"`
	AltTitles   []localisedStrings `json:"altTitles"`
	Year        int                `json:"year"`
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
	TranslatedLanguage string `json:"translatedLanguage"`
	Uploader           string `json:"uploader"`
	ExternalURL        string `json:"externalURL"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
	PublishAt          string `json:"publishAt"`
	ReadableAt         string `json:"readableAt"`
	Pages              int    `json:"pages"`
	Version            int    `json:"version"`
}
