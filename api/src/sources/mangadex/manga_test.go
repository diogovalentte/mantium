package mangadex

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

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
			Name:            "Death Note",
			Source:          "mangadex.org",
			URL:             "https://mangadex.org/title/75ee72ab-c6bf-4b87-badd-de839156934c/death-note",
			CoverImgURL:     "https://uploads.mangadex.org/covers/75ee72ab-c6bf-4b87-badd-de839156934c/d6555598-8202-477d-acde-303202cb3475.jpg",
			CoverImgResized: true,
			LastUploadChapter: &manga.Chapter{
				Chapter:   "108",
				Name:      "End",
				URL:       "https://mangadex.org/chapter/5fff451c-cbe1-4456-9ef5-4e3c3e41dc26",
				UpdatedAt: time.Date(2018, 4, 7, 7, 35, 8, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://mangadex.org/title/75ee72ab-c6bf-4b87-badd-de839156934c/death-note",
	},
	{
		expected: &manga.Manga{
			Name:            "Vagabond",
			Source:          "mangadex.org",
			URL:             "https://mangadex.org/title/d1a9fdeb-f713-407f-960c-8326b586e6fd/vagabond",
			CoverImgURL:     "https://uploads.mangadex.org/covers/d1a9fdeb-f713-407f-960c-8326b586e6fd/05f8dcb4-8ea1-48db-a0b1-3a8fbf695e5a.jpg",
			CoverImgResized: true,
			LastUploadChapter: &manga.Chapter{
				Chapter:   "327",
				Name:      "The Man Named Tadoki",
				URL:       "https://mangadex.org/chapter/0754c218-0240-4752-a688-5e7d9bc74b55",
				UpdatedAt: time.Date(2018, 3, 19, 2, 20, 43, 0, time.UTC), // in the site it's 01-19-2016 (maybe it uses JS or it have to wait a bit to update)
				Type:      1,
			},
		},
		url: "https://mangadex.org/title/d1a9fdeb-f713-407f-960c-8326b586e6fd/vagabond",
	},
	{
		expected: &manga.Manga{
			Name:            "Mob Psycho 100",
			Source:          "mangadex.org",
			URL:             "https://mangadex.org/title/736a2bf0-f875-4b52-a7b4-e8c40505b68a/mob-psycho-100",
			CoverImgURL:     "https://uploads.mangadex.org/covers/736a2bf0-f875-4b52-a7b4-e8c40505b68a/7f07f02e-39ba-4e38-a01d-6f74652013fa.jpg",
			CoverImgResized: true,
			LastUploadChapter: &manga.Chapter{
				Chapter:   "101",
				Name:      "101",
				URL:       "https://mangadex.org/chapter/c8ba4080-2cb0-466e-9a17-02fe12782f70",
				UpdatedAt: time.Date(2018, 2, 12, 1, 49, 12, 0, time.UTC),
				Type:      1,
			},
		},
		url: "https://mangadex.org/title/736a2bf0-f875-4b52-a7b4-e8c40505b68a/mob-psycho-100",
	},
}

func TestGetMangaMetadata(t *testing.T) {
	source := Source{}

	t.Run("should get the  metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			expected := test.expected
			expected.LastUploadChapter.UpdatedAt = expected.LastUploadChapter.UpdatedAt.In(time.Local)
			mangaURL := test.url

			actualManga, err := source.GetMangaMetadata(mangaURL)
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
	t.Run("should not get the metadata from multiple mangas", func(t *testing.T) {
		for _, test := range mangasTestTable {
			mangaURL := test.url
			mangaURL, err := replaceURLID(test.url, "00000000-0000-0000-0000-000000000000")
			if err != nil {
				t.Fatalf("Error while replacing manga URL ID: %v", err)
			}

			_, err = source.GetMangaMetadata(mangaURL)
			if err != nil {
				if util.ErrorContains(err, "Non-200 status code -> (404)") {
					continue
				}
				t.Fatalf("expected error, got %s", err)
			}
			t.Fatalf("expected error, got nil")
		}
	})
}

var getMangaIDTestTable = []string{
	"https://mangadex.org/title/87ebd557-8394-4f16-8afe-a8644e555ddc/hirayasumi/",
	"https://mangadex.org/title/87ebd557-8394-4f16-8afe-a8644e555ddc/hirayasumi",
	"https://mangadex.org/title/87ebd557-8394-4f16-8afe-a8644e555ddc",
}

func TestGetMangaID(t *testing.T) {
	t.Run("should return the ID of a manga URL", func(t *testing.T) {
		for _, mangaURL := range getMangaIDTestTable {
			expected := "87ebd557-8394-4f16-8afe-a8644e555ddc"
			result, err := getMangaID(mangaURL)
			if err != nil {
				t.Fatalf("Error: %s", err)
			}
			if result != expected {
				t.Fatalf("Expected %s, got %s", expected, result)
			}
		}
	})
}

// replaceMangaURLID replaces the ID of a manga/chapter URL with a replacement ID.
func replaceURLID(urlString string, replacement string) (string, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}

	pathParts := strings.Split(u.Path, "/")
	titleIndex := -1
	for i, part := range pathParts {
		if part == "title" || part == "chapter" {
			titleIndex = i
			break
		}
	}

	if titleIndex != -1 && titleIndex+1 < len(pathParts) {
		pathParts[titleIndex+1] = replacement
	}

	u.Path = strings.Join(pathParts, "/")

	return u.String(), nil
}
