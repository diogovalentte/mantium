package mangadex

import (
	"reflect"
	"testing"
	"time"

	"github.com/diogovalentte/manga-dashboard-api/src/manga"
)

type chapterTestType struct {
	expected    *manga.Chapter
	chapter_url string
	manga_url   string
}

var chapterTestTable = []chapterTestType{
	{
		expected: &manga.Chapter{
			Number:    155,
			Name:      "Proof of Life",
			URL:       "https://mangadex.org/chapter/87b7b182-e930-4f97-86b5-e243f1645514",
			UpdatedAt: time.Date(2020, 6, 30, 4, 51, 45, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		chapter_url: "https://mangadex.org/chapter/87b7b182-e930-4f97-86b5-e243f1645514",
		manga_url:   "https://mangadex.org/title/be8fe64b-37da-4fba-b14d-603aba19be1f/claymore",
	},
	{
		expected: &manga.Chapter{
			Number:    249,
			Name:      "Resolution",
			URL:       "https://mangadex.org/chapter/885f6206-7713-4c3d-be91-2e53ac17e2a0",
			UpdatedAt: time.Date(2018, 2, 5, 12, 14, 22, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		chapter_url: "https://mangadex.org/chapter/885f6206-7713-4c3d-be91-2e53ac17e2a0",
		manga_url:   "https://mangadex.org/title/ad06790a-01e3-400c-a449-0ec152d6756a/20th-century-boys",
	},
	{
		expected: &manga.Chapter{
			Number:    21,
			Name:      "Ch. 21",
			URL:       "https://mangadex.org/chapter/8ed06f7c-f921-44b4-853c-a4cd6d1e840a",
			UpdatedAt: time.Date(2023, 2, 21, 2, 1, 27, 0, time.UTC),
		},
		chapter_url: "https://mangadex.org/chapter/8ed06f7c-f921-44b4-853c-a4cd6d1e840a",
		manga_url:   "https://mangadex.org/title/67bd081f-1c40-4ae2-95a2-6af29de4fc01/the-horizon",
	},
}

func TestGetChapterMetadata(t *testing.T) {
	t.Run("should scrape metadata of a chapter from multiple mangas", func(t *testing.T) {
		source := Source{}
		for _, test := range chapterTestTable {
			expected := test.expected
			chapterURL := test.chapter_url

			actualChapter, err := source.GetChapterMetadata("", 0, chapterURL)
			if err != nil {
				t.Errorf("error while getting chapter: %v", err)
				return
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Errorf("expected chapter %v, got %v", expected, actualChapter)
				return
			}
		}
	})
}

func TestGetLastChapterMetadata(t *testing.T) {
	t.Run("should scrape metadata of the last chapter of multiple mangas", func(t *testing.T) {
		source := Source{}
		for _, test := range chapterTestTable {
			expected := test.expected
			mangaURL := test.manga_url

			actualChapter, err := source.GetLastChapterMetadata(mangaURL)
			if err != nil {
				t.Errorf("error while getting chapter: %v", err)
				return
			}

			// Compare chapter
			if !reflect.DeepEqual(actualChapter, expected) {
				t.Errorf("expected chapter %v, got %v", expected, actualChapter)
				return
			}
		}
	})
}

type chaptersTestType struct {
	url      string
	quantity int
}

var chaptersTestTable = []chaptersTestType{
	{
		url:      "https://mangadex.org/title/67bd081f-1c40-4ae2-95a2-6af29de4fc01/the-horizon",
		quantity: 21,
	},
	{
		url:      "https://mangadex.org/title/239d6260-d71f-43b0-afff-074e3619e3de/bleach",
		quantity: 704,
	},
	// Nisekoi has a oneshot chapter, the chapter number is "", so I can't parse it
	// TODO: Fix this
	// {
	// 	url:      "https://mangadex.org/title/ce16b1c3-d6bb-41e0-8671-d8b065248ba2/nisekoi",
	// 	quantity: 300,
	// },
}

func TestGetChaptersMetadata(t *testing.T) {
	t.Run("should scrape chapters metadata from multiple mangas", func(t *testing.T) {
		source := Source{}
		for _, test := range chaptersTestTable {
			mangaURL := test.url
			expectedQuantity := test.quantity

			chapters, err := source.GetChaptersMetadata(mangaURL)
			if err != nil {
				t.Errorf("error while getting chapters: %v", err)
				return
			}

			if len(chapters) != expectedQuantity {
				t.Errorf("expected %v chapters, got %v", expectedQuantity, len(chapters))
				return
			}

			for _, chapter := range chapters {
				if chapter.Number == 0 {
					t.Errorf("expected chapter.Chapter to be different than 0")
					return
				}
				if chapter.Name == "" {
					t.Errorf("expected chapter.ChapterName to be different than ''")
					return
				}
				if chapter.URL == "" {
					t.Errorf("expected chapter.URL to be different than ''")
					return
				}
				if chapter.UpdatedAt.IsZero() {
					t.Errorf("expected chapter.UpdatedAt to be different than 0")
					return
				}
			}
		}
	})
}

func TestGetChapterID(t *testing.T) {
	t.Run("should return the ID of a chapter URL", func(t *testing.T) {
		chapterURL := "https://mangadex.org/chapter/e393167b-573c-414f-8514-f7ff1fc6604d"
		expected := "e393167b-573c-414f-8514-f7ff1fc6604d"
		result, err := getChapterID(chapterURL)
		if err != nil {
			t.Errorf("Error: %s", err)
			return
		}
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
			return
		}
	})
}
