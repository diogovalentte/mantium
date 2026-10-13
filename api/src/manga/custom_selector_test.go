package manga

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diogovalentte/mantium/api/src/errordefs"
)

const customMangaPage = `<html><body>
<div class="chapters">
  <a class="chapter" href="/manga/chapter-12">Chapter 12</a>
  <a class="chapter" href="/manga/chapter-11">Chapter 11</a>
  <span class="empty"></span>
</div>
</body></html>`

func TestGetSelectorFromPageUsingHTTPRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(customMangaPage))
	}))
	defer server.Close()

	tests := []struct {
		name        string
		path        string
		selector    HTMLSelector
		want        string
		userMessage string
	}{
		{name: "css with regex", selector: HTMLSelector{Selector: "css:a.chapter", Regex: `Chapter (\d+)`, GetFirst: true}, want: "12"},
		{name: "xpath attribute", selector: HTMLSelector{Selector: "xpath://a[@class='chapter']", Attribute: "href"}, want: "/manga/chapter-11"},
		{name: "page not found", path: "/missing", selector: HTMLSelector{Selector: "css:a"}, userMessage: "Couldn't open the manga URL: Not Found"},
		{name: "no prefix", selector: HTMLSelector{Selector: "a.chapter"}, userMessage: "Chapter name selector must start with 'css:' or 'xpath:'"},
		{name: "empty after prefix", selector: HTMLSelector{Selector: "css: "}, userMessage: "Chapter name selector is empty after 'css:'"},
		{name: "no match", selector: HTMLSelector{Selector: "css:div.nothing"}, userMessage: "Chapter name selector didn't match any element on the page"},
		{name: "missing attribute", selector: HTMLSelector{Selector: "css:a.chapter", Attribute: "title"}, userMessage: "Chapter name selector matched, but the element has no 'title' attribute"},
		{name: "empty text", selector: HTMLSelector{Selector: "css:span.empty"}, userMessage: "Chapter name selector matched, but the element's text is empty"},
		{name: "invalid regex", selector: HTMLSelector{Selector: "css:a.chapter", Regex: `Chapter (\d+`}, userMessage: "Chapter name regex is invalid: error parsing regexp: missing closing ): `Chapter (\\d+`"},
		{name: "regex without group", selector: HTMLSelector{Selector: "css:a.chapter", Regex: `Chapter \d+`}, userMessage: `Chapter name regex has no capture group: wrap the chapter in ( ), e.g. Chapter (\d+)`},
		{name: "regex no match", selector: HTMLSelector{Selector: "css:a.chapter", Regex: `Episode (\d+)`}, userMessage: `Chapter name regex didn't match the text "Chapter 11"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := getSelectorFromPageUsingHTTPRequest(server.URL+test.path, &test.selector, chapterNameField)
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
			if !errors.As(err, &userErr) {
				t.Fatalf("expected a UserError, got: %v", err)
			}
			if userErr.Message != test.userMessage {
				t.Fatalf("got message %q, want %q", userErr.Message, test.userMessage)
			}
		})
	}
}
