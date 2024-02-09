package routes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/diogovalentte/manga-dashboard-api/api/src"
	"github.com/diogovalentte/manga-dashboard-api/api/src/manga"
	"github.com/diogovalentte/manga-dashboard-api/api/src/routes"
)

func setup() error {
	err := godotenv.Load("../../../../.env")
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
		URL:             "https://mangahub.io/manga/berserk",
		Status:          5,
		LastReadChapter: "370",
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
	router := api.SetupRouter()

	t.Run("Add valid manga with read chapter", func(t *testing.T) {
		err := testAddMangaRouteHelper("valid manga with read chapter", router, "Manga added successfully")
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Don't add manga with invalid URL", func(t *testing.T) {
		err := testAddMangaRouteHelper("invalid manga URL", router, "error while getting manga metadata from source: manga not found, is the URL correct?")
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Don't add manga with invalid last read chapter", func(t *testing.T) {
		err := testAddMangaRouteHelper("invalid chapter", router, "error while getting chapter metadata from source: chapter not found, is the URL or chapter number correct?")
		if err != nil {
			t.Error(err)
		}
	})
}

func TestGetMangas(t *testing.T) {
	router := api.SetupRouter()

	t.Run("Get one manga with read chapter", func(t *testing.T) {
		err := testGetMangaRouteHelper("valid manga with read chapter", router)
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Don't get one manga with invalid URL", func(t *testing.T) {
		err := testGetMangaRouteHelper("invalid manga URL", router)
		if err != nil {
			if err.Error() == "error getting manga from DB: manga not found in DB" {
				return
			}
			t.Error(err)
		}
	})
	t.Run("Don't get one manga with invalid last read chapter", func(t *testing.T) {
		err := testGetMangaRouteHelper("invalid chapter", router)
		if err != nil {
			if err.Error() == "error getting manga from DB: manga not found in DB" {
				return
			}
			t.Error(err)
		}
	})
	t.Run("Get mangas from DB", func(t *testing.T) {
		err := testGetMangasRouteHelper(router)
		if err != nil {
			t.Error(err)
		}
	})
}

func TestGetMangaChapters(t *testing.T) {
	router := api.SetupRouter()

	t.Run("Get manga chapters", func(t *testing.T) {
		err := testGetMangaChaptersRouteHelper("valid manga with read chapter", router)
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Don't get one manga with invalid URL", func(t *testing.T) {
		err := testGetMangaRouteHelper("invalid manga URL", router)
		if err != nil {
			if err.Error() == "error getting manga from DB: manga not found in DB" {
				return
			}
			t.Error(err)
		}
	})
}

func TestUpdateManga(t *testing.T) {
	router := api.SetupRouter()

	t.Run("Update a manga status", func(t *testing.T) {
		err := testUpdateMangaRouteHelper("valid manga with read chapter", "status", routes.UpdateMangaStatusRequest{Status: 4}, router, "Manga status updated successfully")
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Update the last read chapter of an existing manga", func(t *testing.T) {
		err := testUpdateMangaRouteHelper("valid manga with read chapter", "last_read_chapter", routes.UpdateMangaChapterRequest{Chapter: "14"}, router, "Manga last read chapter updated successfully")
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Update the last read chapter of an non existing manga", func(t *testing.T) {
		err := testUpdateMangaRouteHelper("invalid manga URL", "last_read_chapter", routes.UpdateMangaChapterRequest{Chapter: "14"}, router, "error getting manga from DB: manga not found in DB")
		if err != nil {
			t.Error(err)
		}
	})
}

func TestUpdateMangasMetadata(t *testing.T) {
	router := api.SetupRouter()

	t.Run("Update all mangas metadata", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodPatch, "/v1/mangas/metadata", nil)
		if err != nil {
			t.Error(err)
		}
		router.ServeHTTP(w, req)

		var resMap map[string]string
		jsonBytes := w.Body.Bytes()
		if err := json.Unmarshal(jsonBytes, &resMap); err != nil {
			t.Error(err)
		}

		actual := resMap["message"]
		expected := "Mangas metadata updated successfully"
		if actual != expected {
			t.Errorf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
}

func TestDeleteManga(t *testing.T) {
	router := api.SetupRouter()

	t.Run("Delete valid manga with read chapter", func(t *testing.T) {
		err := testDeleteMangaRouteHelper("valid manga with read chapter", router, "Manga deleted successfully")
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Don't delete manga with invalid URL", func(t *testing.T) {
		err := testDeleteMangaRouteHelper("invalid manga URL", router, "error getting manga from DB: manga not found in DB")
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Don't delete manga with invalid last read chapter", func(t *testing.T) {
		err := testDeleteMangaRouteHelper("invalid chapter", router, "error getting manga from DB: manga not found in DB")
		if err != nil {
			t.Error(err)
		}
	})
}

func testAddMangaRouteHelper(testKey string, router *gin.Engine, expectedMessage string) error {
	test, ok := mangasRequestsTestTable[testKey]
	if !ok {
		return fmt.Errorf("test key not found in tests map")
	}

	requestBody, err := json.Marshal(test)
	if err != nil {
		return err
	}

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/v1/manga", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	router.ServeHTTP(w, req)

	var resMap map[string]string
	jsonBytes := w.Body.Bytes()
	if err := json.Unmarshal(jsonBytes, &resMap); err != nil {
		return err
	}

	actual := resMap["message"]
	if actual != expectedMessage {
		return fmt.Errorf(`expected message "%s", got "%s"`, expectedMessage, actual)
	}

	return nil
}

func testGetMangaRouteHelper(testKey string, router *gin.Engine) error {
	test, ok := mangasRequestsTestTable[testKey]
	if !ok {
		return fmt.Errorf("test key not found in tests map")
	}

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/v1/manga?url=%s", test.URL), nil)
	if err != nil {
		return err
	}
	router.ServeHTTP(w, req)

	var resMap map[string]manga.Manga
	jsonBytes := w.Body.Bytes()
	if err := json.Unmarshal(jsonBytes, &resMap); err != nil {
		// response is an error message
		ms := err.Error()
		if ms == "json: cannot unmarshal string into Go value of type manga.Manga" {
			var errMap map[string]string
			jsonBytes := w.Body.Bytes()
			if err := json.Unmarshal(jsonBytes, &errMap); err != nil {
				return err
			}
			msg, ok := errMap["message"]
			if !ok {
				return fmt.Errorf(`response is a string and does not have the "message" field`)
			}
			return fmt.Errorf(msg)
		}
		return err
	}

	actual := resMap["manga"]
	if actual.URL != test.URL || actual.Status != manga.Status(test.Status) {
		return fmt.Errorf(`expected manga with URL "%s" and status "%d", got manga with URL "%s" and status "%d"`, test.URL, test.Status, actual.URL, actual.Status)
	}

	return nil
}

func testGetMangaChaptersRouteHelper(testKey string, router *gin.Engine) error {
	test, ok := mangasRequestsTestTable[testKey]
	if !ok {
		return fmt.Errorf("test key not found in tests map")
	}

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/v1/manga/chapters?url=%s", test.URL), nil)
	if err != nil {
		return err
	}
	router.ServeHTTP(w, req)

	var resMap map[string][]manga.Chapter
	jsonBytes := w.Body.Bytes()
	if err := json.Unmarshal(jsonBytes, &resMap); err != nil {
		// response is an error message
		ms := err.Error()
		if ms == "json: cannot unmarshal string into Go value of type manga.Manga" {
			var errMap map[string]string
			jsonBytes := w.Body.Bytes()
			if err := json.Unmarshal(jsonBytes, &errMap); err != nil {
				return err
			}
			msg, ok := errMap["message"]
			if !ok {
				return fmt.Errorf(`response is a string and does not have the "message" field`)
			}
			return fmt.Errorf(msg)
		}
		return err
	}

	actual := resMap["chapters"]
	if len(actual) == 0 {
		return fmt.Errorf(`expected manga with URL "%s" to have chapters, got none`, test.URL)
	}

	return nil
}

func testGetMangasRouteHelper(router *gin.Engine) error {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/v1/mangas", nil)
	if err != nil {
		return err
	}
	router.ServeHTTP(w, req)

	var resMap map[string][]manga.Manga
	jsonBytes := w.Body.Bytes()
	if err := json.Unmarshal(jsonBytes, &resMap); err != nil {
		// response is an error message
		ms := err.Error()
		if ms == "json: cannot unmarshal string into Go value of type []manga.Manga" {
			var errMap map[string]string
			jsonBytes := w.Body.Bytes()
			if err := json.Unmarshal(jsonBytes, &errMap); err != nil {
				return err
			}
			msg, ok := errMap["message"]
			if !ok {
				return fmt.Errorf(`response is a string and does not have the "message" field`)
			}
			return fmt.Errorf(msg)
		}
		return err
	}

	mangas := resMap["mangas"]
	// hardcoded mangas length
	if len(mangas) != 1 {
		return fmt.Errorf(`expected 1 mangas, got %d`, len(mangas))
	}

	return nil
}

func testUpdateMangaRouteHelper(testKey string, propertyToUpdate string, newValue interface{}, router *gin.Engine, expectedMessage string) error {
	test, ok := mangasRequestsTestTable[testKey]
	if !ok {
		return fmt.Errorf("test key not found in tests map")
	}

	requestBody, err := json.Marshal(newValue)
	if err != nil {
		return err
	}

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/manga/%s?url=%s", propertyToUpdate, test.URL), bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	router.ServeHTTP(w, req)

	var resMap map[string]string
	jsonBytes := w.Body.Bytes()
	if err := json.Unmarshal(jsonBytes, &resMap); err != nil {
		return err
	}

	actual := resMap["message"]
	if actual != expectedMessage {
		return fmt.Errorf(`expected message "%s", got "%s"`, expectedMessage, actual)
	}

	return nil
}

func testDeleteMangaRouteHelper(testKey string, router *gin.Engine, expectedMessage string) error {
	test, ok := mangasRequestsTestTable[testKey]
	if !ok {
		return fmt.Errorf("test key not found in tests map")
	}

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/manga?url=%s", test.URL), nil)
	if err != nil {
		return err
	}
	router.ServeHTTP(w, req)

	var resMap map[string]string
	jsonBytes := w.Body.Bytes()
	if err := json.Unmarshal(jsonBytes, &resMap); err != nil {
		return err
	}

	actual := resMap["message"]
	if actual != expectedMessage {
		return fmt.Errorf(`expected message "%s", got "%s"`, expectedMessage, actual)
	}

	return nil
}

func TestNotifyMangaLastUploadChapterUpdate(t *testing.T) {
	t.Run("Notify manga last upload chapter update", func(t *testing.T) {
		oldManga := &manga.Manga{
			Name: "One Piece",
			LastUploadChapter: &manga.Chapter{
				Chapter: "1000",
				URL:     "https://mangahub.io/chapter/one-piece_142/chapter-1000",
			},
		}
		newManga := &manga.Manga{
			Name: "One Piece",
			LastUploadChapter: &manga.Chapter{
				Chapter: "1001",
				URL:     "https://mangahub.io/chapter/one-piece_142/chapter-1001",
			},
		}

		err := routes.NotifyMangaLastUploadChapterUpdate(oldManga, newManga)
		if err != nil {
			t.Error(err)
			return
		}
	})
}
