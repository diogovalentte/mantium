//go:build live

package livetest

import (
	"bytes"
	"image"
	"net/url"
	"testing"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources/models"
)

type liveSource interface {
	GetMangaMetadata(string, string) (*manga.Manga, error)
	GetChapterMetadata(string, string, string, string, string) (*manga.Chapter, error)
	GetLastChapterMetadata(string, string) (*manga.Chapter, error)
	GetChaptersMetadata(string, string) ([]*manga.Chapter, error)
	Search(string, int) ([]*models.MangaSearchResult, error)
}

func CheckLiveSource(t *testing.T, s liveSource, target string) {
	t.Helper()
	checkURL := func(raw string) {
		t.Helper()
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
			t.Fatalf("invalid URL %q", raw)
		}
	}
	checkChapter := func(c *manga.Chapter) {
		t.Helper()
		if c == nil || c.Name == "" || c.Chapter == "" {
			t.Fatalf("missing chapter metadata: %+v", c)
		}
		checkURL(c.URL)
	}
	m, err := s.GetMangaMetadata(target, "")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil || m.Name == "" || m.Source == "" {
		t.Fatalf("missing manga metadata: %+v", m)
	}
	checkURL(m.URL)
	checkURL(m.CoverImgURL)
	if _, _, err := image.DecodeConfig(bytes.NewReader(m.CoverImg)); err != nil {
		t.Fatalf("invalid cover: %v", err)
	}
	checkChapter(m.LastReleasedChapter)
	c, err := s.GetLastChapterMetadata(target, m.InternalID)
	if err != nil {
		t.Fatal(err)
	}
	checkChapter(c)
	c, err = s.GetChapterMetadata(target, m.InternalID, c.Chapter, c.URL, c.InternalID)
	if err != nil {
		t.Fatal(err)
	}
	checkChapter(c)
	chapters, err := s.GetChaptersMetadata(target, m.InternalID)
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) == 0 {
		t.Fatal("empty chapter list")
	}
	for _, c := range chapters {
		checkChapter(c)
	}
	results, err := s.Search(m.Name, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("empty search results")
	}
	for _, r := range results {
		if r.Name == "" {
			t.Fatal("empty result name")
		}
		checkURL(r.URL)
	}
}
