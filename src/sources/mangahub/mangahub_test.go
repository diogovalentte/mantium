package mangahub

import (
	"reflect"
	"testing"
	"time"

	"github.com/diogovalentte/manga-dashboard-api/src/manga"
)

type mangaTest struct {
	expected *manga.Manga
	url      string
}

var mangasTestTable = []mangaTest{
	{
		expected: &manga.Manga{
			Name:        "Death Note",
			Source:      "mangahub.io",
			URL:         "https://mangahub.io/manga/death-note_119",
			CoverImgURL: "https://thumb.mangahub.io/mn/death-note.jpg",
			LastUploadChapter: &manga.Chapter{
				Number:    112,
				Name:      "Chapter 112",
				URL:       "https://mangahub.io/chapter/death-note_119/chapter-112",
				UpdatedAt: time.Date(2018, 6, 16, 0, 0, 0, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://mangahub.io/manga/death-note_119",
	},
	{
		expected: &manga.Manga{
			Name:        "Vagabond",
			Source:      "mangahub.io",
			URL:         "https://mangahub.io/manga/vagabond_119",
			CoverImgURL: "https://thumb.mangahub.io/mn/vagabond.jpg",
			LastUploadChapter: &manga.Chapter{
				Number:    327,
				Name:      "The Man Named Tadaoki",
				URL:       "https://mangahub.io/chapter/vagabond_119/chapter-327",
				UpdatedAt: time.Date(2016, 1, 20, 0, 0, 0, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
				Type:      1,
			},
		},
		url: "https://mangahub.io/manga/vagabond_119",
	},
	{
		expected: &manga.Manga{
			Name:        "Mob Psycho 100",
			Source:      "mangahub.io",
			URL:         "https://mangahub.io/manga/mob-psycho-100",
			CoverImgURL: "https://thumb.mangahub.io/mn/mob-psycho-100.jpg",
			LastUploadChapter: &manga.Chapter{
				Number:    101,
				Name:      "101[END]",
				URL:       "https://mangahub.io/chapter/mob-psycho-100/chapter-101",
				UpdatedAt: time.Date(2018, 4, 26, 0, 0, 0, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://mangahub.io/manga/mob-psycho-100",
	},
}

func TestGetMangaMetadata(t *testing.T) {
	t.Run("should scrape data from multiple mangas", func(t *testing.T) {
		source := Source{}
		for _, test := range mangasTestTable {
			expected := test.expected
			mangaURL := test.url

			actualManga, err := source.GetMangaMetadata(mangaURL)
			if err != nil {
				t.Errorf("error while getting manga: %v", err)
				return
			}

			// Cover img
			if actualManga.CoverImg == nil {
				t.Errorf("expected manga.CoverImg to be different than nil")
				return
			}
			actualManga.CoverImg = nil

			// Compare manga
			if !reflect.DeepEqual(actualManga, expected) {
				t.Errorf("expected manga %v, got %v", expected, actualManga)
				t.Errorf("expected manga.LastChapter %v, got %v", expected.LastUploadChapter, actualManga.LastUploadChapter)
				return
			}
		}
	})
}

type chapterTest struct {
	url      string
	quantity int
}

var getChaptersTestTable = []chapterTest{
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
	t.Run("should scrape chapters' info from multiple mangas", func(t *testing.T) {
		source := Source{}
		for _, test := range getChaptersTestTable {
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
				if chapter.Number == 0 {
					t.Errorf("expected chapter.Chapter to be different than 0")
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

type getLastChapterTest struct {
	expected *manga.Chapter
	url      string
}

var chaptersTestTable = []getLastChapterTest{
	{
		expected: &manga.Chapter{
			Number:    155.2,
			Name:      "Interview with Yagi Norihiro Extended",
			URL:       "https://mangahub.io/chapter/claymore_116/chapter-155.2",
			UpdatedAt: time.Date(2016, 1, 20, 0, 0, 0, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		url: "https://mangahub.io/manga/claymore_116",
	},
	{
		expected: &manga.Chapter{
			Number:    249,
			Name:      "[End]",
			URL:       "https://mangahub.io/chapter/20th-century-boys_116/chapter-249",
			UpdatedAt: time.Date(2016, 1, 20, 0, 0, 0, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
		},
		url: "https://mangahub.io/manga/20th-century-boys_116",
	},
	{
		expected: &manga.Chapter{
			Number:    21,
			Name:      "The Horizon 21",
			URL:       "https://mangahub.io/chapter/the-horizon/chapter-21",
			UpdatedAt: time.Date(2020, 5, 10, 0, 0, 0, 0, time.UTC),
		},
		url: "https://mangahub.io/manga/the-horizon",
	},
}

func TestGetChapterMetadata(t *testing.T) {
	t.Run("should scrape chapter's info from multiple mangas", func(t *testing.T) {
		source := Source{}
		for _, test := range chaptersTestTable {
			expected := test.expected
			mangaURL := test.url

			actualChapter, err := source.GetChapterMetadata(mangaURL, expected.Number)
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
	t.Run("should scrape last chapter's info from multiple mangas", func(t *testing.T) {
		source := Source{}
		for _, test := range chaptersTestTable {
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

type getMangaUploadedTimeTest struct {
	arg      string
	expected time.Time
	sub      time.Duration
}

var getMangaUploadedTimeTestsAbsTime = []getMangaUploadedTimeTest{
	{
		arg:      "01-18-2023",
		expected: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC),
	},
	{
		arg:      "12-01-1999",
		expected: time.Date(1999, 12, 1, 0, 0, 0, 0, time.UTC),
	},
}

var getMangaUploadedTimeTestsRelativeTime = []getMangaUploadedTimeTest{
	{
		arg: "3 hours ago",
		sub: 3 * time.Hour,
	},
	{
		arg: "5 days ago",
		sub: (5 * time.Hour) * 24,
	},
	{
		arg: "2 weeks ago",
		sub: (2 * time.Hour) * 24 * 7,
	},
}

func TestGetMangaUploadedTime(t *testing.T) {
	t.Run("should return a time.Time from absolute time args", func(t *testing.T) {
		for _, test := range getMangaUploadedTimeTestsAbsTime {
			actual, err := getMangaUploadedTime(test.arg)
			if err != nil {
				t.Errorf("error while getting manga uploaded time: %v", err)
				return
			}
			if actual != test.expected {
				t.Errorf("expected %v, got %v", test.expected, actual)
				return
			}
		}
	})
	t.Run("should return a time.Time from relative time args", func(t *testing.T) {
		for _, test := range getMangaUploadedTimeTestsRelativeTime {
			actual, err := getMangaUploadedTime(test.arg)
			if err != nil {
				t.Errorf("error while getting manga uploaded time: %v", err)
			}

			expectedDate := time.Now().Add(test.sub * -1)
			expected := time.Date(expectedDate.Year(), expectedDate.Month(), expectedDate.Day(), 0, 0, 0, 0, time.UTC)
			if actual != expected {
				t.Errorf("expected %v, got %v", expected, actual)
				return
			}
		}
	})
}
