package jmanga

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
			Source:          "jmanga",
			URL:             "https://jmanga.ltd/read/%E3%82%A2%E3%83%B3%E3%83%81%E3%83%AD%E3%83%9E%E3%83%B3%E3%82%B9-raw/",
			CoverImgURL:     "https://imgjm.jmanga.ac/thumb/300/upload/2024/11/3706860f185338ce5bc03f12bb9cfedb.jpeg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter: "18",
				Name:    "第18話",
				URL:     "https://jmanga.ltd/read/アンチロマンス/ja/chapter-18-raw/",
				Type:    1,
			},
		},
		url: "https://jmanga.ltd/read/%E3%82%A2%E3%83%B3%E3%83%81%E3%83%AD%E3%83%9E%E3%83%B3%E3%82%B9-raw/",
	},
	{
		expected: &manga.Manga{
			Name:            "アイマイミーマイン",
			Source:          "jmanga",
			URL:             "https://jmanga.ltd/read/%E3%82%A2%E3%82%A4%E3%83%9E%E3%82%A4%E3%83%9F%E3%83%BC%E3%83%9E%E3%82%A4%E3%83%B3-raw/",
			CoverImgURL:     "https://imgjm.jmanga.ac/thumb/300/upload/2024/11/80b191b8a643b1af2ac73e456ac33dce.jpeg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter: "8",
				Name:    "第8話",
				URL:     "https://jmanga.ltd/read/アイマイミーマイン/ja/chapter-8-raw/",
				Type:    1,
			},
		},
		url: "https://jmanga.ltd/read/%E3%82%A2%E3%82%A4%E3%83%9E%E3%82%A4%E3%83%9F%E3%83%BC%E3%83%9E%E3%82%A4%E3%83%B3-raw/",
	},
	{
		expected: &manga.Manga{
			Name:            "思えば遠くにオブスクラ",
			Source:          "jmanga",
			URL:             "https://jmanga.ltd/read/%E6%80%9D%E3%81%88%E3%81%B0%E9%81%A0%E3%81%8F%E3%81%AB%E3%82%AA%E3%83%96%E3%82%B9%E3%82%AF%E3%83%A9-raw/",
			CoverImgURL:     "https://imgjm.jmanga.ac/thumb/300/upload/2024/11/ecd8f873c209664c9405bac06c06153c.jpeg",
			CoverImgResized: true,
			LastReleasedChapter: &manga.Chapter{
				Chapter: "20",
				Name:    "第20話",
				URL:     "https://jmanga.ltd/read/思えば遠くにオブスクラ/ja/chapter-20-raw/",
				Type:    1,
			},
		},
		url: "https://jmanga.ltd/read/%E6%80%9D%E3%81%88%E3%81%B0%E9%81%A0%E3%81%8F%E3%81%AB%E3%82%AA%E3%83%96%E3%82%B9%E3%82%AF%E3%83%A9-raw/",
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
