package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/diogovalentte/mantium/api/src"
)

func TestTestCustomMangaSelectors(t *testing.T) {
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a class="chapter" href="/chapter-7">Chapter 7</a></body></html>`))
	}))
	defer page.Close()

	tests := []struct {
		name        string
		body        map[string]any
		wantStatus  int
		wantChapter string
		wantURL     string
		userMessage string
	}{
		{
			name: "valid selectors",
			body: map[string]any{
				"url":           page.URL,
				"name_selector": map[string]any{"selector": "css:a.chapter", "regex": `Chapter (\d+)`},
				"url_selector":  map[string]any{"selector": "css:a.chapter", "attribute": "href"},
			},
			wantStatus:  http.StatusOK,
			wantChapter: "7",
			wantURL:     page.URL + "/chapter-7",
		},
		{
			name:        "no selectors",
			body:        map[string]any{"url": page.URL},
			wantStatus:  http.StatusBadRequest,
			userMessage: "Provide at least one selector to test",
		},
		{
			name: "selector doesn't match",
			body: map[string]any{
				"url":           page.URL,
				"name_selector": map[string]any{"selector": "css:div.nothing"},
			},
			wantStatus:  http.StatusBadRequest,
			userMessage: "Chapter name selector didn't match any element on the page",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, _ := json.Marshal(test.body)
			req, _ := http.NewRequest(http.MethodPost, "/v1/custom_manga/test_selectors", bytes.NewReader(body))
			w := httptest.NewRecorder()
			api.SetupRouter().ServeHTTP(w, req)

			if w.Code != test.wantStatus {
				t.Fatalf("got status %d, want %d: %s", w.Code, test.wantStatus, w.Body.String())
			}
			var resp struct {
				Chapter struct {
					Chapter string
					URL     string
				} `json:"chapter"`
				UserMessage string `json:"user_message"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatal(err)
			}
			if resp.UserMessage != test.userMessage {
				t.Errorf("got user_message %q, want %q", resp.UserMessage, test.userMessage)
			}
			if resp.Chapter.Chapter != test.wantChapter || resp.Chapter.URL != test.wantURL {
				t.Errorf("got chapter %q %q, want %q %q", resp.Chapter.Chapter, resp.Chapter.URL, test.wantChapter, test.wantURL)
			}
		})
	}
}
