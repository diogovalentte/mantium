package mangahub

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/testutil"
)

func TestFixtureMetadata(t *testing.T) {
	server := testutil.Fixtures(t, map[string]string{"/": "manga.json"})
	old := baseAPIURL
	baseAPIURL = server.URL + "/"
	t.Cleanup(func() { baseAPIURL = old })
	oldUploads := baseUploadsURL
	baseUploadsURL = server.URL
	t.Cleanup(func() { baseUploadsURL = oldUploads })
	s := &Source{}
	target := "https://mangahub.io/manga/fixture-story"
	internal := ""
	m, err := s.GetMangaMetadata(target, internal)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.CoverImg) == 0 || m.Name != "Fixture Story" || m.Source != "mangahub" || m.LastReleasedChapter == nil || m.LastReleasedChapter.Chapter != "12.5" {
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
	chapter, err := s.GetChapterMetadata(target, internal, "12.5", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if chapter.Chapter != "12.5" {
		t.Fatalf("unexpected chapter: %+v", chapter)
	}

	results, err := s.Search("Fixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "Fixture Story" {
		t.Fatalf("unexpected search: %+v", results)
	}
}

func TestFixtureMissingChapter(t *testing.T) {
	server := testutil.Fixtures(t, map[string]string{"/": "manga.json"})
	old := baseAPIURL
	baseAPIURL = server.URL + "/"
	t.Cleanup(func() { baseAPIURL = old })
	s := &Source{}
	for _, number := range []string{"999", "not-a-number"} {
		if _, err := s.GetChapterMetadata("https://mangahub.io/manga/fixture-story", "", number, "", ""); err == nil {
			t.Fatalf("expected missing chapter error for %q", number)
		}
	}
}
