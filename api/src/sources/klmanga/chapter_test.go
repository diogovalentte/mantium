package klmanga

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
			Name:    "アンチロマンス (Raw – Free) 【第18話】",
			URL:     "https://klmanga.lt/manga-raw/%e3%82%a2%e3%83%b3%e3%83%81%e3%83%ad%e3%83%9e%e3%83%b3%e3%82%b9-raw-free/chapter-18/",
		},
		url: "https://klmanga.lt/manga-raw/%e3%82%a2%e3%83%b3%e3%83%81%e3%83%ad%e3%83%9e%e3%83%b3%e3%82%b9-raw-free/",
	},
	{
		expected: &manga.Chapter{
			Chapter: "8",
			Name:    "アイマイミーマイン (Raw – Free) 【第8話】",
			URL:     "https://klmanga.lt/manga-raw/%e3%82%a2%e3%82%a4%e3%83%9e%e3%82%a4%e3%83%9f%e3%83%bc%e3%83%9e%e3%82%a4%e3%83%b3-raw-free/chapter-8/",
		},
		url: "https://klmanga.lt/manga-raw/%e3%82%a2%e3%82%a4%e3%83%9e%e3%82%a4%e3%83%9f%e3%83%bc%e3%83%9e%e3%82%a4%e3%83%b3-raw-free/",
	},
	{
		expected: &manga.Chapter{
			Chapter: "20",
			Name:    "思えば遠くにオブスクラ (Raw – Free) 【第20話】",
			URL:     "https://klmanga.lt/manga-raw/%e6%80%9d%e3%81%88%e3%81%b0%e9%81%a0%e3%81%8f%e3%81%ab%e3%82%aa%e3%83%96%e3%82%b9%e3%82%af%e3%83%a9-raw-free/chapter-20/",
		},
		url: "https://klmanga.lt/manga-raw/%e6%80%9d%e3%81%88%e3%81%b0%e9%81%a0%e3%81%8f%e3%81%ab%e3%82%aa%e3%83%96%e3%82%b9%e3%82%af%e3%83%a9-raw-free/",
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
		url:      "https://klmanga.lt/manga-raw/%e3%82%a2%e3%83%b3%e3%83%81%e3%83%ad%e3%83%9e%e3%83%b3%e3%82%b9-raw-free/",
		quantity: 18,
	},
	{
		url:      "https://klmanga.lt/manga-raw/%e3%82%a2%e3%82%a4%e3%83%9e%e3%82%a4%e3%83%9f%e3%83%bc%e3%83%9e%e3%82%a4%e3%83%b3-raw-free/",
		quantity: 8,
	},
	{
		url:      "https://klmanga.lt/manga-raw/%e6%80%9d%e3%81%88%e3%81%b0%e9%81%a0%e3%81%8f%e3%81%ab%e3%82%aa%e3%83%96%e3%82%b9%e3%82%af%e3%83%a9-raw-free/",
		quantity: 21,
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
