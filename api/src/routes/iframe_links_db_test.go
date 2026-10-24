//go:build db

package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/diogovalentte/mantium/api/internal/testutil"
	"github.com/diogovalentte/mantium/api/src/util"
)

func TestIframeLinksOfCustomMangaChaptersWithoutURL(t *testing.T) {
	testutil.Database(t)
	// The test page is served on 127.0.0.1
	util.SetAllowPrivateURLs(true)
	defer util.SetAllowPrivateURLs(false)

	feed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0"?><rss><channel><item><title>Chapter 21</title></item></channel></rss>`))
	}))
	defer feed.Close()
	jsonBody := func(v any) string {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	nameSelector := map[string]any{"selector": "xpath://item/title", "regex": `Chapter (\d+)`, "get_first": true}

	// Tracked by the chapter name only, in a feed, so the chapter has no URL
	mangaURL := "https://example.com/manga/with-url"
	request(t, "POST", "/v1/multimanga", jsonBody(map[string]any{
		"name": "With URL", "status": 1, "url": mangaURL, "check_url": feed.URL + "/feed.xml",
		"last_released_chapter_name_selector": nameSelector,
	}), 200)
	request(t, "POST", "/v1/multimanga", jsonBody(map[string]any{
		"name": "Without URL", "status": 1, "check_url": feed.URL + "/feed.xml",
		"last_released_chapter_name_selector": nameSelector,
	}), 200)

	html := request(t, "GET", "/v1/mangas/iframe?api_url=http://localhost:8080", "", 200).Body.String()

	// An empty href opens the iFrame itself
	if regexp.MustCompile(`href="\s*"`).MatchString(html) {
		t.Fatalf("iFrame has an empty link:\n%s", html)
	}
	if strings.Contains(html, "custom_manga.com") {
		t.Fatalf("iFrame shows a placeholder URL:\n%s", html)
	}
	// The chapter of the manga with URL links to the manga, also when there's no last read chapter
	for _, class := range []string{"last-released-chapter-label", "last-read-chapter-label"} {
		want := `<a href="` + mangaURL + `" class="chapter-label ` + class + `" target="_blank">`
		if !strings.Contains(html, want) {
			t.Fatalf("iFrame doesn't link the %s to the manga URL:\n%s", class, html)
		}
	}
	// The manga without URL has no links, only text
	for _, want := range []string{
		`<span class="manga-name">Without URL</span>`,
		`<span class="chapter-label last-released-chapter-label">21</span>`,
		`<span class="chapter-label last-read-chapter-label">N/A</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("iFrame doesn't have %q:\n%s", want, html)
		}
	}
}
