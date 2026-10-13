package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/diogovalentte/mantium/api/src"
)

func request(t *testing.T, method, path, body string, status int) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	api.SetupRouter().ServeHTTP(w, req)
	if w.Code != status {
		t.Fatalf("%s %s: status %d, want %d; body %s", method, path, w.Code, status, w.Body.String())
	}
	return w
}

func TestCurrentRoutes(t *testing.T) {
	w := request(t, http.MethodGet, "/v1/health", "", 200)
	if w.Body.String() != "OK" {
		t.Fatalf("unexpected health response: %s", w.Body.String())
	}
	for _, tc := range []struct{ method, path, body string }{
		{"POST", "/v1/multimanga", `{}`},
		{"POST", "/v1/multimanga", `{"status":1}`},
		{"POST", "/v1/multimanga", `{"name":"Test","status":6}`},
		{"POST", "/v1/multimanga", `{"name":"Test","status":1,"cover_img":"aW52YWxpZA=="}`},
		{"GET", "/v1/multimanga?id=abc", ``},
		{"PATCH", "/v1/multimanga/status?id=abc", `{"status":2}`},
		{"PATCH", "/v1/custom_manga/name?id=abc", `{"name":"Renamed"}`},
		{"GET", "/v1/mangas/iframe?limit=abc&api_url=http://localhost", ``},
	} {
		t.Run(tc.method+tc.path+tc.body, func(t *testing.T) {
			w := request(t, tc.method, tc.path, tc.body, 400)
			var message struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &message); err != nil {
				t.Fatal(err)
			}
			if message.Message == "" {
				t.Fatal("missing validation message")
			}
		})
	}
	for _, path := range []string{"/v1/custom_manga", "/v1/manga"} {
		request(t, "POST", path, `{}`, 404)
	}
}
