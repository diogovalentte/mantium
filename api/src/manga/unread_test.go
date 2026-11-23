package manga

import "testing"

func TestIsUnreadChapter(t *testing.T) {
	tests := []struct {
		lastRead, lastReadURL, lastReleased, lastReleasedURL string
		want                                                 bool
	}{
		{"", "", "", "", false},
		{"10", "", "", "", false},
		{"", "", "10", "", true},
		{"9", "", "10", "", true},
		{"10", "", "10", "", false},
		{"11", "", "10", "", false},
		{"Extra", "", "10", "", true},
		{UnknownChapter, "https://a/1", UnknownChapter, "https://a/2", true},
		{UnknownChapter, "https://a/1", UnknownChapter, "https://a/1", false},
		// Same name, different chapter (e.g. a chapter re-released with the same number)
		{"10", "https://a/10", "10", "https://a/10-v2", true},
		{"10", "https://a/10", "10", "https://a/10", false},
		// Without both URLs there is nothing to compare
		{"10", "", "10", "https://a/10", false},
		{"10", "https://a/10", "10", "", false},
		// A multimanga read on one source and released on another: the URLs differ but it's the same chapter
		{"10", "https://mangadex.org/chapter/x", "10", "https://mangaplus.shueisha.co.jp/viewer/y", false},
	}
	for _, test := range tests {
		got := IsUnreadChapter(test.lastRead, test.lastReadURL, test.lastReleased, test.lastReleasedURL)
		if got != test.want {
			t.Errorf("IsUnreadChapter(%q, %q, %q, %q) = %v, want %v", test.lastRead, test.lastReadURL, test.lastReleased, test.lastReleasedURL, got, test.want)
		}
	}
}

func TestFilterUnreadChapterMangasSameNameDifferentURL(t *testing.T) {
	chapter := func(name, url string) *Chapter { return &Chapter{Chapter: name, URL: url} }
	mangas := []*Manga{
		{Name: "different URL", LastReadChapter: chapter("10", "https://a/10"), LastReleasedChapter: chapter("10", "https://a/10-v2")},
		{Name: "same URL", LastReadChapter: chapter("10", "https://a/10"), LastReleasedChapter: chapter("10", "https://a/10")},
		{Name: "read without URL", LastReadChapter: chapter("10", ""), LastReleasedChapter: chapter("10", "https://a/10")},
	}
	got := FilterUnreadChapterMangas(mangas)
	if len(got) != 1 || got[0].Name != "different URL" {
		names := []string{}
		for _, m := range got {
			names = append(names, m.Name)
		}
		t.Fatalf("got %v, want [different URL]", names)
	}
}
