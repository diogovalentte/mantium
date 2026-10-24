//go:build db

package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/diogovalentte/mantium/api/internal/testutil"
	"github.com/diogovalentte/mantium/api/src/util"
)

func TestCustomMangaCheckURLLifecycle(t *testing.T) {
	testutil.Database(t)
	// The test pages are served on 127.0.0.1
	util.SetAllowPrivateURLs(true)
	defer util.SetAllowPrivateURLs(false)

	// The manga page has no chapters, they're only in the feeds
	feed := `<?xml version="1.0"?><rss><channel><item><title>Chapter 21</title></item><item><title>Chapter 20</title></item></channel></rss>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/feed") {
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(feed))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>About this manga</body></html>`))
	}))
	defer server.Close()
	jsonBody := func(v any) string {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	nameSelector := map[string]any{"selector": "xpath://item/title", "regex": `Chapter (\d+)`, "get_first": true}
	type customManga struct {
		ID                  int
		URL                 string
		CheckURL            string
		LastReleasedChapter struct{ Chapter, URL string }
	}
	get := func() customManga {
		t.Helper()
		w := request(t, "GET", "/v1/multimangas", "", 200)
		var list struct {
			MultiMangas []struct{ ID int } `json:"multimangas"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.MultiMangas) != 1 {
			t.Fatalf("expected one multimanga: %s", w.Body.String())
		}
		w = request(t, "GET", fmt.Sprintf("/v1/multimanga?id=%d", list.MultiMangas[0].ID), "", 200)
		var got struct {
			MultiManga struct{ CurrentManga customManga } `json:"multimanga"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		return got.MultiManga.CurrentManga
	}

	request(t, "POST", "/v1/multimanga", jsonBody(map[string]any{
		"name": "Feed Only", "status": 1, "url": server.URL + "/manga", "check_url": server.URL + "/feed.xml",
		"last_released_chapter_name_selector": nameSelector,
	}), 200)
	m := get()
	if m.URL != server.URL+"/manga" || m.CheckURL != server.URL+"/feed.xml" || m.LastReleasedChapter.Chapter != "21" {
		t.Fatalf("unexpected manga: %+v", m)
	}
	placeholder := m.LastReleasedChapter.URL

	// Another check URL with the same chapter keeps the chapter
	request(t, "PATCH", fmt.Sprintf("/v1/custom_manga?id=%d", m.ID), jsonBody(map[string]any{
		"name": "Feed Only", "url": server.URL + "/manga", "check_url": server.URL + "/feed2.xml", "name_selector": nameSelector,
	}), 200)
	m = get()
	if m.CheckURL != server.URL+"/feed2.xml" || m.LastReleasedChapter.URL != placeholder {
		t.Fatalf("check URL change changed the chapter: %+v, want chapter URL %q", m, placeholder)
	}

	// Without the check URL, the selectors are applied to the manga page, which has no chapters
	w := request(t, "PATCH", fmt.Sprintf("/v1/custom_manga?id=%d", m.ID), jsonBody(map[string]any{
		"name": "Feed Only", "url": server.URL + "/manga", "name_selector": nameSelector,
	}), 400)
	if !strings.Contains(w.Body.String(), "didn't match any element") {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}

	// The check URL alone is enough to track chapters
	request(t, "PATCH", fmt.Sprintf("/v1/custom_manga?id=%d", m.ID), jsonBody(map[string]any{
		"name": "Feed Only", "check_url": server.URL + "/feed.xml", "name_selector": nameSelector,
	}), 200)
	m = get()
	if m.URL != "" || m.LastReleasedChapter.Chapter != "21" {
		t.Fatalf("unexpected manga without URL: %+v", m)
	}

	// No page at all can't use the selectors
	w = request(t, "PATCH", fmt.Sprintf("/v1/custom_manga?id=%d", m.ID), jsonBody(map[string]any{
		"name": "Feed Only", "name_selector": nameSelector,
	}), 400)
	if !strings.Contains(w.Body.String(), "Provide the manga URL or the check URL to use the chapter selectors") {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}

	// Testing with only the check URL
	w = request(t, "POST", "/v1/custom_manga/test_selectors", jsonBody(map[string]any{
		"check_url": server.URL + "/feed.xml", "name_selector": nameSelector,
	}), 200)
	if !strings.Contains(w.Body.String(), `"Chapter":"21"`) {
		t.Fatalf("unexpected test response: %s", w.Body.String())
	}
}
