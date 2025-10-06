package rawkuma

import (
	"reflect"
	"testing"
	"time"

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
			Name:            "Go-Toubun no Hanayome",
			Source:          "rawkuma",
			URL:             "https://rawkuma.net/manga/go-toubun-no-hanayome",
			CoverImgURL:     "https://rawkuma.net/wp-content/uploads/2025/09/i360331.jpg",
			CoverImgResized: true,
			InternalID:      "742",
			LastReleasedChapter: &manga.Chapter{
				Chapter:   "122",
				Name:      "Chapter 122",
				URL:       "https://rawkuma.net/manga/go-toubun-no-hanayome/chapter-122.27772/",
				UpdatedAt: time.Date(2025, 9, 14, 10, 34, 51, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://rawkuma.net/manga/go-toubun-no-hanayome",
	},
	{
		expected: &manga.Manga{
			Name:            "Shingeki no Kyojin",
			Source:          "rawkuma",
			URL:             "https://rawkuma.net/manga/shingeki-no-kyojin",
			CoverImgURL:     "https://rawkuma.net/wp-content/uploads/2025/09/i424594.png",
			CoverImgResized: true,
			InternalID:      "1270",
			LastReleasedChapter: &manga.Chapter{
				Chapter:   "139",
				Name:      "Chapter 139",
				URL:       "https://rawkuma.net/manga/shingeki-no-kyojin/chapter-139.49406/",
				UpdatedAt: time.Date(2025, 9, 17, 3, 57, 30, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://rawkuma.net/manga/shingeki-no-kyojin",
	},
	{
		expected: &manga.Manga{
			Name:            "Darling in the Franxx",
			Source:          "rawkuma",
			URL:             "https://rawkuma.net/manga/darling-in-the-franxx",
			CoverImgURL:     "https://rawkuma.net/wp-content/uploads/2025/09/i310814.jpg",
			CoverImgResized: true,
			InternalID:      "360",
			LastReleasedChapter: &manga.Chapter{
				Chapter:   "60",
				Name:      "Chapter 60",
				URL:       "https://rawkuma.net/manga/darling-in-the-franxx/chapter-60.14004/",
				UpdatedAt: time.Date(2025, 9, 12, 2, 43, 46, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://rawkuma.net/manga/darling-in-the-franxx",
	},
}

func TestGetMangaMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			expected := test.expected
			mangaURL := test.url

			actualManga, err := source.GetMangaMetadata(mangaURL, "")
			if err != nil {
				t.Fatalf("error while getting manga: %v", err)
			}

			if actualManga.CoverImg == nil {
				t.Fatalf("expected manga.CoverImg to be different than nil")
			}
			actualManga.CoverImg = nil
			actualManga.LastReleasedChapter.UpdatedAt = actualManga.LastReleasedChapter.UpdatedAt.UTC()

			if !reflect.DeepEqual(actualManga, expected) {
				t.Fatalf("expected manga %s, got %s", expected, actualManga)
			}
		}
	})
	t.Run("Should not scrape metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			mangaURL := test.url + "salt"

			_, err := source.GetMangaMetadata(mangaURL, "")
			if err != nil {
				if util.ErrorContains(err, errordefs.ErrMangaNotFound.Error()) {
					continue
				}
				t.Fatalf("expected error, got %s", err)
			}
			t.Fatalf("expected error, got nil")
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

			for _, result := range results {
				if result.Name == "" {
					t.Fatalf("expected result.Name to be different than empty")
				}
				if result.URL == "" {
					t.Fatalf("expected result.URL to be different than empty")
				}
			}
		}
	})
}
