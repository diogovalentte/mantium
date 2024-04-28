package mangahub

import (
	"reflect"
	"testing"
	"time"

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
			Source:          "mangahub.io",
			URL:             "https://mangahub.io/manga/death-note_119",
			CoverImgURL:     "https://thumb.mghcdn.com/mn/death-note.jpg",
			CoverImgResized: true,
			LastUploadChapter: &manga.Chapter{
				Chapter:   "112",
				Name:      "Chapter 112",
				URL:       "https://mangahub.io/chapter/death-note_119/chapter-112",
				UpdatedAt: time.Date(2018, 6, 16, 0, 0, 0, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://mangahub.io/manga/death-note_119",
	},
	{
		expected: &manga.Manga{
			Name:            "Vagabond",
			Source:          "mangahub.io",
			URL:             "https://mangahub.io/manga/vagabond_119",
			CoverImgURL:     "https://thumb.mghcdn.com/mn/vagabond.jpg",
			CoverImgResized: true,
			LastUploadChapter: &manga.Chapter{
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
			Source:          "mangahub.io",
			URL:             "https://mangahub.io/manga/mob-psycho-100",
			CoverImgURL:     "https://thumb.mghcdn.com/mn/mob-psycho-100.jpg",
			CoverImgResized: false,
			LastUploadChapter: &manga.Chapter{
				Chapter:   "101",
				Name:      "101[END]",
				URL:       "https://mangahub.io/chapter/mob-psycho-100/chapter-101",
				UpdatedAt: time.Date(2018, 4, 26, 0, 0, 0, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://mangahub.io/manga/mob-psycho-100",
	},
}

func TestGetMangaMetadata(t *testing.T) {
	source := Source{}

	t.Run("should scrape metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			expected := test.expected
			mangaURL := test.url

			actualManga, err := source.GetMangaMetadata(mangaURL)
			if err != nil {
				t.Errorf("error while getting manga: %v", err)
				return
			}

			if actualManga.CoverImg == nil {
				t.Errorf("expected manga.CoverImg to be different than nil")
				return
			}
			actualManga.CoverImg = nil

			if !reflect.DeepEqual(actualManga, expected) {
				t.Errorf("expected manga %s, got %s", expected, actualManga)
				return
			}
		}
	})
	t.Run("should not scrape metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			mangaURL := test.url + "salt"

			_, err := source.GetMangaMetadata(mangaURL)
			if err != nil {
				if util.ErrorContains(err, "Manga not found") {
					continue
				}
				t.Errorf("expected error, got %s", err)
				return
			}
			t.Errorf("expected error, got nil")
			return
		}
	})
}
