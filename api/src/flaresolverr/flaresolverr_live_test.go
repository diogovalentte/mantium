//go:build live

package flaresolverr

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

// TestLiveTransport gets a page through the FlareSolverr in FLARESOLVERR_ADDRESS.
func TestLiveTransport(t *testing.T) {
	addr := os.Getenv("FLARESOLVERR_ADDRESS")
	if addr == "" {
		t.Skip("FLARESOLVERR_ADDRESS isn't set")
	}
	SetAddress(addr)
	defer SetAddress("")

	client := &http.Client{Transport: Transport(DefaultTimeout), Timeout: DefaultTimeout + clientExtraTime}
	res, err := client.Get("https://httpbin.org/redirect-to?url=/html")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "Herman Melville") {
		t.Fatalf("status %d, body %.300s", res.StatusCode, body)
	}
	if final := res.Request.URL.String(); final != "https://httpbin.org/html" {
		t.Errorf("final URL = %s, want the redirect target", final)
	}
}
