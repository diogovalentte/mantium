package mangahub

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
			Source:          "mangahub",
			URL:             "https://mangahub.io/manga/death-note_119",
			CoverImgURL:     "https://thumb.mghcdn.com/mn/death-note.jpg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter:   "112",
				Name:      "Chapter 112",
				URL:       "https://mangahub.io/chapter/death-note_119/chapter-112",
				UpdatedAt: time.Date(2018, 6, 16, 5, 30, 12, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://mangahub.io/manga/death-note_119",
	},
	{
		expected: &manga.Manga{
			Name:            "Vagabond",
			Source:          "mangahub",
			URL:             "https://mangahub.io/manga/vagabond_119",
			CoverImgURL:     "https://thumb.mghcdn.com/mn/vagabond.jpg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter:   "327",
				Name:      "The Man Named Tadaoki",
				URL:       "https://mangahub.io/chapter/vagabond_119/chapter-327",
				UpdatedAt: time.Date(2016, 1, 20, 0, 0, 0, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
				Type:      1,
			},
		},
		url: "https://mangahub.io/manga/vagabond_119",
	},
	{
		expected: &manga.Manga{
			Name:            "Mob Psycho 100",
			Source:          "mangahub",
			URL:             "https://mangahub.io/manga/mob-psycho-100",
			CoverImgURL:     "https://thumb.mghcdn.com/mn/mob-psycho-100.jpg",
			CoverImgResized: false,
			LastReleasedChapter: &manga.Chapter{
				Chapter:   "101",
				Name:      "101[END]",
				URL:       "https://mangahub.io/chapter/mob-psycho-100/chapter-101",
				UpdatedAt: time.Date(2018, 4, 26, 3, 57, 50, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://mangahub.io/manga/mob-psycho-100",
	},
}

func TestGetMangaMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			expected := test.expected
			expected.LastReleasedChapter.UpdatedAt = expected.LastReleasedChapter.UpdatedAt.In(time.Local)
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
				if util.ErrorContains(err, errordefs.ErrMangaAttributesNotFound.Error()) {
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
