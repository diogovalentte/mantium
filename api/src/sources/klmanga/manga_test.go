package klmanga

import (
	"reflect"
	"testing"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

type mangaTestType struct {
	expected *manga.Manga
	url      string
}

var mangasTestTable = []mangaTestType{
	{
		expected: &manga.Manga{
			Name:            "アンチロマンス",
			Source:          "klmanga",
			URL:             "https://klmanga.talk/manga-raw/%e3%82%a2%e3%83%b3%e3%83%81%e3%83%ad%e3%83%9e%e3%83%b3%e3%82%b9-raw-free/",
			CoverImgURL:     "https://klmanga.talk/wp-content/uploads/2024/04/001-100.webp",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter: "18",
				Name:    "アンチロマンス (Raw – Free) 【第18話】",
				URL:     "https://klmanga.talk/manga-raw/%e3%82%a2%e3%83%b3%e3%83%81%e3%83%ad%e3%83%9e%e3%83%b3%e3%82%b9-raw-free/chapter-18/",
				Type:    1,
			},
		},
		url: "https://klmanga.talk/manga-raw/%e3%82%a2%e3%83%b3%e3%83%81%e3%83%ad%e3%83%9e%e3%83%b3%e3%82%b9-raw-free/",
	},
	{
		expected: &manga.Manga{
			Name:            "アイマイミーマイン",
			Source:          "klmanga",
			URL:             "https://klmanga.talk/manga-raw/%e3%82%a2%e3%82%a4%e3%83%9e%e3%82%a4%e3%83%9f%e3%83%bc%e3%83%9e%e3%82%a4%e3%83%b3-raw-free/",
			CoverImgURL:     "https://klmanga.talk/wp-content/uploads/2024/03/1245784.jpg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter: "8",
				Name:    "アイマイミーマイン (Raw – Free) 【第8話】",
				URL:     "https://klmanga.talk/manga-raw/%e3%82%a2%e3%82%a4%e3%83%9e%e3%82%a4%e3%83%9f%e3%83%bc%e3%83%9e%e3%82%a4%e3%83%b3-raw-free/chapter-8/",
				Type:    1,
			},
		},
		url: "https://klmanga.talk/manga-raw/%e3%82%a2%e3%82%a4%e3%83%9e%e3%82%a4%e3%83%9f%e3%83%bc%e3%83%9e%e3%82%a4%e3%83%b3-raw-free/",
	},
	{
		expected: &manga.Manga{
			Name:            "思えば遠くにオブスクラ",
			Source:          "klmanga",
			URL:             "https://klmanga.talk/manga-raw/%e6%80%9d%e3%81%88%e3%81%b0%e9%81%a0%e3%81%8f%e3%81%ab%e3%82%aa%e3%83%96%e3%82%b9%e3%82%af%e3%83%a9-raw-free/",
			CoverImgURL:     "https://klmanga.talk/wp-content/uploads/2024/04/DL-Raw.Se_0001-10.jpg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter: "20",
				Name:    "思えば遠くにオブスクラ (Raw – Free) 【第20話】",
				URL:     "https://klmanga.talk/manga-raw/%e6%80%9d%e3%81%88%e3%81%b0%e9%81%a0%e3%81%8f%e3%81%ab%e3%82%aa%e3%83%96%e3%82%b9%e3%82%af%e3%83%a9-raw-free/chapter-20/",
				Type:    1,
			},
		},
		url: "https://klmanga.talk/manga-raw/%e6%80%9d%e3%81%88%e3%81%b0%e9%81%a0%e3%81%8f%e3%81%ab%e3%82%aa%e3%83%96%e3%82%b9%e3%82%af%e3%83%a9-raw-free/",
	},
}

func TestGetMangaMetadata(t *testing.T) {
	source := Source{}

	t.Run("Should scrape metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			expected := test.expected
			mangaURL := test.url

			actualManga, err := source.GetMangaMetadata(mangaURL, "")
			if err != nil {
				t.Fatalf("error while getting manga: %v", err)
			}

			if actualManga.CoverImg == nil {
				t.Fatalf("expected manga.CoverImg to be different than nil")
			}
			actualManga.CoverImg = nil

			if !reflect.DeepEqual(actualManga, expected) {
				t.Fatalf("expected manga %s, got %s", expected, actualManga)
			}
		}
	})
	t.Run("Should not scrape metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			mangaURL := test.url + "salt/salt2"

			_, err := source.GetMangaMetadata(mangaURL, "")
			if err != nil {
				if util.ErrorContains(err, errordefs.ErrMangaNotFound.Error()) {
					continue
				}
				t.Fatalf("expected error, got %s", err)
			}
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestSearch(t *testing.T) {
	source := Source{}

	t.Run("Should search for multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			mangaName := test.expected.Name

			results, err := source.Search(mangaName, 20)
			if err != nil {
				t.Fatalf("error while searching: %v", err)
			}

			if len(results) == 0 {
				t.Fatalf("expected results to be different than 0")
			}
		}
	})
}
