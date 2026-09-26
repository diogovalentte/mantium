package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeBaseURL(t *testing.T) {
	valid := map[string]string{
		"https://klmanga.zone":                "https://klmanga.zone",
		"https://klmanga.zone/":               "https://klmanga.zone",
		" https://klmanga.zone/manga-raw/x/ ": "https://klmanga.zone",
		"http://jmanga.locker:8080/read/?q=a": "http://jmanga.locker:8080",
	}
	for in, expected := range valid {
		got, err := NormalizeBaseURL(in)
		if err != nil {
			t.Errorf("NormalizeBaseURL(%q) returned error: %s", in, err)
			continue
		}
		if got != expected {
			t.Errorf("NormalizeBaseURL(%q) = %q, want %q", in, got, expected)
		}
	}

	for _, in := range []string{"", "klmanga.zone", "ftp://klmanga.zone", "https://"} {
		if got, err := NormalizeBaseURL(in); err == nil {
			t.Errorf("NormalizeBaseURL(%q) = %q, want an error", in, got)
		}
	}
}

func TestFollowSiteRedirects(t *testing.T) {
	current := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer current.Close()

	// Like klmanga.cam: an old domain that drops the path and sends to the
	// new root, through another old domain.
	middle := httptest.NewServer(http.RedirectHandler(current.URL+"/", http.StatusMovedPermanently))
	defer middle.Close()
	old := httptest.NewServer(http.RedirectHandler(middle.URL+"/", http.StatusMovedPermanently))
	defer old.Close()

	got, err := FollowSiteRedirects(old.URL)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got != current.URL {
		t.Fatalf("FollowSiteRedirects(%q) = %q, want %q", old.URL, got, current.URL)
	}

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer down.Close()
	if got, err := FollowSiteRedirects(down.URL); err == nil {
		t.Fatalf("FollowSiteRedirects on a 403 = %q, want an error", got)
	}
}
