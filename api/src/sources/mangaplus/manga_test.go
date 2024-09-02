package mangaplus

import (
	"regexp"
	"testing"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

type mangaTestType struct {
	expected *manga.Manga
	url      string
}

var mangasTestTable = []mangaTestType{
	{
		expected: &manga.Manga{
			Name:   "Witch Watch",
			Source: sourceName,
			URL:    "https://mangaplus.shueisha.co.jp/titles/100145",
		},
		url: "https://mangaplus.shueisha.co.jp/titles/100145",
	},
	{
		expected: &manga.Manga{
			Name:   "Dandadan",
			Source: sourceName,
			URL:    "https://mangaplus.shueisha.co.jp/titles/100171",
		},
		url: "https://mangaplus.shueisha.co.jp/titles/100171",
	},
	{
		expected: &manga.Manga{
			Name:   "2.5 Dimensional Seduction",
			Source: sourceName,
			URL:    "https://mangaplus.shueisha.co.jp/titles/100282",
		},
		url: "https://mangaplus.shueisha.co.jp/titles/100282",
	},
}

func TestGetMangaMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			expected := test.expected
			mangaURL := test.url

			actualManga, err := source.GetMangaMetadata(mangaURL, "", false)
			if err != nil {
				t.Fatalf("error while getting manga: %v", err)
			}

			if actualManga.Name != expected.Name {
				t.Fatalf("expected manga name %s, got %s", expected.Name, actualManga.Name)
			}
			if actualManga.Source != expected.Source {
				t.Fatalf("expected manga source %s, got %s", expected.Source, actualManga.Source)
			}
			if actualManga.URL != expected.URL {
				t.Fatalf("expected manga URL %s, got %s", expected.URL, actualManga.URL)
			}
			if actualManga.LastReleasedChapter == nil {
				t.Fatalf("expected manga.LastReleasedChapter to be different than nil")
			}
			if actualManga.CoverImgURL == "" {
				t.Fatalf("expected manga.CoverImgURL to be different than empty")
			}
			if actualManga.CoverImg == nil {
				t.Fatalf("expected manga.CoverImg to be different than nil")
			}
		}
	})
	t.Run("Should not scrape metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			mangaURL := test.url

			_, err := source.GetMangaMetadata(mangaURL+"salt", "", false)
			if err != nil {
				if util.ErrorContains(err, "manga ID not found in the URL") {
					continue
				}
				t.Fatalf("expected error, got %s", err)
			} else {
				t.Fatalf("expected error, got nil")
			}

			re := regexp.MustCompile(`/titles/(\d+)`)
			mangaURL = re.ReplaceAllString(mangaURL, "/titles/000000")
			_, err = source.GetMangaMetadata(mangaURL, "", false)
			if err != nil {
				if util.ErrorContains(err, errordefs.ErrMangaNotFound.Error()) {
					continue
				}
				t.Fatalf("expected error, got %s", err)
			} else {
				t.Fatalf("expected error, got nil")
			}
		}
	})
}

func TestSearch(t *testing.T) {
	source := Source{}

	t.Run("Should search for multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			mangaName := test.expected.Name

			results, err := source.Search(mangaName, 20)
			if err != nil {
				t.Fatalf("error while searching: %v", err)
			}

			if len(results) == 0 {
				t.Fatalf("expected results to be different than 0")
			}
		}
	})
}
