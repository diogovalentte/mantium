package manga

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/flaresolverr"
)

// newFakeFlareSolverr starts a server that answers like FlareSolverr, returning body as the
// page at finalURL, and configures it as the FlareSolverr address. It returns the URLs it was asked for.
func newFakeFlareSolverr(t *testing.T, finalURL, body string) *[]string {
	t.Helper()
	requested := &[]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL string `json:"url"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		*requested = append(*requested, req.URL)
		json.NewEncoder(w).Encode(map[string]any{
			"status":   "ok",
			"solution": map[string]any{"url": finalURL, "status": 200, "response": body},
		})
	}))
	t.Cleanup(server.Close)
	flaresolverr.SetAddress(server.URL)
	t.Cleanup(func() { flaresolverr.SetAddress("") })

	return requested
}

func TestGetCustomMangaChaptersUsingFlareSolverr(t *testing.T) {
	allowPrivateURLs(t)
	requested := newFakeFlareSolverr(t, "https://example.com/title/one-piece/", customMangaPage)

	chapters, err := GetCustomMangaChapters(CustomMangaScraping{
		URL:             "https://example.com/one-piece",
		NameSelector:    &HTMLSelector{Selector: "css:a.chapter", GetFirst: true},
		URLSelector:     &HTMLSelector{Selector: "css:a.chapter", Attribute: "href", GetFirst: true},
		UseFlareSolverr: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(*requested) != 1 || (*requested)[0] != "https://example.com/one-piece" {
		t.Fatalf("FlareSolverr was asked for %v", *requested)
	}
	// Relative URLs are resolved against the page final URL that FlareSolverr returns.
	want := []string{"https://example.com/manga/chapter-12", "https://example.com/title/one-piece/chapter-11"}
	if len(chapters) != len(want) {
		t.Fatalf("got %d chapters, want %d", len(chapters), len(want))
	}
	for i, chapter := range chapters {
		if chapter.URL != want[i] {
			t.Errorf("chapter %d URL = %s, want %s", i, chapter.URL, want[i])
		}
	}
}

func TestCustomMangaFlareSolverrNotConfigured(t *testing.T) {
	flaresolverr.SetAddress("")

	_, err := GetCustomMangaLastReleasedChapter(CustomMangaScraping{
		URL:             "https://example.com/one-piece",
		NameSelector:    &HTMLSelector{Selector: "css:a.chapter"},
		UseFlareSolverr: true,
	})
	if !errors.Is(err, flaresolverr.ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestCustomMangaBrowserAndFlareSolverrTogether(t *testing.T) {
	requested := newFakeFlareSolverr(t, "https://example.com/", customMangaPage)

	_, err := GetCustomMangaLastReleasedChapter(CustomMangaScraping{
		URL:             "https://example.com/one-piece",
		NameSelector:    &HTMLSelector{Selector: "css:a.chapter"},
		UseBrowser:      true,
		UseFlareSolverr: true,
	})
	var userErr *errordefs.UserError
	if !errors.As(err, &userErr) || userErr != errBrowserAndFlareSolverr {
		t.Fatalf("err = %v, want errBrowserAndFlareSolverr", err)
	}
	if len(*requested) != 0 {
		t.Errorf("FlareSolverr was asked for %v", *requested)
	}

	if err := InsertPresetDB(&Preset{Name: "x", NameSelector: &HTMLSelector{Selector: "css:a"}, UseBrowser: true, UseFlareSolverr: true}); !errors.Is(err, errBrowserAndFlareSolverr) {
		t.Fatalf("InsertPresetDB err = %v, want errBrowserAndFlareSolverr", err)
	}
}

func TestDetectPresetsUsingFlareSolverr(t *testing.T) {
	allowPrivateURLs(t)
	server := newCustomMangaTestServer()
	defer server.Close()
	requested := newFakeFlareSolverr(t, server.URL+"/", `<html><body><b class="cf">Chapter 99</b></body></html>`)

	detections, err := DetectPresets(server.URL, []*Preset{
		{Name: "plain", NameSelector: &HTMLSelector{Selector: "css:a.chapter", Regex: chapterNumberRegex, GetFirst: true}},
		{Name: "cloudflare", NameSelector: &HTMLSelector{Selector: "css:b.cf", Regex: chapterNumberRegex, GetFirst: true}, UseFlareSolverr: true},
		{Name: "cloudflare 2", NameSelector: &HTMLSelector{Selector: "css:b.cf", Regex: chapterNumberRegex, GetFirst: true}, UseFlareSolverr: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(*requested) != 1 {
		t.Errorf("FlareSolverr was asked %d times, want once for both presets", len(*requested))
	}
	for i, want := range []string{"12", "99", "99"} {
		if detections[i].Chapter == nil || detections[i].Chapter.Chapter != want {
			t.Errorf("preset %s: chapter = %+v, message = %q, want chapter %s", detections[i].Preset.Name, detections[i].Chapter, detections[i].UserMessage, want)
		}
	}
}

func TestDetectPresetsWhenThePlainRequestIsRefused(t *testing.T) {
	allowPrivateURLs(t)
	// Like a site behind Cloudflare: plain requests get a challenge page with 403
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`<html><body>Just a moment...</body></html>`))
	}))
	defer site.Close()
	newFakeFlareSolverr(t, site.URL+"/", `<html><body><b class="cf">Chapter 99</b></body></html>`)

	plain := &Preset{Name: "plain", NameSelector: &HTMLSelector{Selector: "css:a.chapter", Regex: chapterNumberRegex}}
	cloudflare := &Preset{Name: "cloudflare", NameSelector: &HTMLSelector{Selector: "css:b.cf", Regex: chapterNumberRegex}, UseFlareSolverr: true}

	detections, err := DetectPresets(site.URL, []*Preset{plain, cloudflare})
	if err != nil {
		t.Fatalf("the plain request failure stopped the detection: %v", err)
	}
	if detections[0].Chapter != nil || detections[0].UserMessage == "" {
		t.Errorf("plain preset: %+v, want the plain request error", detections[0])
	}
	if detections[1].Chapter == nil || detections[1].Chapter.Chapter != "99" {
		t.Errorf("FlareSolverr preset: %+v, want chapter 99", detections[1])
	}

	// When no preset works, the plain request error is returned, as before
	_, err = DetectPresets(site.URL, []*Preset{plain})
	var userErr *errordefs.UserError
	if !errors.As(err, &userErr) || !strings.Contains(userErr.Message, "Forbidden") {
		t.Fatalf("err = %v, want the plain request error", err)
	}
}
