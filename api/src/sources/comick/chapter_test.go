package comick

import (
	"reflect"
	"testing"
	"time"

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
			Chapter:   "155",
			Name:      "Proof of Life",
			URL:       "https://comick.io/comic/claymore/LAqvA",
			UpdatedAt: time.Date(2020, 10, 2, 23, 28, 13, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		url: "https://comick.io/comic/claymore",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "249",
			Name:      "Resolution",
			URL:       "https://comick.io/comic/20th-century-boys/mZGW3",
			UpdatedAt: time.Date(2019, 2, 15, 1, 39, 14, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		url: "https://comick.io/comic/20th-century-boys",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "21",
			Name:      "Ch. 21",
			URL:       "https://comick.io/comic/00-the-horizon/gxAok",
			UpdatedAt: time.Date(2020, 5, 23, 4, 33, 14, 0, time.UTC),
		},
		url: "https://comick.io/comic/00-the-horizon",
	},
}

func TestGetChapterMetadata(t *testing.T) {
	source := Source{}

	t.Run("should get the metadata of a chapter from multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)
			mangaURL := test.url
			chapterURL := expected.URL

			actualChapter, err := source.GetChapterMetadata(mangaURL, "", chapterURL)
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}

			actualChapter, err = source.GetChapterMetadata(mangaURL, expected.Chapter, "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("should not get the metadata of a chapter from multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)
			mangaURL := test.url + "salt"
			chapterURL := expected.URL + "salt"

			actualChapter, err := source.GetChapterMetadata(mangaURL, "", chapterURL)
			if err != nil {
				if !util.ErrorContains(err, "Non-200 status code -> (404)") {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				t.Fatalf("expected error, got nil")
			}

			if reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected actual chapter %v to NOT be deep equal to expected chapter %v", actualChapter, expected)
			}

			actualChapter, err = source.GetChapterMetadata(mangaURL, expected.Chapter, "")
			if err != nil {
				if !util.ErrorContains(err, "Non-200 status code -> (404)") {
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

	t.Run("should get the metadata of the last chapter of multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)
			mangaURL := test.url

			actualChapter, err := source.GetLastChapterMetadata(mangaURL)
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("should not get the metadata of the last chapter of multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)
			mangaURL := test.url + "salt"

			actualChapter, err := source.GetLastChapterMetadata(mangaURL)
			if err != nil {
				if !util.ErrorContains(err, "Non-200 status code -> (404)") {
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
		url:      "https://comick.io/comic/00-the-horizon",
		quantity: 21,
	},
	{
		url:      "https://comick.io/comic/bleach",
		quantity: 714,
	},
	{
		url:      "https://comick.io/comic/00-nisekoi",
		quantity: 237,
	},
}

func TestGetChaptersMetadata(t *testing.T) {
	source := Source{}

	t.Run("should get the metadata of multiple chapters", func(t *testing.T) {
		for _, test := range chaptersTestTable {
			mangaURL := test.url
			expectedQuantity := test.quantity

			chapters, err := source.GetChaptersMetadata(mangaURL)
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
	t.Run("should not get the metadata of multiple chapters", func(t *testing.T) {
		for _, test := range chaptersTestTable {
			mangaURL := test.url + "salt"
			expectedQuantity := 0

			chapters, err := source.GetChaptersMetadata(mangaURL)
			if err != nil {
				if !util.ErrorContains(err, "Non-200 status code -> (404)") {
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
