package routes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/diogovalentte/mantium/api/src"
	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/routes"
)

func setup() error {
	err := config.SetConfigs("../../../.env.test")
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

var mangasRequestsTestTable = map[string]routes.AddMangaRequest{
	"valid manga with read chapter": {
		URL:             "https://comick.io/comic/dandadan",
		Status:          5,
		LastReadChapter: "154",
	},
	"invalid manga URL": {
		URL:    "https://mangahub.io/manga/beeerserkk",
		Status: 4,
	},
	"invalid chapter": {
		URL:             "https://mangahub.io/manga/the-twin-swords-of-the-sima",
		Status:          4,
		LastReadChapter: "1000",
	},
}

func TestAddManga(t *testing.T) {
	t.Run("Add valid manga with read chapter", func(t *testing.T) {
		body, err := json.Marshal(mangasRequestsTestTable["valid manga with read chapter"])
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPost, "/v1/manga", bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga added successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't add manga with invalid URL", func(t *testing.T) {
		body, err := json.Marshal(mangasRequestsTestTable["invalid manga URL"])
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPost, "/v1/manga", bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := errordefs.ErrMangaNotFound.Error()
		if !strContains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
	t.Run("Don't add manga with invalid last read chapter", func(t *testing.T) {
		body, err := json.Marshal(mangasRequestsTestTable["invalid chapter"])
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPost, "/v1/manga", bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := errordefs.ErrChapterNotFound.Error()
		if !strContains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
}

func TestGetMangas(t *testing.T) {
	t.Run("Get one manga with read chapter", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		body, err := json.Marshal(test)
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]manga.Manga
		err = requestHelper(http.MethodGet, fmt.Sprintf("/v1/manga?url=%s", test.URL), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["manga"]
		if actual.URL != test.URL || actual.Status != manga.Status(test.Status) {
			t.Fatalf(`expected manga with URL "%s" and status "%d", got manga with URL "%s" and status "%d". Response text: %v`, test.URL, test.Status, actual.URL, actual.Status, resMap)
		}
	})
	t.Run("Don't get one manga with invalid URL", func(t *testing.T) {
		test := mangasRequestsTestTable["invalid manga URL"]
		body, err := json.Marshal(test)
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodGet, fmt.Sprintf("/v1/manga?url=%s", test.URL), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := errordefs.ErrMangaNotFoundDB.Error()
		if !strContains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
	t.Run("Don't get one manga with invalid last read chapter", func(t *testing.T) {
		test := mangasRequestsTestTable["invalid chapter"]
		body, err := json.Marshal(test)
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodGet, fmt.Sprintf("/v1/manga?url=%s", test.URL), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := errordefs.ErrMangaNotFoundDB.Error()
		if !strContains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
	t.Run("Get mangas from DB", func(t *testing.T) {
		var resMap map[string][]manga.Manga
		err := requestHelper(http.MethodGet, "/v1/mangas", nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		mangas := resMap["mangas"]
		if len(mangas) < 1 {
			t.Fatalf(`expected at least 1 manga, got %d`, len(mangas))
		}
		for _, m := range mangas {
			if m.URL == "" || m.Status == 0 {
				t.Fatalf(`expected all mangas to have a URL and a status, got %v`, mangas)
			}
		}
	})
}

func TestGetMangaChapters(t *testing.T) {
	t.Run("Get manga chapters", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		body, err := json.Marshal(test)
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string][]manga.Chapter
		err = requestHelper(http.MethodGet, fmt.Sprintf("/v1/manga/chapters?url=%s", test.URL), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		chapters := resMap["chapters"]
		if len(chapters) < 1 {
			t.Fatalf(`expected at least 1 chapter, got %d`, len(chapters))
		}
	})
	t.Run("Don't get chapters of manga with invalid URL", func(t *testing.T) {
		test := mangasRequestsTestTable["invalid manga URL"]
		body, err := json.Marshal(test)
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodGet, fmt.Sprintf("/v1/manga/chapters?url=%s", test.URL), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := errordefs.ErrMangaNotFound.Error()
		if !strContains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
}

func TestUpdateManga(t *testing.T) {
	t.Run("Update a manga status", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		body, err := json.Marshal(routes.UpdateMangaStatusRequest{Status: 4})
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/status?url=%s", test.URL), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga status updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update the last read chapter of an existing manga", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		body, err := json.Marshal(routes.UpdateMangaChapterRequest{Chapter: "14"})
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/last_read_chapter?url=%s", test.URL), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga last read chapter updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update the last read chapter of an non existing manga", func(t *testing.T) {
		test := mangasRequestsTestTable["invalid manga URL"]
		body, err := json.Marshal(routes.UpdateMangaChapterRequest{Chapter: "14"})
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/last_read_chapter?url=%s", test.URL), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := errordefs.ErrMangaNotFoundDB.Error()
		if !strContains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
}

func TestDeleteManga(t *testing.T) {
	t.Run("Delete valid manga with read chapter", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		var resMap map[string]string
		err := requestHelper(http.MethodDelete, fmt.Sprintf("/v1/manga?url=%s", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga deleted successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't delete manga with invalid URL", func(t *testing.T) {
		test := mangasRequestsTestTable["invalid manga URL"]
		var resMap map[string]string
		err := requestHelper(http.MethodDelete, fmt.Sprintf("/v1/manga?url=%s", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := errordefs.ErrMangaNotFoundDB.Error()
		if !strContains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
	t.Run("Don't delete manga with invalid last read chapter", func(t *testing.T) {
		test := mangasRequestsTestTable["invalid chapter"]
		var resMap map[string]string
		err := requestHelper(http.MethodDelete, fmt.Sprintf("/v1/manga?url=%s", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := errordefs.ErrMangaNotFoundDB.Error()
		if !strContains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
}

func TestUpdateMangasMetadata(t *testing.T) {
	t.Run("Update all mangas metadata", func(t *testing.T) {
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, "/v1/mangas/metadata", nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Mangas metadata updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
}

func requestHelper(method, url string, body io.Reader, target interface{}) error {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	router := api.SetupRouter()
	router.ServeHTTP(w, req)

	jsonBytes := w.Body.Bytes()
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("error unmarshaling JSON: %s\nreponse text: %s", err.Error(), string(jsonBytes))
	}

	return nil
}

func strContains(s1, s2 string) bool {
	return strings.Contains(s1, s2)
}

func TestNotifyMangaLastReleasedChapterUpdate(t *testing.T) {
	t.Run("Notify manga last released chapter update", func(t *testing.T) {
		oldManga := &manga.Manga{
			Name: "One Piece",
			LastReleasedChapter: &manga.Chapter{
				Chapter: "1000",
				URL:     "https://mangahub.io/chapter/one-piece_142/chapter-1000",
			},
		}
		newManga := &manga.Manga{
			Name: "One Piece",
			LastReleasedChapter: &manga.Chapter{
				Chapter: "1001",
				URL:     "https://mangahub.io/chapter/one-piece_142/chapter-1001",
			},
		}

		err := routes.NotifyMangaLastReleasedChapterUpdate(oldManga, newManga)
		if err != nil {
			t.Fatal(err)
		}
	})
}
