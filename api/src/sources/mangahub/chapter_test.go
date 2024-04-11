package mangahub

import (
	"reflect"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/manga"
)

type chapterTestType struct {
	expected *manga.Chapter
	url      string
}

var chapterTestTable = []chapterTestType{
	{
		expected: &manga.Chapter{
			Chapter:   "155.2",
			Name:      "Interview with Yagi Norihiro Extended",
			URL:       "https://mangahub.io/chapter/claymore_116/chapter-155.2",
			UpdatedAt: time.Date(2016, 1, 20, 0, 0, 0, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		url: "https://mangahub.io/manga/claymore_116",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "249",
			Name:      "[End]",
			URL:       "https://mangahub.io/chapter/20th-century-boys_116/chapter-249",
			UpdatedAt: time.Date(2016, 1, 20, 0, 0, 0, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		url: "https://mangahub.io/manga/20th-century-boys_116",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "21",
			Name:      "The Horizon 21",
			URL:       "https://mangahub.io/chapter/the-horizon/chapter-21",
			UpdatedAt: time.Date(2020, 5, 10, 0, 0, 0, 0, time.UTC),
		},
		url: "https://mangahub.io/manga/the-horizon",
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
		url:      "https://mangahub.io/manga/the-horizon",
		quantity: 24,
	},
	{
		url:      "https://mangahub.io/manga/vibration-man",
		quantity: 28,
	},
	{
		url:      "https://mangahub.io/manga/ayaka-chan-wa-hiroko-senpai-ni-koishiteru",
		quantity: 30,
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
				t.Errorf("expected 22 chapters, got %v", len(chapters))
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
