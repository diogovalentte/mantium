package mangadex

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/testutil"
)

func TestFixtureMetadata(t *testing.T) {
	server := testutil.Fixtures(t, map[string]string{"/manga/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa": "manga.json", "/manga/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/feed": "feed.json", "/chapter/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb": "chapter.json", "/manga": "search.json"})
	old := baseAPIURL
	baseAPIURL = server.URL
	t.Cleanup(func() { baseAPIURL = old })
	oldUploads := baseUploadsURL
	baseUploadsURL = server.URL
	t.Cleanup(func() { baseUploadsURL = oldUploads })
	s := &Source{}
	target := "https://mangadex.org/title/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	internal := ""
	m, err := s.GetMangaMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.CoverImg) == 0 || m.Name != "Fixture Story" || m.Source != "mangadex" || m.LastReleasedChapter == nil || m.LastReleasedChapter.Chapter != "12.5" {
		t.Fatalf("unexpected manga: %+v", m)
	}
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
	chapter, err := s.GetChapterMetadata(target, internal, "12.5", "https://mangadex.org/chapter/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "")
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

func TestFixtureNoChapters(t *testing.T) {
	server := testutil.Fixtures(t, map[string]string{"/manga/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/feed": "empty-feed.json"})
	old := baseAPIURL
	baseAPIURL = server.URL
	t.Cleanup(func() { baseAPIURL = old })
	s := &Source{}
	if _, err := s.GetLastChapterMetadata("https://mangadex.org/title/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", ""); err == nil {
		t.Fatal("expected missing chapter error")
	}
}
