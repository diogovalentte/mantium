package util

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// SiteURL is the base URL of a source whose domain changes over time.
// It is read by concurrent update workers and swapped when a new domain is found.
type SiteURL struct {
	mu  sync.RWMutex
	url string
}

// NewSiteURL returns a SiteURL starting at the given base URL.
func NewSiteURL(baseURL string) *SiteURL {
	return &SiteURL{url: baseURL}
}

// Get returns the current base URL, like "https://klmanga.zone".
func (s *SiteURL) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.url
}

// Set replaces the current base URL.
func (s *SiteURL) Set(baseURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.url = baseURL
}

// NormalizeBaseURL reduces a URL to its scheme and host, like "https://klmanga.zone".
func NormalizeBaseURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("'%s' is not an http(s) URL with a host", rawURL)
	}

	return parsed.Scheme + "://" + parsed.Host, nil
}

// FollowSiteRedirects requests the root of baseURL and returns the base URL it
// ends up at after the redirects, which is how sources announce a new domain.
func FollowSiteRedirects(baseURL string) (string, error) {
	contextError := "error following the redirects of '%s'"

	client := &http.Client{Timeout: ExternalRequestTimeout}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/", nil)
	if err != nil {
		return "", AddErrorContext(fmt.Sprintf(contextError, baseURL), err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", AddErrorContext(fmt.Sprintf(contextError, baseURL), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", AddErrorContext(fmt.Sprintf(contextError, baseURL), fmt.Errorf("non-200 status code -> (%d)", resp.StatusCode))
	}

	return NormalizeBaseURL(resp.Request.URL.String())
}
