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
	"github.com/diogovalentte/mantium/api/src/flaresolverr"
	"github.com/diogovalentte/mantium/api/src/util"
)

func TestCustomMangaFlareSolverrLifecycle(t *testing.T) {
	testutil.Database(t)
	util.SetAllowPrivateURLs(true)
	defer util.SetAllowPrivateURLs(false)

	// The site answers plain requests with a challenge page and only FlareSolverr gets the chapters
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`<html><body>Just a moment...</body></html>`))
	}))
	defer site.Close()
	fs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ URL string }
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "solution": map[string]any{
			"url": req.URL, "status": 200, "response": `<html><body><a class="chapter" href="/c/30">Chapter 30</a></body></html>`,
		}})
	}))
	defer fs.Close()
	flaresolverr.SetAddress(fs.URL)
	defer flaresolverr.SetAddress("")

	jsonBody := func(v any) string {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	nameSelector := map[string]any{"selector": "css:a.chapter", "regex": `Chapter (\d+)`, "get_first": true}
	type customManga struct {
		ID                                         int
		LastReleasedChapterSelectorUseBrowser      bool
		LastReleasedChapterSelectorUseFlareSolverr bool
		LastReleasedChapter                        struct{ Chapter, URL string }
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
		"name": "Behind Cloudflare", "status": 1, "url": site.URL + "/manga",
		"last_released_chapter_name_selector":             nameSelector,
		"last_released_chapter_url_selector":              map[string]any{"selector": "css:a.chapter", "attribute": "href", "get_first": true},
		"last_released_chapter_selector_use_flaresolverr": true,
	}), 200)
	m := get()
	if !m.LastReleasedChapterSelectorUseFlareSolverr || m.LastReleasedChapterSelectorUseBrowser || m.LastReleasedChapter.Chapter != "30" {
		t.Fatalf("unexpected manga: %+v", m)
	}
	// The relative chapter URL is resolved against the page URL FlareSolverr returned
	if w := request(t, "GET", "/v1/multimangas", "", 200); !strings.Contains(w.Body.String(), `"URL":"`+site.URL+`/c/30"`) {
		t.Fatalf("chapter URL not resolved against the page URL: %s", w.Body.String())
	}

	// The browser and FlareSolverr can't be used together
	w := request(t, "PATCH", fmt.Sprintf("/v1/custom_manga?id=%d", m.ID), jsonBody(map[string]any{
		"name": "Behind Cloudflare", "url": site.URL + "/manga", "name_selector": nameSelector, "use_browser": true, "use_flaresolverr": true,
	}), 400)
	if !strings.Contains(w.Body.String(), "can't be used together") {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}

	// Without FlareSolverr, the site's challenge page has no chapters
	request(t, "PATCH", fmt.Sprintf("/v1/custom_manga?id=%d", m.ID), jsonBody(map[string]any{
		"name": "Behind Cloudflare", "url": site.URL + "/manga", "name_selector": nameSelector,
	}), 400)
	if m = get(); !m.LastReleasedChapterSelectorUseFlareSolverr {
		t.Fatalf("a failed update changed the manga: %+v", m)
	}

	// Testing the selectors through FlareSolverr
	w = request(t, "POST", "/v1/custom_manga/test_selectors", jsonBody(map[string]any{
		"url": site.URL + "/manga", "name_selector": nameSelector, "use_flaresolverr": true,
	}), 200)
	if !strings.Contains(w.Body.String(), `"Chapter":"30"`) {
		t.Fatalf("unexpected test response: %s", w.Body.String())
	}

	// Presets keep the option
	request(t, "POST", "/v1/custom_manga/presets", jsonBody(map[string]any{
		"name": "Cloudflare site", "name_selector": nameSelector, "use_flaresolverr": true,
	}), 200)
	w = request(t, "GET", "/v1/custom_manga/presets", "", 200)
	var presets struct {
		Presets []struct {
			Name            string `json:"name"`
			UseFlareSolverr bool   `json:"use_flaresolverr"`
		} `json:"presets"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &presets); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range presets.Presets {
		if p.Name == "Cloudflare site" {
			found = p.UseFlareSolverr
		}
	}
	if !found {
		t.Fatalf("preset lost use_flaresolverr: %s", w.Body.String())
	}
}
