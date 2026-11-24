//go:build db

package manga

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/internal/testutil"
	"github.com/diogovalentte/mantium/api/src/db"
)

func TestCustomMangaUpdateKeepsReleaseDateOfSameChapter(t *testing.T) {
	testutil.Database(t)
	allowPrivateURLs(t)

	page := func(chapter string) []byte {
		return []byte(`<html><body><a class="chapter" href="/manga/chapter-` + chapter + `">Chapter ` + chapter + `</a></body></html>`)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/new":
			w.Write(page("13"))
		default:
			w.Write(page("12"))
		}
	}))
	defer server.Close()

	nameSelector := &HTMLSelector{Selector: "css:a.chapter", Regex: `Chapter (\d+)`, GetFirst: true}
	urlSelector := &HTMLSelector{Selector: "css:a.chapter", Attribute: "href", GetFirst: true}
	releasedAt := time.Date(2020, 1, 2, 3, 4, 5, 0, time.Local)

	m := &Manga{
		Source:                          CustomMangaSource,
		URL:                             server.URL + "/manga",
		Name:                            "Custom",
		Status:                          1,
		CoverImg:                        []byte{},
		LastReleasedChapterNameSelector: nameSelector,
		LastReleasedChapterURLSelector:  urlSelector,
		LastReleasedChapter: &Chapter{
			URL:       server.URL + "/manga/chapter-12",
			Name:      "Chapter 12",
			Chapter:   "12",
			UpdatedAt: releasedAt,
			Type:      1,
		},
	}
	conn, err := db.OpenConn()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := conn.Begin()
	if err != nil {
		t.Fatal(err)
	}
	m.ID, err = insertMangaIntoDB(m, tx)
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	reload := func() *Manga {
		t.Helper()
		got, err := GetMangaDB(m.ID, "")
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	assertChapter := func(step, chapter string, sameDate bool) {
		t.Helper()
		got := reload().LastReleasedChapter
		if got == nil || got.Chapter != chapter {
			t.Fatalf("%s: got chapter %+v, want %s", step, got, chapter)
		}
		// The column has no time zone, so compare the wall clock
		const layout = "2006-01-02 15:04:05"
		if sameDate != (got.UpdatedAt.Format(layout) == releasedAt.Format(layout)) {
			t.Fatalf("%s: got release date %s, want the old one kept: %t", step, got.UpdatedAt, sameDate)
		}
	}

	// Each update scrapes the page again and gets the same chapter
	if err := reload().UpdateCustomMangaInDB(CustomMangaUpdate{
		Name: "Custom renamed", URL: server.URL + "/manga", CheckURL: server.URL + "/other",
		NameSelector: nameSelector, URLSelector: urlSelector,
	}); err != nil {
		t.Fatal(err)
	}
	assertChapter("UpdateCustomMangaInDB", "12", true)

	if err := reload().UpdateLastReleasedChapterSelectorsInDB(&HTMLSelector{Selector: "css:a", Regex: `Chapter (\d+)`, GetFirst: true}, urlSelector, false, false); err != nil {
		t.Fatal(err)
	}
	assertChapter("UpdateLastReleasedChapterSelectorsInDB", "12", true)

	if err := reload().UpdateCustomMangaURLInDB(server.URL + "/manga2"); err != nil {
		t.Fatal(err)
	}
	assertChapter("UpdateCustomMangaURLInDB", "12", true)

	// A new chapter gets a new release date
	if err := reload().UpdateCustomMangaInDB(CustomMangaUpdate{
		Name: "Custom renamed", URL: server.URL + "/manga2", CheckURL: server.URL + "/new",
		NameSelector: nameSelector, URLSelector: urlSelector,
	}); err != nil {
		t.Fatal(err)
	}
	assertChapter("new chapter", "13", false)
}
