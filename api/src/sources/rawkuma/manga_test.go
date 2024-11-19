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
			URL:             "https://rawkuma.com/manga/go-toubun-no-hanayome",
			CoverImgURL:     "https://rawkuma.com/wp-content/uploads/2020/06/Go-Toubun-no-Hanayome-cover.jpg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter:   "122",
				Name:      "Chapter 122",
				URL:       "https://rawkuma.com/go-toubun-no-hanayome-chapter-122/",
				UpdatedAt: time.Date(2023, 11, 21, 0, 0, 0, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://rawkuma.com/manga/go-toubun-no-hanayome",
	},
	{
		expected: &manga.Manga{
			Name:            "Shingeki no Kyojin",
			Source:          "rawkuma",
			URL:             "https://rawkuma.com/manga/shingeki-no-kyojin",
			CoverImgURL:     "https://rawkuma.com/wp-content/uploads/2020/07/Shingeki-no-Kyojin-33.jpg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter:   "139",
				Name:      "Chapter 139",
				URL:       "https://rawkuma.com/shingeki-no-kyojin-chapter-139/",
				UpdatedAt: time.Date(2023, 11, 23, 0, 0, 0, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://rawkuma.com/manga/shingeki-no-kyojin",
	},
	{
		expected: &manga.Manga{
			Name:            "Darling in the FRANXX",
			Source:          "rawkuma",
			URL:             "https://rawkuma.com/manga/darling-in-the-franxx",
			CoverImgURL:     "https://rawkuma.com/wp-content/uploads/2020/02/Darling-in-the-FRANXX.jpg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter:   "60-End",
				Name:      "Chapter 60-End",
				URL:       "https://rawkuma.com/darling-in-the-franxx-chapter-60/",
				UpdatedAt: time.Date(2023, 11, 19, 0, 0, 0, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://rawkuma.com/manga/darling-in-the-franxx",
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
		}
	})
}
