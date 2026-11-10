package mangahub

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/internal/testutil"
	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/flaresolverr"
	"github.com/diogovalentte/mantium/api/src/util"
)

// useSite points the source to server and fixes the clock used by relative release times.
func useSite(t *testing.T, siteURL string) {
	t.Helper()
	oldSite, oldNow := baseSiteURL, now
	baseSiteURL = siteURL
	now = func() time.Time { return time.Date(2026, 11, 9, 15, 30, 0, 0, time.UTC) }
	t.Cleanup(func() { baseSiteURL, now = oldSite, oldNow })
}

func TestFixtureMetadata(t *testing.T) {
	server := testutil.Fixtures(t, map[string]string{"/manga/fixture-story_7": "manga.html", "/search": "search.html"})
	useSite(t, server.URL)
	s := &Source{}
	target := "https://mangahub.io/manga/fixture-story_7"
	internal := ""

	m, err := s.GetMangaMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.CoverImg) == 0 || m.CoverImgURL != server.URL+"/cover.png" {
		t.Fatalf("unexpected cover: %s (%d bytes)", m.CoverImgURL, len(m.CoverImg))
	}
	if m.Name != "Fixture Story" || m.Source != "mangahub" || m.URL != server.URL+"/manga/fixture-story_7" {
		t.Fatalf("unexpected manga: %+v", m)
	}
	if m.LastReleasedChapter == nil || m.LastReleasedChapter.Chapter != "12.5" || m.LastReleasedChapter.Type != 1 {
		t.Fatalf("unexpected last released chapter: %+v", m.LastReleasedChapter)
	}

	chapters, err := s.GetChaptersMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	// Newest first, across the chapter tabs, without the other uploads of a chapter
	want := []struct {
		chapter, name, url string
		date               time.Time
	}{
		{"12.5", "The Return", "https://mangahub.io/chapter/fixture-story_7/chapter-12.5", time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)},
		{"12", "Ch.12", "https://mangahub.io/chapter/fixture-story_7/chapter-12", time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)},
		{"11", "Chapter 11", "https://mangahub.io/chapter/fixture-story_7/chapter-11", time.Time{}},
		{"1", "Romance Dawn", "https://mangahub.io/chapter/fixture-story_7/chapter-1", time.Date(2016, 1, 19, 0, 0, 0, 0, time.UTC)},
	}
	if len(chapters) != len(want) {
		t.Fatalf("got %d chapters, want %d: %+v", len(chapters), len(want), chapters)
	}
	for i, w := range want {
		c := chapters[i]
		if c.Chapter != w.chapter || c.Name != w.name || c.URL != w.url || !c.UpdatedAt.Equal(w.date) {
			t.Errorf("chapter %d = %+v, want %+v", i, c, w)
		}
	}

	last, err := s.GetLastChapterMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	if last.Chapter != "12.5" {
		t.Fatalf("unexpected last chapter: %+v", last)
	}

	for _, test := range []struct{ chapter, chapterURL, want string }{
		{"12", "", "12"},
		{"12.0", "", "12"},
		{"1", "", "1"},
		{"", "https://mangahub.io/chapter/fixture-story_7/chapter-11", "11"},
		{"999", "https://mangahub.io/chapter/fixture-story_7/chapter-11", "11"},
	} {
		chapter, err := s.GetChapterMetadata(target, internal, test.chapter, test.chapterURL, "")
		if err != nil {
			t.Fatalf("chapter %q URL %q: %s", test.chapter, test.chapterURL, err)
		}
		if chapter.Chapter != test.want {
			t.Fatalf("chapter %q URL %q: got %+v, want chapter %s", test.chapter, test.chapterURL, chapter, test.want)
		}
	}

	results, err := s.Search("Fixture", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("unexpected search: %+v", results)
	}
	first, second := results[0], results[1]
	if first.Name != "Fixture Story" || first.URL != server.URL+"/manga/fixture-story_7" || first.CoverURL != server.URL+"/cover.png" ||
		first.LastChapter != "12.5" || first.LastChapterURL != "https://mangahub.io/chapter/fixture-story_7/chapter-12.5" || first.Status != "ongoing" {
		t.Fatalf("unexpected first search result: %+v", first)
	}
	if second.Name != "Other Story" || second.Status != "completed" || second.CoverURL == "" || second.LastChapter != "3" {
		t.Fatalf("unexpected second search result: %+v", second)
	}

	results, err = s.Search("Fixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("search limit not applied: %+v", results)
	}
}

func TestFixtureMissingChapter(t *testing.T) {
	server := testutil.Fixtures(t, map[string]string{"/manga/fixture-story_7": "manga.html"})
	useSite(t, server.URL)
	s := &Source{}
	for _, number := range []string{"999", "not-a-number"} {
		_, err := s.GetChapterMetadata("https://mangahub.io/manga/fixture-story_7", "", number, "", "")
		if !errors.Is(err, errordefs.ErrChapterNotFound) && !util.ErrorContains(err, errordefs.ErrChapterNotFound.Message) {
			t.Fatalf("chapter %q: err = %v, want chapter not found", number, err)
		}
	}
	if _, err := s.GetChapterMetadata("https://mangahub.io/manga/fixture-story_7", "", "", "", ""); err == nil {
		t.Fatal("expected an error without chapter and URL")
	}
}

func TestMangaNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	useSite(t, server.URL)

	_, err := (&Source{}).GetMangaMetadata("https://mangahub.io/manga/missing", "")
	if !util.ErrorContains(err, errordefs.ErrMangaNotFound.Message) {
		t.Fatalf("err = %v, want manga not found", err)
	}
}

func TestBlockedWithoutFlareSolverr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("<html><body>Just a moment...</body></html>"))
	}))
	defer server.Close()
	useSite(t, server.URL)
	flaresolverr.SetAddress("")

	_, err := (&Source{}).GetMangaMetadata("https://mangahub.io/manga/fixture-story_7", "")
	var userErr *errordefs.UserError
	if !errors.As(err, &userErr) || !strings.Contains(userErr.Message, "FLARESOLVERR_ADDRESS") {
		t.Fatalf("err = %v, want the FlareSolverr hint", err)
	}
}

func TestPagesThroughFlareSolverr(t *testing.T) {
	util.SetAllowPrivateURLs(true)
	t.Cleanup(func() { util.SetAllowPrivateURLs(false) })
	page, err := os.ReadFile("testdata/manga.html")
	if err != nil {
		t.Fatal(err)
	}
	// The site refuses plain requests, so the page only comes from FlareSolverr
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer site.Close()
	useSite(t, site.URL)
	var requested []string
	fs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ URL string }
		json.NewDecoder(r.Body).Decode(&req)
		requested = append(requested, req.URL)
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "solution": map[string]any{
			"url": req.URL, "status": 200, "response": strings.ReplaceAll(string(page), "{{BASE}}", "https://thumb.example.com"),
		}})
	}))
	defer fs.Close()
	flaresolverr.SetAddress(fs.URL)
	t.Cleanup(func() { flaresolverr.SetAddress("") })

	chapters, err := (&Source{}).GetChaptersMetadata("https://mangahub.io/manga/fixture-story_7", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 4 || chapters[0].Chapter != "12.5" {
		t.Fatalf("unexpected chapters: %+v", chapters)
	}
	if len(requested) != 1 || requested[0] != site.URL+"/manga/fixture-story_7" {
		t.Fatalf("FlareSolverr was asked for %v", requested)
	}
}

func TestParseReleaseTime(t *testing.T) {
	useSite(t, baseSiteURL)
	current := now()
	today := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	for text, want := range map[string]time.Time{
		"01-19-2016":         time.Date(2016, 1, 19, 0, 0, 0, 0, time.UTC),
		"just now":           current,
		"less than an hour":  current.Add(-30 * time.Minute),
		"5 minutes ago":      current.Add(-5 * time.Minute),
		"an hour ago":        current.Add(-time.Hour),
		"1 hour ago":         current.Add(-time.Hour),
		"3 hours ago":        current.Add(-3 * time.Hour),
		"Yesterday":          today.AddDate(0, 0, -1),
		"a day ago":          today.AddDate(0, 0, -1),
		"4 days ago":         today.AddDate(0, 0, -4),
		"1 week ago":         today.AddDate(0, 0, -7),
		"2 weeks ago":        today.AddDate(0, 0, -14),
		"a month ago":        today.AddDate(0, -1, 0),
		"3 months ago":       today.AddDate(0, -3, 0),
		"2 years ago":        today.AddDate(-2, 0, 0),
		"":                   {},
		"sometime last week": {},
	} {
		if got := parseReleaseTime(text); !got.Equal(want) {
			t.Errorf("parseReleaseTime(%q) = %s, want %s", text, got, want)
		}
	}
}
