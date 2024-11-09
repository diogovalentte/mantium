package rawkuma

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
			Chapter:   "122",
			Name:      "Chapter 122",
			URL:       "https://rawkuma.com/go-toubun-no-hanayome-chapter-122/",
			UpdatedAt: time.Date(2023, 11, 21, 0, 0, 0, 0, time.UTC),
		}, url: "https://rawkuma.com/manga/go-toubun-no-hanayome",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "139",
			Name:      "Chapter 139",
			URL:       "https://rawkuma.com/shingeki-no-kyojin-chapter-139/",
			UpdatedAt: time.Date(2023, 11, 23, 0, 0, 0, 0, time.UTC),
		},
		url: "https://rawkuma.com/manga/shingeki-no-kyojin",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "60-End",
			Name:      "Chapter 60-End",
			URL:       "https://rawkuma.com/darling-in-the-franxx-chapter-60/",
			UpdatedAt: time.Date(2023, 11, 19, 0, 0, 0, 0, time.UTC),
		},
		url: "https://rawkuma.com/manga/darling-in-the-franxx",
	},
}

func TestGetChapterMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape the metadata of a chapter from multiple mangas", func(t *testing.T) {
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
		url:      "https://rawkuma.com/manga/go-toubun-no-hanayome",
		quantity: 123,
	},
	{
		url:      "https://rawkuma.com/manga/shingeki-no-kyojin",
		quantity: 142,
	},
	{
		url:      "https://rawkuma.com/manga/darling-in-the-franxx",
		quantity: 60,
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
