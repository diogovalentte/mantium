package mangaupdates

import (
	"reflect"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/manga"
)

type chapterTestType struct {
	expected        *manga.Chapter
	mangaInternalID string
	mangaURL        string
}

var chapterTestTable = []chapterTestType{
	{
		expected: &manga.Chapter{
			Chapter:    "108",
			Name:       "Death Note",
			URL:        "https://www.mangaupdates.com/releases.html?stype=series&search=3479935384",
			InternalID: "681314",
			UpdatedAt:  time.Date(2021, 1, 7, 0, 0, 0, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		mangaInternalID: "3479935384",
		mangaURL:        "https://www.mangaupdates.com/series/1ljv3bs/death-note",
	},
	{
		expected: &manga.Chapter{
			Chapter:    "113",
			Name:       "Yotsubato!",
			URL:        "https://www.mangaupdates.com/releases.html?stype=series&search=23606352927",
			InternalID: "1040674",
			UpdatedAt:  time.Date(2024, 8, 18, 0, 0, 0, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		mangaInternalID: "23606352927",
		mangaURL:        "https://www.mangaupdates.com/series/auem2hr/yotsubato",
	},
}

func TestGetChapterMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should get the metadata of a chapter from multiple mangas using manga and chapter internal IDs", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)

			actualChapter, err := source.GetChapterMetadata("", test.mangaInternalID, "", "", test.expected.InternalID)
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should get the metadata of a chapter from multiple mangas using manga URL and chapter internal ID", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)

			actualChapter, err := source.GetChapterMetadata(test.mangaURL, "", "", "", test.expected.InternalID)
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should get the metadata of a chapter from multiple mangas using manga internal ID and chapter", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)

			actualChapter, err := source.GetChapterMetadata("", test.mangaInternalID, expected.Chapter, "", "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should get the metadata of a chapter from multiple mangas using manga URL and chapter", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)

			actualChapter, err := source.GetChapterMetadata(test.mangaURL, "", expected.Chapter, "", "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
}

func TestGetLastChapterMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should get the metadata of the last chapter from multiple mangas using manga internal ID", func(t *testing.T) {
		for _, test := range chapterTestTable {
			actualChapter, err := source.GetLastChapterMetadata("", test.mangaInternalID)
			if err != nil {
				t.Fatalf("error while getting last chapter: %v", err)
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
	t.Run("Should get the metadata of the last chapter from multiple mangas using manga URL", func(t *testing.T) {
		for _, test := range chapterTestTable {
			actualChapter, err := source.GetLastChapterMetadata(test.mangaURL, "")
			if err != nil {
				t.Fatalf("error while getting last chapter: %v", err)
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
	t.Run("Should get the metadata of a chapter from multiple mangas using manga internal ID and chapter", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)

			actualChapter, err := source.GetChapterMetadata("", test.mangaInternalID, expected.Chapter, "", "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
	t.Run("Should get the metadata of a chapter from multiple mangas using manga URL and chapter", func(t *testing.T) {
		for _, test := range chapterTestTable {
			expected := test.expected
			expected.UpdatedAt = expected.UpdatedAt.In(time.Local)

			actualChapter, err := source.GetChapterMetadata(test.mangaURL, "", expected.Chapter, "", "")
			if err != nil {
				t.Fatalf("error while getting chapter: %v", err)
			}

			if !reflect.DeepEqual(actualChapter, expected) {
				t.Fatalf("expected chapter %s, got %s", expected, actualChapter)
			}
		}
	})
}

type chaptersTestType struct {
	mangaInternalID string
	mangaURL        string
}

var chaptersTestTable = []chaptersTestType{
	{
		mangaInternalID: "3479935384",
		mangaURL:        "https://www.mangaupdates.com/series/1ljv3bs/death-note",
	},
	{
		mangaInternalID: "23606352927",
		mangaURL:        "https://www.mangaupdates.com/series/auem2hr/yotsubato",
	},
}

func TestGetChaptersMetadata(t *testing.T) {
	source := Source{}

	t.Run("should get the metadata of multiple chapters using manga internal ID", func(t *testing.T) {
		for _, test := range chaptersTestTable {
			chapters, err := source.GetChaptersMetadata("", test.mangaInternalID)
			if err != nil {
				t.Fatalf("error while getting chapters: %v", err)
			}

			if len(chapters) == 0 {
				t.Fatalf("expected results to be different than 0")
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
	t.Run("should get the metadata of multiple chapters using manga URL", func(t *testing.T) {
		for _, test := range chaptersTestTable {
			chapters, err := source.GetChaptersMetadata(test.mangaURL, "")
			if err != nil {
				t.Fatalf("error while getting chapters: %v", err)
			}

			if len(chapters) == 0 {
				t.Fatalf("expected results to be different than 0")
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
}
