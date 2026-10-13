package mangaupdates

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/testutil"
)

func TestFixtureMetadata(t *testing.T) {
	server := testutil.Fixtures(t, map[string]string{"/series": "series.html", "/v1/series/123": "manga.json", "/v1/releases/search": "chapters.json", "/v1/series/search": "search.json"})
	old := baseAPIURL
	baseAPIURL = server.URL
	t.Cleanup(func() { baseAPIURL = old })
	s := &Source{}
	target := server.URL + "/series"
	internal := ""
	m, err := s.GetMangaMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	if m.InternalID != "123" || len(m.CoverImg) == 0 || m.Name != "Fixture Story" || m.Source != "mangaupdates" || m.LastReleasedChapter == nil || m.LastReleasedChapter.Chapter != "12.5" {
		t.Fatalf("unexpected manga: %+v", m)
	}
	internal = m.InternalID
	chapters, err := s.GetChaptersMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 1 || chapters[0].Chapter != "12.5" || chapters[0].UpdatedAt.IsZero() {
		t.Fatalf("unexpected chapters: %+v", chapters)
	}
	last, err := s.GetLastChapterMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	if last.Chapter != "12.5" {
		t.Fatalf("unexpected last chapter: %+v", last)
	}
	chapter, err := s.GetChapterMetadata(target, internal, "12.5", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if chapter.Chapter != "12.5" {
		t.Fatalf("unexpected chapter: %+v", chapter)
	}
	results, err := s.Search("Fixture", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "Fixture Story" {
		t.Fatalf("unexpected results: %+v", results)
	}
}
