package comick

import (
	"reflect"
	"testing"
	"time"

	"github.com/diogovalentte/manga-dashboard-api/api/src/manga"
)

type mangaTestType struct {
	expected *manga.Manga
	url      string
}

var mangasTestTable = []mangaTestType{
	{
		expected: &manga.Manga{
			Name:            "Death Note",
			Source:          "comick.xyz",
			URL:             "https://comick.io/comic/death-note",
			CoverImgURL:     "https://meo.comick.pictures/a0yXD.jpg",
			CoverImgResized: true,
			LastUploadChapter: &manga.Chapter{
				Chapter:   "110",
				Name:      "Ch. 110",
				URL:       "https://comick.io/comic/death-note/0MvzG",
				UpdatedAt: time.Date(2021, 4, 11, 5, 45, 16, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://comick.io/comic/death-note",
	},
	{
		expected: &manga.Manga{
			Name:            "Vagabond",
			Source:          "comick.xyz",
			URL:             "https://comick.io/comic/00-vagabond",
			CoverImgURL:     "https://meo.comick.pictures/marne.jpg",
			CoverImgResized: true,
			LastUploadChapter: &manga.Chapter{
				Chapter:   "327",
				Name:      "The Man Named Tadoki",
				URL:       "https://comick.io/comic/00-vagabond/ADgKl",
				UpdatedAt: time.Date(2019, 2, 15, 1, 49, 59, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
				Type:      1,
			},
		},
		url: "https://comick.io/comic/00-vagabond",
	},
	{
		expected: &manga.Manga{
			Name:            "Mob Psycho 100",
			Source:          "comick.xyz",
			URL:             "https://comick.io/comic/mob-psycho-100",
			CoverImgURL:     "https://meo.comick.pictures/NR1xz.jpg",
			CoverImgResized: true,
			LastUploadChapter: &manga.Chapter{
				Chapter:   "101",
				Name:      "101",
				URL:       "https://comick.io/comic/mob-psycho-100/Ro7Lw",
				UpdatedAt: time.Date(2019, 2, 15, 8, 9, 33, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://comick.io/comic/mob-psycho-100",
	},
}

func TestGetMangaMetadata(t *testing.T) {
	t.Run("should scrape metadata from multiple mangas", func(t *testing.T) {
		source := Source{}
		for _, test := range mangasTestTable {
			expected := test.expected
			mangaURL := test.url

			actualManga, err := source.GetMangaMetadata(mangaURL)
			if err != nil {
				t.Errorf("error while getting manga: %v", err)
				return
			}

			// Cover img
			if actualManga.CoverImg == nil {
				t.Errorf("expected manga.CoverImg to be different than nil")
				return
			}
			actualManga.CoverImg = nil

			// Compare manga
			if !reflect.DeepEqual(actualManga, expected) {
				t.Errorf("expected manga %v, got %v", expected, actualManga)
				t.Errorf("expected manga.LastChapter %v, got %v", expected.LastUploadChapter, actualManga.LastUploadChapter)
				return
			}
		}
	})
}

var getMangaIDTestTable = []string{
	"https://comick.io/comic/00-jujutsu-kaisen/",
	"https://comick.io/comic/00-jujutsu-kaisen",
}

func TestGetMangaID(t *testing.T) {
	t.Run("should return the ID of a manga URL", func(t *testing.T) {
		for _, mangaURL := range getMangaIDTestTable {
			expected := "00-jujutsu-kaisen"
			result, err := getMangaSlug(mangaURL)
			if err != nil {
				t.Errorf("Error: %s", err)
				return
			}
			if result != expected {
				t.Errorf("Expected %s, got %s", expected, result)
				return
			}
		}
	})
}
