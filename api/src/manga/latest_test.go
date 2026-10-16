package manga

import "testing"

func TestGetLatestMangaSkipsIgnoredChapter(t *testing.T) {
	newManga := func(source, url, chapter, chapterURL string) *Manga {
		return &Manga{Source: source, URL: url, LastReleasedChapter: &Chapter{Chapter: chapter, URL: chapterURL}}
	}
	a := newManga("mangadex", "https://a", "71", "https://a/71")
	b := newManga("rawkuma", "https://b", "72", "https://b/72")
	c := newManga("mangahub", "https://c", "71", "https://c/71")
	mangas := []*Manga{a, b, c}

	if got, _ := GetLatestManga(mangas); got != b {
		t.Fatalf("without ignored chapters, got %s, want %s", got.URL, b.URL)
	}

	b.IgnoredChapter = &Chapter{Chapter: "72", URL: "https://b/72"}
	if !b.IsLastReleasedChapterIgnored() {
		t.Fatal("b's last released chapter should be ignored")
	}
	if got, _ := GetLatestManga(mangas); got == b {
		t.Fatal("the manga with the ignored chapter was chosen")
	}

	a.LastReleasedChapter = &Chapter{Chapter: "72", URL: "https://a/72"}
	if got, _ := GetLatestManga(mangas); got != a {
		t.Fatalf("after another manga released the chapter, got %s, want %s", got.URL, a.URL)
	}

	b.LastReleasedChapter = &Chapter{Chapter: "73", URL: "https://b/73"}
	if b.IsLastReleasedChapterIgnored() {
		t.Fatal("a new chapter should not be ignored")
	}
	if got, _ := GetLatestManga(mangas); got != b {
		t.Fatalf("after the ignored manga released a new chapter, got %s, want %s", got.URL, b.URL)
	}
}

func TestGetLatestMangaAllIgnored(t *testing.T) {
	a := &Manga{Source: "mangadex", LastReleasedChapter: &Chapter{Chapter: "72"}, IgnoredChapter: &Chapter{Chapter: "72"}}
	if got, err := GetLatestManga([]*Manga{a}); err != nil || got != a {
		t.Fatalf("got %v, %v, want the only manga", got, err)
	}
}

func TestIsLastReleasedChapterIgnoredCustomMangaComparesURL(t *testing.T) {
	m := &Manga{
		Source:              CustomMangaSource,
		LastReleasedChapter: &Chapter{Chapter: UnknownChapter, URL: "https://site/ch-b"},
		IgnoredChapter:      &Chapter{Chapter: UnknownChapter, URL: "https://site/ch-a"},
	}
	if m.IsLastReleasedChapterIgnored() {
		t.Fatal("a custom manga chapter with another URL should not be ignored")
	}
	m.IgnoredChapter.URL = "https://site/ch-b"
	if !m.IsLastReleasedChapterIgnored() {
		t.Fatal("a custom manga chapter with the same URL should be ignored")
	}
}
