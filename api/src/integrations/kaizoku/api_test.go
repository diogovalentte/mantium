package kaizoku

import (
	"fmt"
	"os"
	"testing"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
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

var mangasTest = []*manga.Manga{
	{
		Name:   "Tower Dungeon",
		Source: "mangadex.org",
	},
}

func TestRequest(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test Request", func(t *testing.T) {
		_, err := k.Request("GET", k.Address, nil)
		if err != nil {
			t.Errorf("Error while making request: %v", err)
			return
		}
	})
}

func TestGetSources(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test get manga sources", func(t *testing.T) {
		sources, err := k.GetSources()
		if err != nil {
			t.Errorf("Error while getting manga sources: %v", err)
			return
		}

		if len(sources) == 0 {
			t.Errorf("No sources found")
			return
		}
	})
}

func TestAddManga(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test add manga", func(t *testing.T) {
		for _, testManga := range mangasTest {
			err := k.AddManga(testManga)
			if err != nil {
				t.Errorf("Error while adding manga: %v", err)
				return
			}
		}
	})
	t.Run("Test add manga with invalid source", func(t *testing.T) {
		for _, testManga := range mangasTest {
			testManga.Source = "invalid"
			err := k.AddManga(testManga)
			if err != nil {
				if !util.ErrorContains(err, "Unknown source") {
					t.Errorf("Unknown rrror while adding manga: %v", err)
					return
				}
			} else {
				t.Errorf("Error is nil when it shouldn't")
				return
			}
		}
	})
	t.Run("Test add manga with invalid name", func(t *testing.T) {
		for _, testManga := range mangasTest {
			testManga.Name = "invalid12345"
			err := k.AddManga(testManga)
			if err != nil {
				if !util.ErrorContains(err, fmt.Sprintf("Cannot find the %s.", testManga.Name)) {
					t.Errorf("Unknown rrror while adding manga: %v", err)
					return
				}
			} else {
				t.Errorf("Error is nil when it shouldn't")
				return
			}
		}
	})
}

func TestGetMangas(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test get mangas", func(t *testing.T) {
		mangas, err := k.GetMangas()
		if err != nil {
			t.Errorf("Error while getting mangas: %v", err)
			return
		}

		if len(mangas) == 0 {
			t.Errorf("No mangas found")
			return
		}

		for _, manga := range mangas {
			if manga.ID == 0 {
				t.Errorf("Manga ID not found")
				return
			}
			if manga.Title == "" {
				t.Errorf("Manga title not found")
				return
			}
			if manga.Source == "" {
				t.Errorf("Manga source not found")
				return
			}
		}
	})
}

func TestGetManga(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test get manga", func(t *testing.T) {
		for _, testManga := range mangasTest {
			manga, err := k.GetManga(testManga.Name)
			if err != nil {
				t.Errorf("Error while getting manga: %v", err)
				return
			}

			if manga.ID == 0 {
				t.Errorf("Manga ID not found")
				return
			}
			if manga.Title == "" {
				t.Errorf("Manga title not found")
				return
			}
			if manga.Source == "" {
				t.Errorf("Manga source not found")
				return
			}
		}
	})
	t.Run("Test get manga with invalid name", func(t *testing.T) {
		for _, testManga := range mangasTest {
			testManga.Name = "invalid"
			_, err := k.GetManga(testManga.Name)
			if err != nil {
				if !util.ErrorContains(err, "Manga not found in Kaizoku") {
					t.Errorf("Unknown error while adding manga: %v", err)
					return
				}
			} else {
				t.Errorf("Error is nil when it shouldn't")
				return
			}
		}
	})
}

func TestCheckOutOfSyncChapters(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test check out of sync chapters", func(t *testing.T) {
		err := k.CheckOutOfSyncChapters()
		if err != nil {
			t.Errorf("Error while checking out of sync chapters: %v", err)
			return
		}
	})
}

func TestFixOutOfSyncChapters(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test fix out of sync chapters", func(t *testing.T) {
		err := k.FixOutOfSyncChapters()
		if err != nil {
			// If called right after CheckOutOfSyncChapters, it will return an error
			// because the check out of sync chapters job is still running.
			if !util.ErrorContains(err, "There is another active job running.") {
				t.Errorf("Error while fixing out of sync chapters: %v", err)
				return
			}
		}
	})
}

func TestRemoveManga(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	// Sometimes there is a delay for Kaizoku to add the manga,
	// so this can return an error
	t.Run("Test remove manga", func(t *testing.T) {
		for _, testManga := range mangasTest {
			manga, err := k.GetManga(testManga.Name)
			if err != nil {
				t.Errorf("Error while getting manga: %v", err)
				return
			}

			err = k.RemoveManga(manga.ID, true)
			if err != nil {
				t.Errorf("Error while removing manga: %v", err)
				return
			}
		}
	})
}
