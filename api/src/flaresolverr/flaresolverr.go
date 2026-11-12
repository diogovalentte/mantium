// Package flaresolverr gets pages through FlareSolverr (https://github.com/FlareSolverr/FlareSolverr),
// a proxy that opens them in a real browser and solves Cloudflare's challenges.
//
// Transport is an http.RoundTripper, so any HTTP client or colly collector can use it:
//
//	c := colly.NewCollector()
//	c.WithTransport(flaresolverr.Transport(timeout))
//
// FlareSolverr only returns the page HTML after the browser renders it: no response headers
// besides the ones set here, and no binary content like images. Use it for HTML pages.
package flaresolverr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/util"
)

// address is FlareSolverr's base URL, like http://flaresolverr:8191. Set by the config package from FLARESOLVERR_ADDRESS.
var address atomic.Value

// SetAddress sets FlareSolverr's base URL. An empty address disables FlareSolverr.
func SetAddress(addr string) {
	address.Store(strings.TrimSuffix(strings.TrimSpace(addr), "/"))
}

// Address returns FlareSolverr's base URL, or an empty string if it isn't configured.
func Address() string {
	addr, _ := address.Load().(string)
	return addr
}

// Enabled tells if FlareSolverr is configured.
func Enabled() bool {
	return Address() != ""
}

// ErrNotConfigured is returned when FlareSolverr is used without FLARESOLVERR_ADDRESS.
var ErrNotConfigured = &errordefs.UserError{Message: "FlareSolverr isn't configured. Set FLARESOLVERR_ADDRESS in the API to use it"}

// DefaultTimeout is how long FlareSolverr has to get a page and solve its challenge.
const DefaultTimeout = 60 * time.Second

// clientExtraTime is added to the timeout given to FlareSolverr for the request to it,
// so FlareSolverr reports its own timeout instead of the request being cut.
const clientExtraTime = 15 * time.Second

// maxResponseBytes limits FlareSolverr's JSON response, which carries the whole page.
const maxResponseBytes = 50 << 20

type request struct {
	Cmd        string        `json:"cmd"`
	URL        string        `json:"url"`
	MaxTimeout int64         `json:"maxTimeout"`
	Proxy      *requestProxy `json:"proxy,omitempty"`
}

type requestProxy struct {
	URL string `json:"url"`
}

type response struct {
	Status   string   `json:"status"`
	Message  string   `json:"message"`
	Solution solution `json:"solution"`
}

type solution struct {
	URL       string            `json:"url"`
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Response  string            `json:"response"`
	UserAgent string            `json:"userAgent"`
}

// Page is a page got through FlareSolverr.
type Page struct {
	// URL is the page's final URL, after redirects.
	URL string
	// StatusCode is the page's HTTP status code.
	StatusCode int
	// Body is the page's HTML, rendered by FlareSolverr's browser.
	Body []byte
}

// Get gets the page with FlareSolverr. timeout is how long FlareSolverr has to get it.
// The page URL and its final URL must be public addresses, unless private URLs are allowed,
// since FlareSolverr would reach them from inside the network. FlareSolverr's own address can be private.
func Get(ctx context.Context, pageURL string, timeout time.Duration) (*Page, error) {
	contextError := fmt.Sprintf("error getting page '%s' using FlareSolverr", pageURL)

	addr := Address()
	if addr == "" {
		return nil, util.AddErrorContext(contextError, ErrNotConfigured)
	}
	if err := util.CheckPublicURL(ctx, pageURL); err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	fsReq := request{Cmd: "request.get", URL: pageURL, MaxTimeout: timeout.Milliseconds()}
	// FlareSolverr's browser goes through the proxy for the whole page, so it must reach the proxy too.
	if parsed, err := url.Parse(pageURL); err == nil {
		if proxyURL := util.ProxyFor(parsed.Hostname()); proxyURL != nil {
			fsReq.Proxy = &requestProxy{URL: proxyURL.String()}
		}
	}
	body, err := json.Marshal(fsReq)
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr+"/v1", bytes.NewReader(body))
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout + clientExtraTime}
	res, err := client.Do(req)
	if err != nil {
		return nil, util.AddErrorContext(contextError, &errordefs.UserError{Message: fmt.Sprintf("Couldn't reach FlareSolverr at %s", addr), Err: err})
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil {
		return nil, util.AddErrorContext(contextError, util.AddErrorContext("error reading FlareSolverr response", err))
	}
	if len(resBody) > maxResponseBytes {
		return nil, util.AddErrorContext(contextError, fmt.Errorf("FlareSolverr response is larger than %d bytes", maxResponseBytes))
	}

	var fsRes response
	if err := json.Unmarshal(resBody, &fsRes); err != nil {
		return nil, util.AddErrorContext(contextError, util.AddErrorContext(fmt.Sprintf("error decoding FlareSolverr response (status %d)", res.StatusCode), err))
	}
	// FlareSolverr answers 500 with status "error" when it can't get the page, like on a timeout.
	if fsRes.Status != "ok" {
		message := strings.TrimSpace(fsRes.Message)
		if message == "" {
			message = fmt.Sprintf("status '%s', HTTP %d", fsRes.Status, res.StatusCode)
		}
		return nil, util.AddErrorContext(contextError, &errordefs.UserError{Message: fmt.Sprintf("FlareSolverr couldn't get the page: %s", message)})
	}

	page := &Page{
		URL:        fsRes.Solution.URL,
		StatusCode: fsRes.Solution.Status,
		Body:       []byte(fsRes.Solution.Response),
	}
	if page.URL == "" {
		page.URL = pageURL
	}
	if page.StatusCode == 0 {
		page.StatusCode = http.StatusOK
	}
	// A redirect followed by FlareSolverr's browser can't be blocked before it happens,
	// but its result isn't used.
	if page.URL != pageURL {
		if err := util.CheckPublicURL(ctx, page.URL); err != nil {
			return nil, util.AddErrorContext(contextError, err)
		}
	}

	return page, nil
}

// transport is the http.RoundTripper that gets pages through FlareSolverr.
type transport struct {
	timeout time.Duration
}

// Transport returns an http.RoundTripper that sends GET requests through FlareSolverr,
// giving it timeout to get each page. Other methods fail, since FlareSolverr's request.post
// only sends forms. The request headers aren't sent: FlareSolverr's browser uses its own.
//
// The response's Request has the page's final URL, so clients like colly resolve relative URLs
// against it. The client's timeout must be longer than timeout, or the client cuts the request first.
func Transport(timeout time.Duration) http.RoundTripper {
	return &transport{timeout: timeout}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		req.Body.Close()
	}
	if req.Method != http.MethodGet {
		return nil, fmt.Errorf("FlareSolverr transport only supports GET requests, got %s", req.Method)
	}

	page, err := Get(req.Context(), req.URL.String(), t.timeout)
	if err != nil {
		return nil, err
	}

	finalReq := req
	if page.URL != req.URL.String() {
		finalURL, err := req.URL.Parse(page.URL)
		if err != nil {
			return nil, util.AddErrorContext(fmt.Sprintf("error parsing FlareSolverr final URL '%s'", page.URL), err)
		}
		finalReq = req.Clone(req.Context())
		finalReq.URL = finalURL
		finalReq.Host = finalURL.Host
	}

	header := http.Header{}
	header.Set("Content-Type", "text/html; charset=utf-8")

	return &http.Response{
		Status:        fmt.Sprintf("%d %s", page.StatusCode, http.StatusText(page.StatusCode)),
		StatusCode:    page.StatusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(page.Body)),
		ContentLength: int64(len(page.Body)),
		Request:       finalReq,
	}, nil
}
