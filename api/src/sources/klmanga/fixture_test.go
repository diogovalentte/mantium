package klmanga

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/testutil"
)

func TestFixtureMetadata(t *testing.T) {
	server := testutil.Fixtures(t, map[string]string{"/page/1/": "search.html", "/manga": "manga.html", "/chapter-12.5": "chapter.html", "/missing": "empty.html"})
	old := siteURL.Get()
	siteURL.Set(server.URL)
	t.Cleanup(func() { siteURL.Set(old) })
	target := server.URL + "/manga"
	internal := ""
	s := &Source{}
	m, err := s.GetMangaMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Fixture Story" || m.Source != "klmanga" || len(m.CoverImg) == 0 || m.CoverImgURL != server.URL+"/cover.png" {
		t.Fatalf("unexpected manga: %+v", m)
	}
	if m.LastReleasedChapter == nil || m.LastReleasedChapter.Chapter != "12.5" {
		t.Fatalf("unexpected last chapter: %+v", m.LastReleasedChapter)
	}
	chapters, err := s.GetChaptersMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 1 || chapters[0].Chapter != "12.5" || chapters[0].URL != server.URL+"/chapter-12.5" {
		t.Fatalf("unexpected chapters: %+v", chapters)
	}
	last, err := s.GetLastChapterMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	if last.Chapter != "12.5" {
		t.Fatalf("unexpected last chapter: %+v", last)
	}
	chapter, err := s.GetChapterMetadata(target, internal, "12.5", server.URL+"/chapter-12.5", "")
	if err != nil {
		t.Fatal(err)
	}
	if chapter.Chapter != "12.5" {
		t.Fatalf("unexpected chapter: %+v", chapter)
	}
	if _, err := s.GetMangaMetadata(server.URL+"/missing", ""); err == nil {
		t.Fatal("expected error for missing metadata")
	}
	results, err := s.Search("Fixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "Fixture Story" || results[0].LastChapter != "12.5" {
		t.Fatalf("unexpected search: %+v", results)
	}
}
