package tranga

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

var mangasTest = []struct {
	Name      string
	Source    string
	URL       string
	Connector string
}{
	{
		Name:      "Tower Dungeon",
		URL:       "https://mangadex.org/title/b05918e4-fb1a-4b10-a919-eaecf00fd7dd",
		Source:    "mangadex.org",
		Connector: "MangaDex",
	},
	{
		Name:      "One Punch Man",
		URL:       "https://mangadex.org/title/d8a959f7-648e-4c8d-8f23-f1f3f8e129f3",
		Source:    "mangadex.org",
		Connector: "MangaDex",
	},
}

func TestGetMonitorJobs(t *testing.T) {
	tranga := Tranga{}
	tranga.Init()

	t.Run("Get Monitor Jobs", func(t *testing.T) {
		_, err := tranga.GetMonitorJobs()
		if err != nil {
			t.Fatalf("error while getting monitor jobs: %v", err)
		}
	})
}

func TestSearchManga(t *testing.T) {
	tranga := Tranga{}
	tranga.Init()

	t.Run("Search Mangas", func(t *testing.T) {
		for _, m := range mangasTest {
			manga := &manga.Manga{
				Name:   m.Name,
				URL:    m.URL,
				Source: m.Source,
			}

			_, err := tranga.SearchManga(manga, m.Connector)
			if err != nil {
				t.Fatalf("error while getting manga: %v", err)
			}
		}
	})
}

func TestAddManga(t *testing.T) {
	tranga := Tranga{}
	tranga.Init()

	t.Run("Add Mangas", func(t *testing.T) {
		for _, m := range mangasTest {
			manga := &manga.Manga{
				Name:   m.Name,
				URL:    m.URL,
				Source: m.Source,
			}

			err := tranga.AddManga(manga)
			if err != nil {
				t.Fatalf("error while adding manga: %v", err)
			}
		}
	})
}

func TestStartJob(t *testing.T) {
	tranga := Tranga{}
	tranga.Init()

	t.Run("Start Jobs", func(t *testing.T) {
		for _, m := range mangasTest {
			manga := &manga.Manga{
				Name:   m.Name,
				URL:    m.URL,
				Source: m.Source,
			}

			err := tranga.StartJob(manga)
			if err != nil {
				t.Fatalf("error while starting job: %v", err)
			}
		}
	})
}

func TestGetConnectors(t *testing.T) {
	tranga := Tranga{}
	tranga.Init()

	t.Run("Get Connectors", func(t *testing.T) {
		connectors, err := tranga.GetConnectors()
		if err != nil {
			t.Fatalf("error while getting connectors: %v", err)
		}

		if len(connectors) == 0 {
			t.Fatalf("no connectors found")
		}
	})
}
