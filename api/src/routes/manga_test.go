package routes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
	"github.com/diogovalentte/mantium/api/src/sources/models"
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

func TestSearchManga(t *testing.T) {
	t.Run("Search valid manga with read chapter", func(t *testing.T) {
		body := map[string]string{
			"q":          "yotsubato",
			"source_url": "https://mangaupdates.com",
		}
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string][]*models.MangaSearchResult
		err = requestHelper(http.MethodPost, "/v1/mangas/search", bytes.NewBuffer(payload), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["mangas"]
		if len(actual) < 1 {
			t.Fatalf(`expected at least 1 manga, got %d`, len(actual))
		}
	})
}

func TestGetMangaMetadata(t *testing.T) {
	t.Run("Get metadata of valid manga", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		var resMap map[string]*manga.Manga
		err := requestHelper(http.MethodGet, fmt.Sprintf("/v1/manga/metadata?url=%s", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["manga"]
		if actual.URL != test.URL {
			t.Fatalf(`expected manga with URL "%s", got manga with URL "%s". Response text: %v`, test.URL, actual.URL, resMap)
		}
	})
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
		if !strings.Contains(actual, expected) {
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
		if !strings.Contains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
}

var mangaSearchRequestsTestTable = []routes.SearchMangaRequest{
	{
		Term:      "Death Note",
		SourceURL: "https://mangadex.org",
	},
	{
		Term:      "Blue Box",
		SourceURL: "https://comick.io",
	},
	{
		Term:      "one piece",
		SourceURL: "https://mangahub.io",
	},
	{
		Term:      "dandadan",
		SourceURL: "https://mangaplus.shueisha.co.jp",
	},
}

func TestSearchMangas(t *testing.T) {
	t.Run("Search mangas", func(t *testing.T) {
		for _, test := range mangaSearchRequestsTestTable {
			body, err := json.Marshal(test)
			if err != nil {
				t.Fatal(err)
			}
			var resMap map[string][]*models.MangaSearchResult
			err = requestHelper(http.MethodPost, "/v1/mangas/search", bytes.NewBuffer(body), &resMap)
			if err != nil {
				t.Fatal(err)
			}
			actual := resMap["mangas"]
			if len(actual) < 1 {
				t.Fatalf(`expected at least 1 manga, got %d`, len(actual))
			}
		}
	})
}

func TestGetManga(t *testing.T) {
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
		if !strings.Contains(actual, expected) {
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
		if !strings.Contains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
}

func TestGetMangasiFrame(t *testing.T) {
	t.Run("Get the mangas iframe", func(t *testing.T) {
		url := "/v1/mangas/iframe?api_url=http://localhost:8080&theme=dark&limit=10"
		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("error creating request: %s", err)
		}

		router := api.SetupRouter()
		router.ServeHTTP(w, req)
		if err != nil {
			t.Fatal(err)
		}

		htmlResp := w.Body.String()
		if !strings.Contains(htmlResp, "Mantium") {
			t.Fatalf(`expected response to contain "Mantium", got %s`, htmlResp)
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
		if !strings.Contains(actual, expected) {
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
	t.Run("Update a manga name", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		newName := "Test"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/name?url=%s&name=%s", test.URL, newName), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga name updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update a manga URL", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		newURL := "https://newsite.com/new_manga123"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/url?url=%s&new_url=%s", test.URL, newURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga URL updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}

		err = requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/url?url=%s&new_url=%s", newURL, test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual = resMap["message"]
		expected = "Manga URL updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update the last read chapter of an existing manga to a specific chapter", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		body, err := json.Marshal(routes.UpdateMangaChapterRequest{Chapter: "14"}) // not all sources allow to get a chapter metadata using its chapter number/name
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
	t.Run("Update the last read chapter of an existing manga to the last release chapter", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		body, err := json.Marshal(map[string]string{}) // request needs to have a body, but it can be empty
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
		if !strings.Contains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
	t.Run("Update a manga cover img using URL", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		coverImgURL := "https://i.imgur.com/jMy7evE.jpeg"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/cover_img?url=%s&cover_img_url=%s", test.URL, coverImgURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga cover image updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update a manga cover img with a file", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]

		coverImg, err := os.ReadFile("../../../defaults/default_cover_img.jpg")
		if err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fileWriter, err := w.CreateFormFile("cover_img", "test.jpg")
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(fileWriter, bytes.NewReader(coverImg))
		if err != nil {
			t.Fatal(err)
		}
		w.Close()

		var resMap map[string]string
		url := fmt.Sprintf("/v1/manga/cover_img?url=%s", test.URL)

		rw := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodPatch, url, &b)
		if err != nil {
			t.Fatalf("error creating request: %s", err)
		}
		req.Header.Set("Content-Type", w.FormDataContentType())

		router := api.SetupRouter()
		router.ServeHTTP(rw, req)

		jsonBytes := rw.Body.Bytes()
		if err := json.Unmarshal(jsonBytes, &resMap); err != nil {
			t.Fatalf("error unmarshaling JSON: %s\nreponse text: %s", err.Error(), string(jsonBytes))
		}

		actual := resMap["message"]
		expected := "Manga cover image updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update a manga cover img getting from source site", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/cover_img?url=%s&get_cover_img_from_source=true", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga cover image updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't update a manga cover img because no args", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/cover_img?url=%s", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "you must provide one of the following: cover_img, cover_img_url, get_cover_img_from_source, use_mantium_default_img"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't update a manga cover img because 2 args", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		coverImgURL := "https://i.imgur.com/jMy7evE.jpeg"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/cover_img?url=%s&get_cover_img_from_source=true&cover_img_url=%s", test.URL, coverImgURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "you must provide only one of the following: cover_img, cover_img_url, get_cover_img_from_source, use_mantium_default_img"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't update a manga cover img because invalid image URL", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		coverImgURL := "https://site.com/jMy7evE.jpeg"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/cover_img?url=%s&cover_img_url=%s", test.URL, coverImgURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "error downloading image 'https://site.com/jMy7evE.jpeg'"
		if !strings.Contains(actual, expected) {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
}

func TestGetMangas(t *testing.T) {
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
		if !strings.Contains(actual, expected) {
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
		if !strings.Contains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
}

var addCustomMangaRequest = &routes.AddCustomMangaRequest{
	Name:                 "Custom Manga",
	URL:                  "https://customsite.com/title/custom_manga123",
	Status:               2,
	MangaHasMoreChapters: false,
	CoverImgURL:          "",
	CoverImg:             []byte{},
	NextChapter: &struct {
		Chapter string `json:"chapter"`
		URL     string `json:"chapter_url" binding:"http_url"`
	}{
		Chapter: "10",
		URL:     "https://customsite.com/title/custom_manga123/chapter/10",
	},
}

func TestCustomMangaLifeCycle(t *testing.T) {
	t.Run("Add custom manga", func(t *testing.T) {
		body, err := json.Marshal(addCustomMangaRequest)
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPost, "/v1/custom_manga", bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga added successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Get custom manga", func(t *testing.T) {
		test := addCustomMangaRequest
		var resMap map[string]manga.Manga
		err := requestHelper(http.MethodGet, fmt.Sprintf("/v1/manga?url=%s", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["manga"]
		if actual.URL != test.URL || actual.Status != manga.Status(test.Status) {
			t.Fatalf(`expected manga with URL "%s" and status "%d", got manga with URL "%s" and status "%d". Response text: %v`, test.URL, test.Status, actual.URL, actual.Status, resMap)
		}
	})
	t.Run("Don't get custom manga with invalid URL", func(t *testing.T) {
		test := addCustomMangaRequest
		var resMap map[string]string
		err := requestHelper(http.MethodGet, fmt.Sprintf("/v1/manga?url=%s", test.URL+"salt"), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := errordefs.ErrMangaNotFoundDB.Error()
		if !strings.Contains(actual, expected) {
			t.Fatalf(`expected actual message "%s" to contain expected message "%s"`, actual, expected)
		}
	})
	t.Run("Update custom manga status", func(t *testing.T) {
		test := addCustomMangaRequest
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
	t.Run("Update custom manga name", func(t *testing.T) {
		test := addCustomMangaRequest
		newName := "Test"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/name?url=%s&name=%s", test.URL, newName), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga name updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update custom manga URL", func(t *testing.T) {
		test := addCustomMangaRequest
		newURL := "https://newsite.com/new_manga123"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/url?url=%s&new_url=%s", test.URL, newURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga URL updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}

		err = requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/url?url=%s&new_url=%s", newURL, test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual = resMap["message"]
		expected = "Manga URL updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update the chapter of the last read chapter of custom manga", func(t *testing.T) {
		test := addCustomMangaRequest
		body, err := json.Marshal(routes.UpdateMangaChapterRequest{Chapter: "14"}) // not all sources allow to get a chapter metadata using its chapter number/name
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
	t.Run("Update custom manga has no more chapters to true", func(t *testing.T) {
		test := addCustomMangaRequest
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/custom_manga/has_more_chapters?url=%s&has_more_chapters=true", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Custom manga updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update custom manga has no more chapters to false", func(t *testing.T) {
		test := addCustomMangaRequest
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/custom_manga/has_more_chapters?url=%s&has_more_chapters=false", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Custom manga updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update custom manga cover img using URL", func(t *testing.T) {
		test := addCustomMangaRequest
		coverImgURL := "https://i.imgur.com/jMy7evE.jpeg"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/cover_img?url=%s&cover_img_url=%s", test.URL, coverImgURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga cover image updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update custom manga cover img with a file", func(t *testing.T) {
		test := addCustomMangaRequest
		coverImg, err := os.ReadFile("../../../defaults/default_cover_img.jpg")
		if err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fileWriter, err := w.CreateFormFile("cover_img", "test.jpg")
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(fileWriter, bytes.NewReader(coverImg))
		if err != nil {
			t.Fatal(err)
		}
		w.Close()

		var resMap map[string]string
		url := fmt.Sprintf("/v1/manga/cover_img?url=%s", test.URL)

		rw := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodPatch, url, &b)
		if err != nil {
			t.Fatalf("error creating request: %s", err)
		}
		req.Header.Set("Content-Type", w.FormDataContentType())

		router := api.SetupRouter()
		router.ServeHTTP(rw, req)

		jsonBytes := rw.Body.Bytes()
		if err := json.Unmarshal(jsonBytes, &resMap); err != nil {
			t.Fatalf("error unmarshaling JSON: %s\nreponse text: %s", err.Error(), string(jsonBytes))
		}

		actual := resMap["message"]
		expected := "Manga cover image updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't update custom manga cover img because no args", func(t *testing.T) {
		test := addCustomMangaRequest
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/cover_img?url=%s", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "you must provide one of the following: cover_img, cover_img_url, get_cover_img_from_source, use_mantium_default_img"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't update custom manga cover img because 2 args", func(t *testing.T) {
		test := addCustomMangaRequest
		coverImgURL := "https://i.imgur.com/jMy7evE.jpeg"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/cover_img?url=%s&use_mantium_default_img=true&cover_img_url=%s", test.URL, coverImgURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "you must provide only one of the following: cover_img, cover_img_url, get_cover_img_from_source, use_mantium_default_img"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't update custom manga cover img because invalid image URL", func(t *testing.T) {
		test := addCustomMangaRequest
		coverImgURL := "https://site.com/jMy7evE.jpeg"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/cover_img?url=%s&cover_img_url=%s", test.URL, coverImgURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "error downloading image 'https://site.com/jMy7evE.jpeg'"
		if !strings.Contains(actual, expected) {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't update custom manga cover img to get from source", func(t *testing.T) {
		test := addCustomMangaRequest
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/manga/cover_img?url=%s&get_cover_img_from_source=true", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "you can't get the cover image from the source for custom mangas"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Delete custom manga", func(t *testing.T) {
		test := addCustomMangaRequest
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
}

func TestMultiMangaLifeCycle(t *testing.T) {
	multimanga := &manga.MultiManga{}

	t.Run("Add valid manga with read chapter to turn into multimanga", func(t *testing.T) {
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
	t.Run("Should turn a manga into a multimanga into DB", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		var resMap map[string]string
		err := requestHelper(http.MethodPost, fmt.Sprintf("/v1/manga/turn_into_multimanga?url=%s", test.URL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga turned into multimanga successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Get mangas", func(t *testing.T) {
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
			if m.MultiMangaID != 0 {
				multimanga.ID = m.MultiMangaID
			}
		}
	})
	t.Run("Get a multimanga 1", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		var resMap map[string]manga.MultiManga
		err := requestHelper(http.MethodGet, fmt.Sprintf("/v1/multimanga?id=%d", multimanga.ID), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["multimanga"]
		if actual.Status != manga.Status(test.Status) {
			t.Fatalf(`expected manga with status "%d", got manga with status "%d". Response text: %v`, test.Status, actual.Status, resMap)
		}
		multimanga = &actual
		if len(multimanga.Mangas) != 1 {
			t.Fatalf(`expected multimanga to have 1 manga, got %d`, len(multimanga.Mangas))
		}
	})
	t.Run("Add manga to multimanga's manga list", func(t *testing.T) {
		body, err := json.Marshal(routes.AddMangaToMultiMangaRequest{MangaURL: "https://mangadex.org/title/68112dc1-2b80-4f20-beb8-2f2a8716a430"})
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPost, fmt.Sprintf("/v1/multimanga/manga?id=%d", multimanga.ID), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga added to multimanga successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Get mangas 2", func(t *testing.T) {
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
			if m.MultiMangaID != 0 {
				multimanga.ID = m.MultiMangaID
			}
		}
	})
	t.Run("Get a multimanga 2", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		var resMap map[string]manga.MultiManga
		err := requestHelper(http.MethodGet, fmt.Sprintf("/v1/multimanga?id=%d", multimanga.ID), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["multimanga"]
		if actual.Status != manga.Status(test.Status) {
			t.Fatalf(`expected manga with status "%d", got manga with status "%d". Response text: %v`, test.Status, actual.Status, resMap)
		}
		multimanga = &actual
		if len(multimanga.Mangas) != 2 {
			t.Fatalf(`expected multimanga to have 2 manga, got %d`, len(multimanga.Mangas))
		}
	})
	t.Run("Remove manga from multimanga's manga list", func(t *testing.T) {
		var resMap map[string]string
		err := requestHelper(http.MethodDelete, fmt.Sprintf("/v1/multimanga/manga?id=%d&manga_id=%d", multimanga.ID, multimanga.CurrentManga.ID), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Manga removed from multimanga successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Get a multimanga 3", func(t *testing.T) {
		test := mangasRequestsTestTable["valid manga with read chapter"]
		var resMap map[string]manga.MultiManga
		err := requestHelper(http.MethodGet, fmt.Sprintf("/v1/multimanga?id=%d", multimanga.ID), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["multimanga"]
		if actual.Status != manga.Status(test.Status) {
			t.Fatalf(`expected manga with status "%d", got manga with status "%d". Response text: %v`, test.Status, actual.Status, resMap)
		}
		multimanga = &actual
		if len(multimanga.Mangas) != 1 {
			t.Fatalf(`expected multimanga to have 1 manga, got %d`, len(multimanga.Mangas))
		}
	})
	t.Run("Don't remove last manga from multimanga's manga list", func(t *testing.T) {
		var resMap map[string]string
		err := requestHelper(http.MethodDelete, fmt.Sprintf("/v1/multimanga/manga?id=%d&manga_id=%d", multimanga.ID, multimanga.CurrentManga.ID), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := errordefs.ErrAttemptedToRemoveLastMultiMangaManga.Error()
		if !strings.Contains(actual, expected) {
			t.Fatalf(`expected message to contain "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Get chapters of a multimanga", func(t *testing.T) {
		var resMap map[string][]manga.Chapter
		err := requestHelper(http.MethodGet, fmt.Sprintf("/v1/multimanga/chapters?id=%d&manga_id=%d", multimanga.ID, multimanga.CurrentManga.ID), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		chapters := resMap["chapters"]
		if len(chapters) < 1 {
			t.Fatalf(`expected at least 1 chapter, got %d`, len(chapters))
		}
	})
	t.Run("Update a multimanga status", func(t *testing.T) {
		body, err := json.Marshal(routes.UpdateMangaStatusRequest{Status: 4})
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPatch, fmt.Sprintf("/v1/multimanga/status?id=%d", multimanga.ID), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Multimanga status updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update the last read chapter of a multimanga to a specific chapter", func(t *testing.T) {
		body, err := json.Marshal(routes.UpdateMangaChapterRequest{Chapter: "14"}) // not all sources allows to get a chapter metadata using its chapter number/name
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPatch, fmt.Sprintf("/v1/multimanga/last_read_chapter?id=%d&manga_id=%d", multimanga.ID, multimanga.CurrentManga.ID), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Multimanga last read chapter updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update the last read chapter of a multimanga to the last released chapter", func(t *testing.T) {
		body, err := json.Marshal(map[string]string{}) // request needs to have a body, but it can be empty
		if err != nil {
			t.Fatal(err)
		}
		var resMap map[string]string
		err = requestHelper(http.MethodPatch, fmt.Sprintf("/v1/multimanga/last_read_chapter?id=%d&manga_id=%d", multimanga.ID, multimanga.CurrentManga.ID), bytes.NewBuffer(body), &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Multimanga last read chapter updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update a multimanga cover img using URL", func(t *testing.T) {
		coverImgURL := "https://i.imgur.com/jMy7evE.jpeg"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/multimanga/cover_img?id=%d&cover_img_url=%s", multimanga.ID, coverImgURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Multimanga cover image updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update a multimanga cover img with a file", func(t *testing.T) {
		coverImg, err := os.ReadFile("../../../defaults/default_cover_img.jpg")
		if err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fileWriter, err := w.CreateFormFile("cover_img", "test.jpg")
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(fileWriter, bytes.NewReader(coverImg))
		if err != nil {
			t.Fatal(err)
		}
		w.Close()

		var resMap map[string]string
		url := fmt.Sprintf("/v1/multimanga/cover_img?id=%d", multimanga.ID)

		rw := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodPatch, url, &b)
		if err != nil {
			t.Fatalf("error creating request: %s", err)
		}
		req.Header.Set("Content-Type", w.FormDataContentType())

		router := api.SetupRouter()
		router.ServeHTTP(rw, req)

		jsonBytes := rw.Body.Bytes()
		if err := json.Unmarshal(jsonBytes, &resMap); err != nil {
			t.Fatalf("error unmarshaling JSON: %s\nreponse text: %s", err.Error(), string(jsonBytes))
		}

		actual := resMap["message"]
		expected := "Multimanga cover image updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Update a multimanga cover img to use current manga's cover image", func(t *testing.T) {
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/multimanga/cover_img?id=%d&use_current_manga_cover_img=true", multimanga.ID), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Multimanga cover image updated successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't update a multimanga cover img because no args", func(t *testing.T) {
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/multimanga/cover_img?id=%d", multimanga.ID), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "you must provide one of the following: cover_img, cover_img_url, use_current_manga_cover_img"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't update a multimanga cover img because 2 args", func(t *testing.T) {
		coverImgURL := "https://i.imgur.com/jMy7evE.jpeg"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/multimanga/cover_img?id=%d&use_current_manga_cover_img=true&cover_img_url=%s", multimanga.ID, coverImgURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "you must provide only one of the following: cover_img, cover_img_url, use_current_manga_cover_img"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
		}
	})
	t.Run("Don't update a multimanga cover img because invalid image URL", func(t *testing.T) {
		coverImgURL := "https://site.com/jMy7evE.jpeg"
		var resMap map[string]string
		err := requestHelper(http.MethodPatch, fmt.Sprintf("/v1/multimanga/cover_img?id=%d&cover_img_url=%s", multimanga.ID, coverImgURL), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "error downloading image 'https://site.com/jMy7evE.jpeg'"
		if !strings.Contains(actual, expected) {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
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
	t.Run("Delete multimanga", func(t *testing.T) {
		var resMap map[string]string
		err := requestHelper(http.MethodDelete, fmt.Sprintf("/v1/multimanga?id=%d", multimanga.ID), nil, &resMap)
		if err != nil {
			t.Fatal(err)
		}

		actual := resMap["message"]
		expected := "Multimanga deleted successfully"
		if actual != expected {
			t.Fatalf(`expected message "%s", got "%s"`, expected, actual)
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

func TestNotifyMangaLastReleasedChapterUpdate(t *testing.T) {
	t.Run("Notify manga last released chapter update", func(t *testing.T) {
		m := &manga.Manga{
			Name: "One Piece",
			LastReleasedChapter: &manga.Chapter{
				Chapter: "1001",
				URL:     "https://mangahub.io/chapter/one-piece_142/chapter-1001",
			},
		}

		err := routes.NotifyMangaLastReleasedChapterUpdate(m)
		if err != nil {
			t.Fatal(err)
		}
	})
}
