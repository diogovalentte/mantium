package suwayomi

import (
	"fmt"
	"os"
	"testing"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/manga"
)

func setup() error {
	err := config.SetConfigs("../../../../.env.test")
	if err != nil {
		return err
	}

	return nil
}

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}

var inputManga = &manga.Manga{
	// Name:   "One Punch Man",
	// Source: "comick",
	// URL:    "https://comick.io/comic/WfaSlMP9",
	Name:   "Neighborhood Craftsmen: Stories from Kanda’s Gokura-chou",
	Source: "mangadex",
	URL:    "https://mangadex.org/title/c1b61cc0-7bb1-4280-a35a-c30a9ee0ff56",
}

func TestFetchSources(t *testing.T) {
	s := Suwayomi{}
	s.Init()

	t.Run("Test fetch sources", func(t *testing.T) {
		sources, err := s.fetchSources()
		if err != nil {
			t.Fatalf("error while fetching sources: %v", err)
		}

		if len(sources) == 0 {
			t.Fatal("no sources found")
		}
	})
}

func TestFetchSourceID(t *testing.T) {
	s := Suwayomi{}
	s.Init()

	t.Run("Test fetch source ID", func(t *testing.T) {
		source, err := s.translateSuwayomiSource(inputManga.Source)
		if err != nil {
			t.Fatalf("error while translating source: %v", err)
		}
		sourceID, err := s.fetchSourceID(source)
		if err != nil {
			t.Fatalf("error while fetching source ID: %v", err)
		}

		if sourceID == "" {
			t.Fatal("no source ID found")
		}
	})
}

func TestFetchManga(t *testing.T) {
	s := Suwayomi{}
	s.Init()

	t.Run("Test fetch manga", func(t *testing.T) {
		source, err := s.translateSuwayomiSource(inputManga.Source)
		if err != nil {
			t.Fatalf("error while translating source: %v", err)
		}
		sourceID, err := s.fetchSourceID(source)
		if err != nil {
			t.Fatalf("error while fetching source ID: %v", err)
		}
		manga, err := s.fetchSourceManga(sourceID, inputManga, 1)
		if err != nil {
			t.Fatalf("error while fetching manga: %v", err)
		}

		if manga.ID == 0 {
			t.Fatal("no manga ID found")
		}
		if manga.URL == "" {
			t.Fatal("no manga URL found")
		}
	})
}

func TestAddManga(t *testing.T) {
	s := Suwayomi{}
	s.Init()

	t.Run("Test add manga", func(t *testing.T) {
		err := s.AddManga(inputManga)
		if err != nil {
			t.Fatalf("error while adding manga: %v", err)
		}
	})
}

func TestGetLibraryMangaID(t *testing.T) {
	s := Suwayomi{}
	s.Init()

	t.Run("Test get library manga ID", func(t *testing.T) {
		mangaID, err := s.GetLibraryMangaID(inputManga)
		if err != nil {
			t.Fatalf("error while getting library manga ID: %v", err)
		}

		if mangaID == 0 {
			t.Fatal("no manga ID found")
		}
	})
}

func TestGetMangaChapters(t *testing.T) {
	s := Suwayomi{}
	s.Init()

	t.Run("Test get manga chapters", func(t *testing.T) {
		mangaID, err := s.GetLibraryMangaID(inputManga)
		if err != nil {
			t.Fatalf("error while getting library manga ID: %v", err)
		}
		chapters, err := s.GetChapters(mangaID)
		if err != nil {
			t.Fatalf("error while getting manga chapters: %v", err)
		}

		if len(chapters) == 0 {
			t.Fatal("no chapters found")
		}
	})
}

func TestGetMangaChapter(t *testing.T) {
	s := Suwayomi{}
	s.Init()

	t.Run("Test get manga chapter", func(t *testing.T) {
		mangaID, err := s.GetLibraryMangaID(inputManga)
		if err != nil {
			t.Fatalf("error while getting library manga ID: %v", err)
		}
		chapters, err := s.GetChapters(mangaID)
		if err != nil {
			t.Fatalf("error while getting manga chapters: %v", err)
		}

		if len(chapters) == 0 {
			t.Fatal("no chapters found")
		}

		chapter, err := s.GetChapter(mangaID, chapters[0].RealURL)
		if err != nil {
			t.Fatalf("error while getting manga chapter: %v", err)
		}

		if chapter.ID == 0 {
			t.Fatal("no chapter ID found")
		}
	})
}

func TestEnqueueChapterDownload(t *testing.T) {
	s := Suwayomi{}
	s.Init()

	t.Run("Test enqueue chapter download", func(t *testing.T) {
		mangaID, err := s.GetLibraryMangaID(inputManga)
		if err != nil {
			t.Fatalf("error while getting library manga ID: %v", err)
		}
		chapters, err := s.GetChapters(mangaID)
		if err != nil {
			t.Fatalf("error while getting manga chapters: %v", err)
		}

		if len(chapters) == 0 {
			t.Fatal("no chapters found")
		}

		err = s.EnqueueChapterDownload(chapters[0].ID)
		if err != nil {
			t.Fatalf("error while enqueuing chapter download: %v", err)
		}
	})
}
