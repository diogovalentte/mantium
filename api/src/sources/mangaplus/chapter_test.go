package mangaplus

import (
	"reflect"
	"regexp"
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

// Manga Plus will show the page in the language of your country if it has a version of the manga in that language.
// This will also change the chapters info, which can affect the tests.
// The tests below are in PT-BR or English.
var chapterTestTable = []chapterTestType{
	{
		expected: &manga.Chapter{
			Chapter:   "1",
			Name:      "Capítulo 1: Romance Dawn",
			URL:       "https://mangaplus.shueisha.co.jp/viewer/1009174",
			UpdatedAt: time.Date(2021, 4, 11, 16, 0, 0, 0, time.UTC),
		},
		url: "https://mangaplus.shueisha.co.jp/titles/100149",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "2",
			Name:      "Missão: 2",
			URL:       "https://mangaplus.shueisha.co.jp/viewer/1009194",
			UpdatedAt: time.Date(2021, 4, 11, 15, 0, 0, 0, time.UTC),
		},
		url: "https://mangaplus.shueisha.co.jp/titles/100151",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "3",
			Name:      "Capítulo 3: Chegada em Tóquio",
			URL:       "https://mangaplus.shueisha.co.jp/viewer/5000069",
			UpdatedAt: time.Date(2022, 7, 12, 15, 0, 0, 0, time.UTC),
		},
		url: "https://mangaplus.shueisha.co.jp/titles/500001",
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
	t.Run("Should not scrape the metadata of a chapter from multiple mangas with wrong URL", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			mangaURL := test.url + "salt"

			actualChapter, err := source.GetChapterMetadata(mangaURL, "", expected.Chapter, "", "")
			if err != nil {
				if !util.ErrorContains(err, "manga ID not found in the URL") {
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
	t.Run("Should not scrape the metadata of a chapter from multiple mangas with wrong chapter", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			mangaURL := test.url

			actualChapter, err := source.GetChapterMetadata(mangaURL, "", expected.Chapter+"salt", "", "")
			if err != nil {
				if !util.ErrorContains(err, errordefs.ErrChapterNotFound.Error()) {
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
			mangaURL := test.url

			actualChapter, err := source.GetLastChapterMetadata(mangaURL, "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if actualChapter.Chapter == "" {
				t.Fatalf("expected chapter.Chapter to be different than ''")
			}
			if actualChapter.Name == "" {
				t.Fatalf("expected chapter.ChapterName to be different than ''")
			}
			if actualChapter.URL == "" {
				t.Fatalf("expected chapter.URL to be different than ''")
			}
			if actualChapter.UpdatedAt.IsZero() {
				t.Fatalf("expected chapter.UpdatedAt to be different than 0")
			}
		}
	})
	t.Run("Should not scrape the metadata of the last chapter of multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			mangaURL := test.url

			_, err := source.GetLastChapterMetadata(mangaURL+"salt", "")
			if err != nil {
				if util.ErrorContains(err, "manga ID not found in the URL") {
					continue
				}
				t.Fatalf("expected error, got %s", err)
			} else {
				t.Fatalf("expected error, got nil")
			}

			re := regexp.MustCompile(`/titles/(\d+)`)
			mangaURL = re.ReplaceAllString(mangaURL, "/titles/000000")
			_, err = source.GetMangaMetadata(mangaURL, "")
			if err != nil {
				if util.ErrorContains(err, errordefs.ErrMangaNotFound.Error()) {
					continue
				}
				t.Fatalf("expected error, got %s", err)
			} else {
				t.Fatalf("expected error, got nil")
			}
		}
	})
}

func TestGetChaptersMetadata(t *testing.T) {
	source := Source{}
	expectedQuantity := 6

	t.Run("Should scrape the metadata of multiple chapters", func(t *testing.T) {
		for _, test := range chapterTestTable {
			mangaURL := test.url

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
		for _, test := range chapterTestTable {
			mangaURL := test.url + "salt"
			expectedQuantity := 0

			chapters, err := source.GetChaptersMetadata(mangaURL, "")
			if err != nil {
				if !util.ErrorContains(err, "manga ID not found in the URL") {
					t.Fatalf("expected error, got %s", err)
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
