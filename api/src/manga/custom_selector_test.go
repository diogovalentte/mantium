package manga

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/util"
)

// allowPrivateURLs allows the test to reach httptest servers, which listen on 127.0.0.1.
func allowPrivateURLs(t *testing.T) {
	t.Helper()
	util.SetAllowPrivateURLs(true)
	t.Cleanup(func() { util.SetAllowPrivateURLs(false) })
}

const customMangaPage = `<html><body>
<div class="chapters">
  <a class="chapter" href="/manga/chapter-12">Chapter 12</a>
  <a class="chapter" href="chapter-11">Chapter 11</a>
  <span class="empty"></span>
</div>
</body></html>`

const customMangaPageWithBase = `<html><head><base href="https://cdn.example.com/reader/"></head><body>
<a class="chapter" href="chapter-5">Chapter 5</a>
<a class="bad" href="javascript:void(0)">Chapter 6</a>
</body></html>`

const customMangaFeed = `<?xml version="1.0"?>
<rss><channel>
  <item><title>Chapter 21</title><link>https://example.com/manga/21</link></item>
  <item><title>Chapter 20</title><link>https://example.com/manga/20</link></item>
</channel></rss>`

func newCustomMangaTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/missing":
			http.NotFound(w, r)
		case "/redirect":
			http.Redirect(w, r, "/title/one-piece/", http.StatusFound)
		case "/with-base":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(customMangaPageWithBase))
		case "/feed.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(customMangaFeed))
		default:
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(customMangaPage))
		}
	}))
}

func TestGetCustomMangaLastReleasedChapter(t *testing.T) {
	allowPrivateURLs(t)
	server := newCustomMangaTestServer()
	defer server.Close()

	tests := []struct {
		name         string
		path         string
		nameSelector *HTMLSelector
		urlSelector  *HTMLSelector
		wantChapter  string
		wantURL      string
	}{
		{
			name:         "css with regex and get first",
			nameSelector: &HTMLSelector{Selector: "css:a.chapter", Regex: `Chapter (\d+)`, GetFirst: true},
			urlSelector:  &HTMLSelector{Selector: "css:a.chapter", Attribute: "href", GetFirst: true},
			wantChapter:  "12",
			wantURL:      server.URL + "/manga/chapter-12",
		},
		{
			name:         "last match by default, relative URL resolved against redirected page URL",
			path:         "/redirect",
			nameSelector: &HTMLSelector{Selector: "xpath://a[@class='chapter']", Regex: `Chapter (\d+)`},
			urlSelector:  &HTMLSelector{Selector: "xpath://a[@class='chapter']", Attribute: "href"},
			wantChapter:  "11",
			wantURL:      server.URL + "/title/one-piece/chapter-11",
		},
		{
			name:        "URL only, resolved against <base>",
			path:        "/with-base",
			urlSelector: &HTMLSelector{Selector: "css:a.chapter", Attribute: "href"},
			wantChapter: UnknownChapter,
			wantURL:     "https://cdn.example.com/reader/chapter-5",
		},
		{
			name:         "xpath on XML feed",
			path:         "/feed.xml",
			nameSelector: &HTMLSelector{Selector: "xpath://item/title", Regex: `Chapter (\d+)`, GetFirst: true},
			urlSelector:  &HTMLSelector{Selector: "xpath://item/link", GetFirst: true},
			wantChapter:  "21",
			wantURL:      "https://example.com/manga/21",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chapter, err := GetCustomMangaLastReleasedChapter(server.URL+test.path, test.nameSelector, test.urlSelector, false)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if chapter.Chapter != test.wantChapter {
				t.Errorf("got chapter %q, want %q", chapter.Chapter, test.wantChapter)
			}
			if chapter.URL != test.wantURL {
				t.Errorf("got URL %q, want %q", chapter.URL, test.wantURL)
			}
		})
	}
}

func TestGetCustomMangaLastReleasedChapterUserErrors(t *testing.T) {
	allowPrivateURLs(t)
	server := newCustomMangaTestServer()
	defer server.Close()

	tests := []struct {
		name         string
		path         string
		nameSelector *HTMLSelector
		urlSelector  *HTMLSelector
		userMessage  string
	}{
		{name: "page not found", path: "/missing", nameSelector: &HTMLSelector{Selector: "css:a"}, userMessage: "Couldn't open the manga URL: Not Found"},
		{name: "no prefix", nameSelector: &HTMLSelector{Selector: "a.chapter"}, userMessage: "Chapter name selector must start with 'css:' or 'xpath:'"},
		{name: "empty after prefix", nameSelector: &HTMLSelector{Selector: "css: "}, userMessage: "Chapter name selector is empty after 'css:'"},
		{name: "invalid xpath", urlSelector: &HTMLSelector{Selector: "xpath://a["}, userMessage: "Chapter URL selector is not a valid XPath: expression must evaluate to a node-set"},
		{name: "no match", nameSelector: &HTMLSelector{Selector: "css:div.nothing"}, userMessage: "Chapter name selector didn't match any element on the page"},
		{name: "missing attribute", urlSelector: &HTMLSelector{Selector: "css:a.chapter", Attribute: "title"}, userMessage: "Chapter URL selector matched, but the element has no 'title' attribute"},
		{name: "empty text", nameSelector: &HTMLSelector{Selector: "css:span.empty"}, userMessage: "Chapter name selector matched, but the element's text is empty"},
		{name: "invalid regex", nameSelector: &HTMLSelector{Selector: "css:a.chapter", Regex: `Chapter (\d+`}, userMessage: "Chapter name regex is invalid: error parsing regexp: missing closing ): `Chapter (\\d+`"},
		{name: "regex without group", nameSelector: &HTMLSelector{Selector: "css:a.chapter", Regex: `Chapter \d+`}, userMessage: `Chapter name regex has no capture group: wrap the chapter in ( ), e.g. Chapter (\d+)`},
		{name: "regex no match", nameSelector: &HTMLSelector{Selector: "css:a.chapter", Regex: `Episode (\d+)`}, userMessage: `Chapter name regex didn't match the text "Chapter 11"`},
		{name: "not an http URL", path: "/with-base", urlSelector: &HTMLSelector{Selector: "css:a.bad", Attribute: "href"}, userMessage: `Chapter URL selector returned "javascript:void(0)", which isn't an http(s) URL`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := GetCustomMangaLastReleasedChapter(server.URL+test.path, test.nameSelector, test.urlSelector, false)
			var userErr *errordefs.UserError
			if !errors.As(err, &userErr) {
				t.Fatalf("expected a UserError, got: %v", err)
			}
			if userErr.Message != test.userMessage {
				t.Fatalf("got message %q, want %q", userErr.Message, test.userMessage)
			}
		})
	}
}

func TestGetCustomMangaLastReleasedChapterBlocksPrivateAddresses(t *testing.T) {
	server := newCustomMangaTestServer()
	defer server.Close()
	util.SetAllowPrivateURLs(false)

	for _, useBrowser := range []bool{false, true} {
		_, err := GetCustomMangaLastReleasedChapter(server.URL, &HTMLSelector{Selector: "css:a.chapter"}, nil, useBrowser)
		var userErr *errordefs.UserError
		if !errors.As(err, &userErr) {
			t.Fatalf("browser %v: expected a UserError, got: %v", useBrowser, err)
		}
		want := "Blocked access to a private address (127.0.0.1). Set ALLOW_PRIVATE_URLS=true in the API to allow it"
		if userErr.Message != want {
			t.Fatalf("browser %v: got message %q, want %q", useBrowser, userErr.Message, want)
		}
	}
}

func TestProcessSelectionRegexGroups(t *testing.T) {
	tests := []struct {
		regex       string
		text        string
		want        string
		userMessage string
	}{
		{regex: `Chapter (\d+)`, text: "Chapter 12 - The end", want: "12"},
		{regex: `Chapter (\d+)(\.\d+)?`, text: "Chapter 12.5", want: "12.5"},
		{regex: `Chapter (\d+)(\.\d+)?`, text: "Chapter 12", want: "12"},
		{regex: `Ch\.(\d+)|Chapter (\d+)`, text: "Chapter 12", want: "12"},
		{regex: `Ch\.(\d+)|Chapter (\d+)`, text: "Ch.7", want: "7"},
		{regex: `Vol\.(\d+) Ch\.(\d+)`, text: "Vol.3 Ch.25", want: "325"},
		{regex: `(?:Chapter) (\d+)`, text: "Chapter 4", want: "4"},
		{regex: `Chapter (\d*)`, text: "Chapter ?", userMessage: `Chapter name regex matched "Chapter", but its groups are empty`},
	}

	for _, test := range tests {
		t.Run(test.regex+" on "+test.text, func(t *testing.T) {
			got, err := processSelection(test.text, true, &HTMLSelector{Selector: "css:a", Regex: test.regex}, chapterNameField)
			if test.userMessage == "" {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if got != test.want {
					t.Fatalf("got %q, want %q", got, test.want)
				}
				return
			}
			var userErr *errordefs.UserError
			if !errors.As(err, &userErr) || userErr.Message != test.userMessage {
				t.Fatalf("got error %v, want user message %q", err, test.userMessage)
			}
		})
	}
}
