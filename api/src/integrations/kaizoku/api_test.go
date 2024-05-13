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
			t.Fatalf("Error while making request: %v", err)
		}
	})
}

func TestGetSources(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test get manga sources", func(t *testing.T) {
		sources, err := k.GetSources()
		if err != nil {
			t.Fatalf("Error while getting manga sources: %v", err)
		}

		if len(sources) == 0 {
			t.Fatalf("No sources found")
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
				t.Fatalf("Error while adding manga: %v", err)
			}
		}
	})
	t.Run("Test add manga with invalid source", func(t *testing.T) {
		for _, testManga := range mangasTest {
			testManga.Source = "invalid"
			err := k.AddManga(testManga)
			if err != nil {
				if !util.ErrorContains(err, "Unknown source") {
					t.Fatalf("Unknown rrror while adding manga: %v", err)
				}
			} else {
				t.Fatalf("Error is nil when it shouldn't")
			}
		}
	})
	t.Run("Test add manga with invalid name", func(t *testing.T) {
		for _, testManga := range mangasTest {
			testManga.Name = "invalid12345"
			err := k.AddManga(testManga)
			if err != nil {
				if !util.ErrorContains(err, fmt.Sprintf("Cannot find the %s.", testManga.Name)) {
					t.Fatalf("Unknown rrror while adding manga: %v", err)
				}
			} else {
				t.Fatalf("Error is nil when it shouldn't")
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
			t.Fatalf("Error while getting mangas: %v", err)
		}

		if len(mangas) == 0 {
			t.Fatalf("No mangas found")
		}

		for _, manga := range mangas {
			if manga.ID == 0 {
				t.Fatalf("Manga ID not found")
			}
			if manga.Title == "" {
				t.Fatalf("Manga title not found")
			}
			if manga.Source == "" {
				t.Fatalf("Manga source not found")
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
				t.Fatalf("Error while getting manga: %v", err)
			}

			if manga.ID == 0 {
				t.Fatalf("Manga ID not found")
			}
			if manga.Title == "" {
				t.Fatalf("Manga title not found")
			}
			if manga.Source == "" {
				t.Fatalf("Manga source not found")
			}
		}
	})
	t.Run("Test get manga with invalid name", func(t *testing.T) {
		for _, testManga := range mangasTest {
			testManga.Name = "invalid"
			_, err := k.GetManga(testManga.Name)
			if err != nil {
				if !util.ErrorContains(err, "Manga not found in Kaizoku") {
					t.Fatalf("Unknown error while adding manga: %v", err)
				}
			} else {
				t.Fatalf("Error is nil when it shouldn't")
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
			t.Fatalf("Error while checking out of sync chapters: %v", err)
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
				t.Fatalf("Error while fixing out of sync chapters: %v", err)
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
				t.Fatalf("Error while getting manga: %v", err)
			}

			err = k.RemoveManga(manga.ID, true)
			if err != nil {
				t.Fatalf("Error while removing manga: %v", err)
			}
		}
	})
}

var queuesTest = []string{
	"downloadQueue",
	"checkChaptersQueue",
	"notificationQueue",
	"updateMetadataQueue",
	"integrationQueue",
	"checkOutOfSyncChaptersQueue",
	"fixOutOfSyncChaptersQueue",
}

func TestGetQueues(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test get queues", func(t *testing.T) {
		queues, err := k.GetQueues()
		if err != nil {
			t.Fatalf("Error while getting queues: %v", err)
		}

		if len(queues) != len(queuesTest) {
			t.Fatalf("Invalid number of queues")
		}
	})
}

func TestGetQueue(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test get queue", func(t *testing.T) {
		for _, queueName := range queuesTest {
			queue, err := k.GetQueue(queueName)
			if err != nil {
				t.Fatalf("Error while getting queue: %v", err)
			}

			if queue.Name != queueName {
				t.Fatalf("Invalid queue name")
			}
		}
	})
}

func TestRetryFailedFixOutOfSyncChaptersQueueJobs(t *testing.T) {
	k := Kaizoku{}
	k.Init()

	t.Run("Test retry failed fix out of sync chapters queue jobs", func(t *testing.T) {
		err := k.RetryFailedFixOutOfSyncChaptersQueueJobs()
		if err != nil {
			t.Fatalf("Error while retrying failed fix out of sync chapters queue jobs: %v", err)
		}
	})
}
