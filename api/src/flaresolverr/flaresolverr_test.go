package flaresolverr

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/util"
)

// fakeFlareSolverr starts a server that answers like FlareSolverr's /v1 endpoint with
// the response returned by answer, and counts the requests it gets.
func fakeFlareSolverr(t *testing.T, answer func(req request) (int, response)) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/v1" {
			t.Errorf("FlareSolverr got %s %s, want POST /v1", r.Method, r.URL.Path)
		}
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("FlareSolverr got an invalid body: %s", err)
		}
		status, res := answer(req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(res)
	}))
	t.Cleanup(server.Close)
	SetAddress(server.URL + "/")
	t.Cleanup(func() { SetAddress("") })

	return server, &calls
}

func TestTransportReturnsThePageWithItsFinalURL(t *testing.T) {
	util.SetAllowPrivateURLs(true)
	defer util.SetAllowPrivateURLs(false)

	var got request
	fakeFlareSolverr(t, func(req request) (int, response) {
		got = req
		return http.StatusOK, response{Status: "ok", Solution: solution{
			URL:      "http://example.com/manga/one-piece/",
			Status:   http.StatusOK,
			Response: "<html><body><a href='chapter-1'>Chapter 1</a></body></html>",
		}}
	})

	client := &http.Client{Transport: Transport(30 * time.Second), Timeout: time.Minute}
	res, err := client.Get("http://example.com/one-piece")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if got.Cmd != "request.get" || got.URL != "http://example.com/one-piece" || got.MaxTimeout != 30000 {
		t.Errorf("FlareSolverr got %+v", got)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
	if final := res.Request.URL.String(); final != "http://example.com/manga/one-piece/" {
		t.Errorf("final URL = %s", final)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %s", ct)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "Chapter 1") {
		t.Errorf("body = %s", body)
	}
}

func TestTransportKeepsThePageStatusCode(t *testing.T) {
	util.SetAllowPrivateURLs(true)
	defer util.SetAllowPrivateURLs(false)

	fakeFlareSolverr(t, func(req request) (int, response) {
		return http.StatusOK, response{Status: "ok", Solution: solution{URL: req.URL, Status: http.StatusNotFound, Response: "<html>not found</html>"}}
	})

	res, err := (&http.Client{Transport: Transport(time.Second)}).Get("http://example.com/missing")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", res.StatusCode)
	}
}

func TestGetReturnsFlareSolverrErrorAsUserError(t *testing.T) {
	util.SetAllowPrivateURLs(true)
	defer util.SetAllowPrivateURLs(false)

	fakeFlareSolverr(t, func(request) (int, response) {
		return http.StatusInternalServerError, response{Status: "error", Message: "Error: Error solving the challenge. Timeout after 1.0 seconds."}
	})

	_, err := Get(t.Context(), "http://example.com/", time.Second)
	var userErr *errordefs.UserError
	if !errors.As(err, &userErr) || !strings.Contains(userErr.Message, "Timeout after 1.0 seconds") {
		t.Fatalf("err = %v, want a user error with FlareSolverr's message", err)
	}
}

func TestGetWithoutAddress(t *testing.T) {
	SetAddress("")
	if Enabled() {
		t.Fatal("Enabled() = true without an address")
	}
	_, err := Get(t.Context(), "http://example.com/", time.Second)
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestGetRefusesPrivatePageURL(t *testing.T) {
	_, calls := fakeFlareSolverr(t, func(req request) (int, response) {
		return http.StatusOK, response{Status: "ok", Solution: solution{URL: req.URL, Status: http.StatusOK}}
	})

	_, err := Get(t.Context(), "http://127.0.0.1:8080/admin", time.Second)
	var userErr *errordefs.UserError
	if !errors.As(err, &userErr) || !strings.Contains(userErr.Message, "private address") {
		t.Fatalf("err = %v, want the private address error", err)
	}
	if calls.Load() != 0 {
		t.Errorf("FlareSolverr was called %d times for a private URL", calls.Load())
	}
}

func TestGetRefusesRedirectToPrivateURL(t *testing.T) {
	fakeFlareSolverr(t, func(request) (int, response) {
		return http.StatusOK, response{Status: "ok", Solution: solution{URL: "http://127.0.0.1/admin", Status: http.StatusOK, Response: "secret"}}
	})

	page, err := Get(t.Context(), "http://1.1.1.1/", time.Second)
	if err == nil || !strings.Contains(err.Error(), "private address") {
		t.Fatalf("page = %+v, err = %v, want the private address error", page, err)
	}
}

func TestTransportOnlySupportsGET(t *testing.T) {
	_, calls := fakeFlareSolverr(t, func(request) (int, response) {
		return http.StatusOK, response{Status: "ok"}
	})

	_, err := (&http.Client{Transport: Transport(time.Second)}).Post("http://example.com/", "text/plain", strings.NewReader("x"))
	if err == nil || !strings.Contains(err.Error(), "only supports GET") {
		t.Fatalf("err = %v, want the GET only error", err)
	}
	if calls.Load() != 0 {
		t.Errorf("FlareSolverr was called %d times", calls.Load())
	}
}

func TestGetSendsTheProxyOnlyForItsDomains(t *testing.T) {
	util.SetAllowPrivateURLs(true)
	defer util.SetAllowPrivateURLs(false)
	if err := util.SetProxy("http://gluetun:8888", []string{"example.com"}); err != nil {
		t.Fatal(err)
	}
	defer util.SetProxy("", nil)

	var proxies []string
	fakeFlareSolverr(t, func(req request) (int, response) {
		if req.Proxy == nil {
			proxies = append(proxies, "")
		} else {
			proxies = append(proxies, req.Proxy.URL)
		}
		return http.StatusOK, response{Status: "ok", Solution: solution{URL: req.URL, Status: http.StatusOK, Response: "<html></html>"}}
	})

	for _, pageURL := range []string{"http://www.example.com/manga", "http://example.org/manga"} {
		if _, err := Get(t.Context(), pageURL, time.Second); err != nil {
			t.Fatalf("%s: %s", pageURL, err)
		}
	}
	if len(proxies) != 2 || proxies[0] != "http://gluetun:8888" || proxies[1] != "" {
		t.Fatalf("proxies sent to FlareSolverr = %q, want the proxy only for example.com", proxies)
	}
}

func TestGetSendsTheProxyCredentialsApart(t *testing.T) {
	util.SetAllowPrivateURLs(true)
	defer util.SetAllowPrivateURLs(false)
	if err := util.SetProxy("http://mantium:s3cr%40t@gluetun:8888", []string{"example.com"}); err != nil {
		t.Fatal(err)
	}
	defer util.SetProxy("", nil)

	var got *requestProxy
	fakeFlareSolverr(t, func(req request) (int, response) {
		got = req.Proxy
		return http.StatusOK, response{Status: "ok", Solution: solution{URL: req.URL, Status: http.StatusOK, Response: "<html></html>"}}
	})

	if _, err := Get(t.Context(), "http://example.com/manga", time.Second); err != nil {
		t.Fatal(err)
	}
	if got == nil || got.URL != "http://gluetun:8888" || got.Username != "mantium" || got.Password != "s3cr@t" {
		t.Fatalf("proxy sent to FlareSolverr = %+v, want the URL without credentials and them apart", got)
	}
}
