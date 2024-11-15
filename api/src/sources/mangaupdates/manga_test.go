package mangaupdates

import (
	"reflect"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/manga"
)

type mangasTestT struct {
	expected        *manga.Manga
	url             string
	mangaInternalID string
}

var mangasTestTable = []mangasTestT{
	{
		expected: &manga.Manga{
			Name:            "Death Note",
			Source:          "mangaupdates",
			URL:             "https://www.mangaupdates.com/series/1ljv3bs/death-note",
			InternalID:      "3479935384",
			CoverImgURL:     "https://cdn.mangaupdates.com/image/i295749.jpg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter:    "108",
				Name:       "Death Note",
				URL:        "https://www.mangaupdates.com/releases.html?stype=series&search=3479935384",
				InternalID: "681314",
				UpdatedAt:  time.Date(2021, 1, 7, 0, 0, 0, 0, time.UTC),
				Type:       1,
			},
		},
		url:             "https://www.mangaupdates.com/series/1ljv3bs/death-note",
		mangaInternalID: "3479935384",
	},
	{
		expected: &manga.Manga{
			Name:            "Vagabond",
			Source:          "mangaupdates",
			URL:             "https://www.mangaupdates.com/series/su6blie/vagabond",
			InternalID:      "62774509478",
			CoverImgURL:     "https://cdn.mangaupdates.com/image/i426098.png",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter:    "2",
				Name:       "Vagabond",
				URL:        "https://www.mangaupdates.com/releases.html?stype=series&search=62774509478",
				InternalID: "665004",
				UpdatedAt:  time.Date(2020, 11, 8, 0, 0, 0, 0, time.UTC),
				Type:       1,
			},
		},
		url:             "https://www.mangaupdates.com/series/su6blie/vagabond",
		mangaInternalID: "62774509478",
	},
}

func TestGetMangaMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should get the metadata of multiple mangas using their manga internal ID", func(t *testing.T) {
		for _, test := range mangasTestTable {
			expected := test.expected

			manga, err := source.GetMangaMetadata("", test.mangaInternalID)
			if err != nil {
				t.Fatalf("error while searching: %v", err)
			}

			if manga.CoverImg == nil {
				t.Fatalf("expected manga.CoverImg to be different than nil")
			}
			manga.CoverImg = nil

			if !reflect.DeepEqual(manga, expected) {
				t.Fatalf("expected manga %s, got %s", expected, manga)
			}
		}
	})
	t.Run("Should get the metadata of multiple mangas using their manga URL", func(t *testing.T) {
		for _, test := range mangasTestTable {
			expected := test.expected

			manga, err := source.GetMangaMetadata(test.url, "")
			if err != nil {
				t.Fatalf("error while searching: %v", err)
			}

			if manga.CoverImg == nil {
				t.Fatalf("expected manga.CoverImg to be different than nil")
			}
			manga.CoverImg = nil

			if !reflect.DeepEqual(manga, expected) {
				t.Fatalf("expected manga %s, got %s", expected, manga)
			}
		}
	})
}

func TestSearch(t *testing.T) {
	source := Source{}

	t.Run("Should search for multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			mangaName := test.expected.Name

			results, err := source.Search(mangaName, 15)
			if err != nil {
				t.Fatalf("error while searching: %v", err)
			}

			if len(results) == 0 {
				t.Fatalf("expected results to be different than 0")
			}
		}
	})
}

func TestGetMangaIDFromURL(t *testing.T) {
	source := Source{}

	t.Run("Should get the ID of multiple mangas using their URLs", func(t *testing.T) {
		for _, test := range mangasTestTable {
			excepted := test.mangaInternalID
			mangaURL := test.url
			results, err := source.getMangaIDFromURL(mangaURL)
			if err != nil {
				t.Fatalf("error while getting manga ID from URL: %v", err)
			}

			if results != excepted {
				t.Fatalf("expected %s, got %s", excepted, results)
			}
		}
	})
}
