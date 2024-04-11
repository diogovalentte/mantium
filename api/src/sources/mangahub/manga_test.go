package mangahub

import (
	"reflect"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/manga"
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
			CoverImgURL:     "https://thumb.mangahub.io/mn/death-note.jpg",
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
			CoverImgURL:     "https://thumb.mangahub.io/mn/vagabond.jpg",
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
			CoverImgURL:     "https://thumb.mangahub.io/mn/mob-psycho-100.jpg",
			CoverImgResized: true,
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
