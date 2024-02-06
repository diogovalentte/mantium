package mangadex

import (
	"testing"
)

func TestGetChapterMetadata(t *testing.T) {
	t.Run("should return the metadata of a chapter given its URL", func(t *testing.T) {
		source := &Source{}
		chapterURL := "https://mangadex.org/chapter/e393167b-573c-414f-8514-f7ff1fc6604d"
		_, err := source.GetChapterMetadata("", 0, chapterURL)
		if err != nil {
			t.Errorf("Error: %s", err)
			return
		}
	})
}

func TestGetLastChapterMetadata(t *testing.T) {
	t.Run("should return the metadata of the last chapter of a manga", func(t *testing.T) {
		source := &Source{}
		mangaURL := "https://mangadex.org/title/75ee72ab-c6bf-4b87-badd-de839156934c/death-note"
		_, err := source.GetLastChapterMetadata(mangaURL)
		if err != nil {
			t.Errorf("Error: %s", err)
			return
		}
	})
}

func TestGetChaptersMetadata(t *testing.T) {
	t.Run("should return the manga's chapters metadata", func(t *testing.T) {
		source := &Source{}
		mangaURL := "https://mangadex.org/title/239d6260-d71f-43b0-afff-074e3619e3de/bleach"
		chapters, err := source.GetChaptersMetadata(mangaURL)
		if err != nil {
			t.Errorf("Error: %s", err)
			return
		}

		expected := 704
		if len(chapters) != expected {
			t.Errorf("Expected %d chapters, got %d", expected, len(chapters))
			return
		}
	})
}

func TestGetChapterID(t *testing.T) {
	t.Run("should return the ID of a chapter URL", func(t *testing.T) {
		chapterURL := "https://mangadex.org/chapter/e393167b-573c-414f-8514-f7ff1fc6604d"
		expected := "e393167b-573c-414f-8514-f7ff1fc6604d"
		result, err := getChapterID(chapterURL)
		if err != nil {
			t.Errorf("Error: %s", err)
			return
		}
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
			return
		}
	})
}
