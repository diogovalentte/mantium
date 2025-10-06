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
	expected   *manga.Chapter
	url        string
	internalID string
}

var chapterTestTable = []chapterTestType{
	{
		expected: &manga.Chapter{
			Chapter:   "122",
			Name:      "Chapter 122",
			URL:       "https://rawkuma.net/manga/go-toubun-no-hanayome/chapter-122.27772/",
			UpdatedAt: time.Date(2025, 9, 14, 10, 34, 51, 0, time.UTC),
		},
		url:        "https://rawkuma.net/manga/go-toubun-no-hanayome",
		internalID: "742",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "139",
			Name:      "Chapter 139",
			URL:       "https://rawkuma.net/manga/shingeki-no-kyojin/chapter-139.49406/",
			UpdatedAt: time.Date(2025, 9, 17, 3, 57, 30, 0, time.UTC),
		},
		url:        "https://rawkuma.net/manga/shingeki-no-kyojin",
		internalID: "1270",
	},
	{
		expected: &manga.Chapter{
			Chapter:   "60",
			Name:      "Chapter 60",
			URL:       "https://rawkuma.net/manga/darling-in-the-franxx/chapter-60.14004/",
			UpdatedAt: time.Date(2025, 9, 12, 2, 43, 46, 0, time.UTC),
		},
		url:        "https://rawkuma.net/manga/darling-in-the-franxx",
		internalID: "360",
	},
}

func TestGetChapterMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape the metadata of a chapter from multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			mangaURL := test.url

			actualChapter, err := source.GetChapterMetadata(mangaURL, "", expected.Chapter, expected.URL, "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}
			actualChapter.UpdatedAt = actualChapter.UpdatedAt.UTC()

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should not scrape the metadata of a chapter from multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			chapterURL := expected.URL + "salt"

			actualChapter, err := source.GetChapterMetadata("", "", expected.Chapter, chapterURL, "")
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
			expected := test.expected
			mangaID := test.internalID

			actualChapter, err := source.GetLastChapterMetadata("", mangaID)
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}
			actualChapter.UpdatedAt = actualChapter.UpdatedAt.UTC()

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should not scrape the metadata of the last chapter of multiple mangas", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			mangaID := test.internalID + "salt"

			actualChapter, err := source.GetLastChapterMetadata("", mangaID)
			if err != nil {
				if !util.ErrorContains(err, errordefs.ErrMangaHasNoIDOrURL.Error()) {
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
	url        string
	quantity   int
	internalID string
}

var chaptersTestTable = []chaptersTestType{
	{
		url:        "https://rawkuma.net/manga/go-toubun-no-hanayome",
		quantity:   122,
		internalID: "742",
	},
	{
		url:        "https://rawkuma.net/manga/shingeki-no-kyojin",
		quantity:   59,
		internalID: "1270",
	},
	{
		url:        "https://rawkuma.net/manga/darling-in-the-franxx",
		quantity:   60,
		internalID: "360",
	},
}

func TestGetChaptersMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape the metadata of multiple chapters using internal ID", func(t *testing.T) {
		for _, test := range chaptersTestTable {
			mangaID := test.internalID
			expectedQuantity := test.quantity

			chapters, err := source.GetChaptersMetadata("", mangaID)
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
	t.Run("Should scrape the metadata of multiple chapters using mangaURL", func(t *testing.T) {
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
		for range chaptersTestTable {
			expectedQuantity := 0

			chapters, err := source.GetChaptersMetadata("", "salt")
			if err != nil {
				if !util.ErrorContains(err, errordefs.ErrMangaHasNoIDOrURL.Error()) {
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
