// Package testutil contains helpers shared by the API tests.
package testutil

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/diogovalentte/mantium/api/src/util"
)

// Fixtures serves saved responses, substituting only the local server URL.
// Unexpected requests fail the test rather than escaping to a real site.
// The server listens on a private address, which covers are only downloaded
// from when private URLs are allowed, so it allows them for the test.
func Fixtures(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	util.SetAllowPrivateURLs(true)
	t.Cleanup(func() { util.SetAllowPrivateURLs(false) })
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/cover.png") {
			w.Header().Set("Content-Type", "image/png")
			if err := png.Encode(w, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
				t.Error(err)
			}
			return
		}
		file, ok := routes[r.URL.Path]
		if !ok {
			t.Errorf("unexpected fixture request: %s %s", r.Method, r.URL)
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Error(err)
			http.Error(w, err.Error(), 500)
			return
		}
		if strings.HasSuffix(file, ".json") {
			w.Header().Set("Content-Type", "application/json")
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		_, _ = w.Write(bytes.ReplaceAll(body, []byte("{{BASE}}"), []byte(server.URL)))
	}))
	t.Cleanup(server.Close)
	return server
}
