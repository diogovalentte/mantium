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
	Name:   "One Punch Man",
	Source: "comick",
	URL:    "https://comick.io/comic/WfaSlMP9",
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
		sourceID, err := s.fetchSourceID("Comick (ALL)")
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
		sourceID, err := s.fetchSourceID("MangaDex (EN)")
		if err != nil {
			t.Fatalf("error while fetching source ID: %v", err)
		}

		_, err = s.fetchSourceManga(sourceID, "Neighborhood Craftsmen: Stories from Kanda’s Gokura-chou", "https://mangadex.org/title/c1b61cc0-7bb1-4280-a35a-c30a9ee0ff56/kanda-gokura-chou-shokunin-banashi", 1)
		if err != nil {
			t.Fatalf("error while fetching manga: %v", err)
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
