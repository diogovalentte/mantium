package comick

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
			Name:            "Death Note",
			Source:          "comick.xyz",
			URL:             "https://comick.io/comic/death-note",
			CoverImgURL:     "https://meo.comick.pictures/a0yXD.jpg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
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
			LastReleasedChapter: &manga.Chapter{
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
			LastReleasedChapter: &manga.Chapter{
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
	source := Source{}

	t.Run("Should get the metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			expected := test.expected
			expected.LastReleasedChapter.UpdatedAt = expected.LastReleasedChapter.UpdatedAt.In(time.Local)
			mangaURL := test.url

			actualManga, err := source.GetMangaMetadata(mangaURL, "", false)
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
	t.Run("Should not get the metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			mangaURL := test.url + "salt"

			_, err := source.GetMangaMetadata(mangaURL, "", false)
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

var getMangaIDTestTable = []string{
	"https://comick.io/comic/00-jujutsu-kaisen/",
	"https://comick.io/comic/00-jujutsu-kaisen",
}

func TestGetMangaID(t *testing.T) {
	t.Run("Should return the ID of a manga URL", func(t *testing.T) {
		for _, mangaURL := range getMangaIDTestTable {
			expected := "00-jujutsu-kaisen"
			result, err := getMangaSlug(mangaURL)
			if err != nil {
				t.Fatal(err)
			}
			if result != expected {
				t.Fatalf("expected %s, got %s", expected, result)
			}
		}
	})
}
