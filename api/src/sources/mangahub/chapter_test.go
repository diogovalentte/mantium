package mangahub

import (
	"reflect"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
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
			URL:       "https://mangahub.io/chapter/claymore_116/interview-with-yagi-norihiro-extended",
			UpdatedAt: time.Date(2016, 1, 20, 0, 0, 0, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		url: "https://mangahub.io/manga/claymore_116",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "249",
			Name:      "[End]",
			URL:       "https://mangahub.io/chapter/20th-century-boys_116/end",
			UpdatedAt: time.Date(2016, 1, 20, 0, 0, 0, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		url: "https://mangahub.io/manga/20th-century-boys_116",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "21",
			Name:      "Chapter 21",
			URL:       "https://mangahub.io/chapter/the-horizon/chapter-21",
			UpdatedAt: time.Date(2020, 5, 10, 6, 31, 4, 0, time.UTC),
		},
		url: "https://mangahub.io/manga/the-horizon",
	},
}

func TestGetChapterMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape the metadata of a chapter from multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)
			mangaURL := test.url

			actualChapter, err := source.GetChapterMetadata(mangaURL, "", expected.Chapter, "", "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should not scrape the metadata of a chapter from multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			mangaURL := test.url + "salt"

			actualChapter, err := source.GetChapterMetadata(mangaURL, "", expected.Chapter, "", "")
			if err != nil {
				if !util.ErrorContains(err, errordefs.ErrMangaNotFound.Error()) {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				t.Fatalf("expected error, got nil")
			}

			if reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected actual chapter %s to NOT be deep equal to expected chapter %s", actualChapter, expected)
			}
		}
	})
}

func TestGetLastChapterMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape the metadata of the last chapter of multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)
			mangaURL := test.url

			actualChapter, err := source.GetLastChapterMetadata(mangaURL, "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should not scrape the metadata of the last chapter of multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			mangaURL := test.url + "salt"

			actualChapter, err := source.GetLastChapterMetadata(mangaURL, "")
			if err != nil {
				if !util.ErrorContains(err, errordefs.ErrMangaNotFound.Error()) {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				t.Fatalf("expected error, got nil")
			}

			if reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected actual chapter %s to NOT be deep equal to expected chapter %s", actualChapter, expected)
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
		quantity: 31,
	},
}

func TestGetChaptersMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape the metadata of multiple chapters", func(t *testing.T) {
		for _, test := range chaptersTestTable {
			mangaURL := test.url
			expectedQuantity := test.quantity

			chapters, err := source.GetChaptersMetadata(mangaURL, "")
			if err != nil {
				t.Fatalf("error while getting chapters: %v", err)
			}

			if len(chapters) != expectedQuantity {
				t.Fatalf("expected %v chapters, got %v", expectedQuantity, len(chapters))
			}

			for _, chapter := range chapters {
				if chapter.Chapter == "" {
					t.Fatalf("expected chapter.Chapter to be different than ''")
				}
				if chapter.Name == "" {
					t.Fatalf("expected chapter.ChapterName to be different than ''")
				}
				if chapter.URL == "" {
					t.Fatalf("expected chapter.URL to be different than ''")
				}
				if chapter.UpdatedAt.IsZero() {
					t.Fatalf("expected chapter.UpdatedAt to be different than 0")
				}
			}
		}
	})
	t.Run("should not scrape the metadata of multiple chapters", func(t *testing.T) {
		for _, test := range chaptersTestTable {
			mangaURL := test.url + "salt"
			expectedQuantity := 0

			chapters, err := source.GetChaptersMetadata(mangaURL, "")
			if err != nil {
				if !util.ErrorContains(err, errordefs.ErrMangaNotFound.Error()) {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				t.Fatalf("expected error, got nil")
			}

			if len(chapters) != expectedQuantity {
				t.Fatalf("expected %v chapters, got %v", expectedQuantity, len(chapters))
			}
		}
	})
}
