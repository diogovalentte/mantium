package mangadex

import (
	"reflect"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

type chapterTestType struct {
	expected   *manga.Chapter
	chapterURL string
	mangaURL   string
}

var chapterTestTable = []chapterTestType{
	{
		expected: &manga.Chapter{
			Chapter:   "155",
			Name:      "Proof of Life",
			URL:       "https://mangadex.org/chapter/87b7b182-e930-4f97-86b5-e243f1645514",
			UpdatedAt: time.Date(2020, 6, 30, 4, 51, 45, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		chapterURL: "https://mangadex.org/chapter/87b7b182-e930-4f97-86b5-e243f1645514",
		mangaURL:   "https://mangadex.org/title/be8fe64b-37da-4fba-b14d-603aba19be1f/claymore",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "249",
			Name:      "Resolution",
			URL:       "https://mangadex.org/chapter/885f6206-7713-4c3d-be91-2e53ac17e2a0",
			UpdatedAt: time.Date(2018, 2, 5, 12, 14, 22, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		chapterURL: "https://mangadex.org/chapter/885f6206-7713-4c3d-be91-2e53ac17e2a0",
		mangaURL:   "https://mangadex.org/title/ad06790a-01e3-400c-a449-0ec152d6756a/20th-century-boys",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "21",
			Name:      "Ch. 21",
			URL:       "https://mangadex.org/chapter/8ed06f7c-f921-44b4-853c-a4cd6d1e840a",
			UpdatedAt: time.Date(2023, 2, 21, 2, 1, 27, 0, time.UTC),
		},
		chapterURL: "https://mangadex.org/chapter/8ed06f7c-f921-44b4-853c-a4cd6d1e840a",
		mangaURL:   "https://mangadex.org/title/67bd081f-1c40-4ae2-95a2-6af29de4fc01/the-horizon",
	},
}

func TestGetChapterMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should get the metadata of a chapter from multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)
			chapterURL := test.chapterURL

			actualChapter, err := source.GetChapterMetadata("", "", "", chapterURL, "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should not get the metadata of a chapter from multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)
			chapterURL := expected.URL
			chapterURL, err := replaceURLID(chapterURL, "00000000-0000-0000-0000-000000000000")
			if err != nil {
				t.Fatalf("error while replacing chapter URL ID: %v", err)
			}

			actualChapter, err := source.GetChapterMetadata("", "", "", chapterURL, "")
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

			actualChapter, err = source.GetChapterMetadata("", "", expected.Chapter, "", "")
			if err != nil {
				if !util.ErrorContains(err, "not implemented") {
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

	t.Run("Should get the metadata of the last chapter of multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)
			mangaURL := test.mangaURL

			actualChapter, err := source.GetLastChapterMetadata(mangaURL, "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should not get the metadata of the last chapter of multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)
			mangaURL := test.mangaURL
			mangaURL, err := replaceURLID(mangaURL, "00000000-0000-0000-0000-000000000000")
			if err != nil {
				t.Fatalf("error while replacing chapter URL ID: %v", err)
			}

			actualChapter, err := source.GetLastChapterMetadata(mangaURL, "")
			if err != nil {
				if !util.ErrorContains(err, errordefs.ErrLastReleasedChapterNotFound.Error()) {
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
		url:      "https://mangadex.org/title/67bd081f-1c40-4ae2-95a2-6af29de4fc01/the-horizon",
		quantity: 21,
	},
	{
		url:      "https://mangadex.org/title/239d6260-d71f-43b0-afff-074e3619e3de/bleach",
		quantity: 704,
	},
	{
		url:      "https://mangadex.org/title/ce16b1c3-d6bb-41e0-8671-d8b065248ba2/nisekoi",
		quantity: 237,
	},
}

func TestGetChaptersMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should get the metadata of multiple chapters", func(t *testing.T) {
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
	t.Run("Should not get the metadata of multiple chapters", func(t *testing.T) {
		for _, test := range chaptersTestTable {
			mangaURL := test.url
			mangaURL, err := replaceURLID(mangaURL, "00000000-0000-0000-0000-000000000000")
			if err != nil {
				t.Fatalf("error while replacing manga URL ID: %v", err)
			}
			expectedQuantity := 0

			chapters, err := source.GetChaptersMetadata(mangaURL, "")
			if err != nil {
				t.Fatalf("expected error, got nil")
			}

			if len(chapters) != expectedQuantity {
				t.Fatalf("expected %v chapters, got %v", expectedQuantity, len(chapters))
			}
		}
	})
}

func TestGetChapterID(t *testing.T) {
	t.Run("Should return the ID of a chapter URL", func(t *testing.T) {
		chapterURL := "https://mangadex.org/chapter/e393167b-573c-414f-8514-f7ff1fc6604d"
		expected := "e393167b-573c-414f-8514-f7ff1fc6604d"
		result, err := getChapterID(chapterURL)
		if err != nil {
			t.Fatalf("error: %s", err)
		}
		if result != expected {
			t.Fatalf("expected %s, got %s", expected, result)
		}
	})
}
