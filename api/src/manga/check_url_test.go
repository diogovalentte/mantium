package manga

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/diogovalentte/mantium/api/src/errordefs"
)

func TestCustomMangaScrapingPageURL(t *testing.T) {
	tests := []struct {
		name     string
		scraping CustomMangaScraping
		want     string
	}{
		{name: "manga URL", scraping: CustomMangaScraping{URL: "https://example.com/manga"}, want: "https://example.com/manga"},
		{name: "check URL wins", scraping: CustomMangaScraping{URL: "https://example.com/manga", CheckURL: "https://example.com/feed.xml"}, want: "https://example.com/feed.xml"},
		{name: "check URL without manga URL", scraping: CustomMangaScraping{URL: CustomMangaURLPrefix + "/abc", CheckURL: "https://example.com/feed.xml"}, want: "https://example.com/feed.xml"},
		{name: "no real page", scraping: CustomMangaScraping{URL: CustomMangaURLPrefix + "/abc"}, want: ""},
		{name: "nothing", scraping: CustomMangaScraping{}, want: ""},
	}
	for _, test := range tests {
		if got := test.scraping.PageURL(); got != test.want {
			t.Errorf("%s: got %q, want %q", test.name, got, test.want)
		}
	}
}

func TestGetCustomMangaLastReleasedChapterUsesCheckURL(t *testing.T) {
	allowPrivateURLs(t)
	var mangaPageRequested atomic.Bool
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mangaPageRequested.Store(true)
	}))
	defer page.Close()

	feedServer := newCustomMangaTestServer()
	defer feedServer.Close()

	chapter, err := GetCustomMangaLastReleasedChapter(CustomMangaScraping{
		URL:          page.URL + "/manga",
		CheckURL:     feedServer.URL + "/feed.xml",
		NameSelector: &HTMLSelector{Selector: "xpath://item/title", Regex: `Chapter (\d+)`, GetFirst: true},
		URLSelector:  &HTMLSelector{Selector: "xpath://item/link", GetFirst: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if chapter.Chapter != "21" || chapter.URL != "https://example.com/manga/21" {
		t.Fatalf("got chapter %q %q, want the newest of the feed", chapter.Chapter, chapter.URL)
	}
	if mangaPageRequested.Load() {
		t.Fatal("the manga URL was requested instead of the check URL")
	}
}

func TestChapterPlaceholderURLComesFromTheMangaURL(t *testing.T) {
	allowPrivateURLs(t)
	server := newCustomMangaTestServer()
	defer server.Close()

	mangaURL := "https://example.com/manga/one"
	scraping := CustomMangaScraping{
		URL:          mangaURL,
		CheckURL:     server.URL + "/feed.xml",
		NameSelector: &HTMLSelector{Selector: "xpath://item/title", Regex: `Chapter (\d+)`, GetFirst: true},
	}
	chapter, err := GetCustomMangaLastReleasedChapter(scraping)
	if err != nil {
		t.Fatal(err)
	}
	// Another check URL with the same chapter keeps the chapter URL, so it isn't seen as a new chapter
	scraping.CheckURL = server.URL + "/feed.xml?other=1"
	again, err := GetCustomMangaLastReleasedChapter(scraping)
	if err != nil {
		t.Fatal(err)
	}
	want := customMangaChapterPlaceholderURL(mangaURL, "21")
	if chapter.URL != want || again.URL != want {
		t.Fatalf("got chapter URLs %q and %q, want %q", chapter.URL, again.URL, want)
	}

	chapters, err := GetCustomMangaChapters(scraping)
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) == 0 || chapters[0].URL != want {
		t.Fatalf("got chapters %+v, want the newest with URL %q", chapters, want)
	}
}

func TestGetCustomMangaLastReleasedChapterWithoutPage(t *testing.T) {
	for _, useBrowser := range []bool{false, true} {
		_, err := GetCustomMangaLastReleasedChapter(CustomMangaScraping{
			URL:          CustomMangaURLPrefix + "/abc",
			NameSelector: &HTMLSelector{Selector: "css:a"},
			UseBrowser:   useBrowser,
		})
		var userErr *errordefs.UserError
		if !errors.As(err, &userErr) || userErr.Message != "Provide the manga URL or the check URL to use the chapter selectors" {
			t.Fatalf("browser %v: got %v, want the no page error", useBrowser, err)
		}
	}
}
