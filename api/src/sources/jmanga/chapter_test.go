package jmanga

import (
	"reflect"
	"testing"

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
			Chapter: "18",
			Name:    "第18話",
			URL:     "https://jmanga.ltd/read/%E3%82%A2%E3%83%B3%E3%83%81%E3%83%AD%E3%83%9E%E3%83%B3%E3%82%B9/ja/chapter-18-raw/",
		},
		url: "https://jmanga.ltd/read/%E3%82%A2%E3%83%B3%E3%83%81%E3%83%AD%E3%83%9E%E3%83%B3%E3%82%B9-raw/",
	},
	{
		expected: &manga.Chapter{
			Chapter: "13",
			Name:    "第13話",
			URL:     "https://jmanga.ltd/read/デスノート/ja/chapter-13-raw/",
		},
		url: "https://jmanga.ltd/read/%E3%83%87%E3%82%B9%E3%83%8E%E3%83%BC%E3%83%88-raw/",
	},
}

func TestGetChapterMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape the metadata of a chapter from multiple mangas using the chapter URL", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			chapterURL := expected.URL

			actualChapter, err := source.GetChapterMetadata("", "", expected.Chapter, chapterURL, "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should scrape the metadata of a chapter from multiple mangas using the chapter", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
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
			chapterURL := expected.URL + "salt/salt2"
			mangaURL := test.url + "salt/salt2"

			actualChapter, err := source.GetChapterMetadata(mangaURL, "", expected.Chapter, chapterURL, "")
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
			expected := *test.expected
			mangaURL := test.url

			actualChapter, err := source.GetLastChapterMetadata(mangaURL, "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if actualChapter.URL == "" {
				t.Fatalf("expected chapter.URL to be different than ''")
			}

			expected.URL = actualChapter.URL
			if !reflect.DeepEqual(*actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should not scrape the metadata of the last chapter of multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			mangaURL := test.url + "salt/salt2"

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
		url:      "https://jmanga.ltd/read/%E3%82%A2%E3%83%B3%E3%83%81%E3%83%AD%E3%83%9E%E3%83%B3%E3%82%B9-raw/",
		quantity: 18,
	},
	{
		url:      "https://jmanga.ltd/read/%E3%83%87%E3%82%B9%E3%83%8E%E3%83%BC%E3%83%88-raw/",
		quantity: 13,
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
			}
		}
	})
	t.Run("should not scrape the metadata of multiple chapters", func(t *testing.T) {
		for _, test := range chaptersTestTable {
			mangaURL := test.url + "salt/salt2"
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
