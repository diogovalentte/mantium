package comick

import (
	"reflect"
	"testing"
	"time"

	"github.com/diogovalentte/manga-dashboard-api/api/src/manga"
)

type chapterTestType struct {
	expected *manga.Chapter
	url      string
}

var chapterTestTable = []chapterTestType{
	{
		expected: &manga.Chapter{
			Chapter:   "155",
			Name:      "Proof of Life",
			URL:       "https://comick.cc/comic/claymore/LAqvA",
			UpdatedAt: time.Date(2020, 10, 2, 23, 28, 13, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		url: "https://comick.cc/comic/claymore",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "249",
			Name:      "Resolution",
			URL:       "https://comick.cc/comic/20th-century-boys/mZGW3",
			UpdatedAt: time.Date(2019, 2, 15, 1, 39, 14, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		url: "https://comick.cc/comic/20th-century-boys",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "21",
			Name:      "Ch. 21",
			URL:       "https://comick.cc/comic/00-the-horizon/gxAok",
			UpdatedAt: time.Date(2020, 5, 23, 4, 33, 14, 0, time.UTC),
		},
		url: "https://comick.cc/comic/00-the-horizon",
	},
}

func TestGetChapterMetadata(t *testing.T) {
	t.Run("should scrape metadata of a chapter from multiple mangas", func(t *testing.T) {
		source := Source{}
		for _, test := range chapterTestTable {
			expected := test.expected
			mangaURL := test.url

			actualChapter, err := source.GetChapterMetadata(mangaURL, expected.Chapter, "")
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

func TestGetLastChapterMetadata(t *testing.T) {
	t.Run("should scrape metadata of the last chapter of multiple mangas", func(t *testing.T) {
		source := Source{}
		for _, test := range chapterTestTable {
			expected := test.expected
			mangaURL := test.url

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
		url:      "https://comick.cc/comic/00-the-horizon",
		quantity: 21,
	},
	{
		url:      "https://comick.cc/comic/bleach",
		quantity: 700,
	},
	{
		url:      "https://comick.cc/comic/00-nisekoi",
		quantity: 233,
	},
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
				if chapter.Chapter == "" {
					t.Errorf("expected chapter.Chapter to be different than ''")
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
