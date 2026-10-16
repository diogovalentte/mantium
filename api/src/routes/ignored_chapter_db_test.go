//go:build db

package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/internal/testutil"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

func TestUpdateMultiMangaMetadataIgnoredChapter(t *testing.T) {
	testutil.Database(t)
	util.SetAllowPrivateURLs(true)
	t.Cleanup(func() { util.SetAllowPrivateURLs(false) })

	var chapters = map[string]*atomic.Int32{"/right": {}, "/wrong": {}}
	chapters["/right"].Store(71)
	chapters["/wrong"].Store(72)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := chapters[r.URL.Path].Load()
		fmt.Fprintf(w, `<html><body><a class="chapter" href="%s/%d">Chapter %d</a></body></html>`, r.URL.Path, n, n)
	}))
	defer server.Close()

	newManga := func(path string) *manga.Manga {
		n := chapters[path].Load()
		return &manga.Manga{
			Source: manga.CustomMangaSource, URL: server.URL + path, Name: "Manga A", Status: 1, CoverImg: []byte{},
			LastReleasedChapter:             &manga.Chapter{URL: fmt.Sprintf("%s%s/%d", server.URL, path, n), Name: fmt.Sprintf("Chapter %d", n), Chapter: fmt.Sprint(n), UpdatedAt: time.Now().Truncate(time.Second), Type: 1},
			LastReleasedChapterNameSelector: &manga.HTMLSelector{Selector: "css:a.chapter", Regex: `Chapter (\d+)`},
			LastReleasedChapterURLSelector:  &manga.HTMLSelector{Selector: "css:a.chapter", Attribute: "href"},
		}
	}
	right, wrong := newManga("/right"), newManga("/wrong")
	mm := &manga.MultiManga{Status: 1, CurrentManga: right, Mangas: []*manga.Manga{right}}
	if err := mm.InsertIntoDB(); err != nil {
		t.Fatal(err)
	}
	if err := mm.AddManga(wrong); err != nil {
		t.Fatal(err)
	}

	logger := zerolog.Nop()
	update := func() (*manga.MultiManga, *manga.Manga) {
		t.Helper()
		got, err := manga.GetMultiMangaFromDB(mm.ID)
		if err != nil {
			t.Fatal(err)
		}
		notify, _, _, errs := updateMultiMangaMetadata(got, newBrowserLimiter(1), 1, 0, &logger)
		if len(errs) > 0 {
			t.Fatal(errs)
		}
		got, err = manga.GetMultiMangaFromDB(mm.ID)
		if err != nil {
			t.Fatal(err)
		}
		return got, notify
	}
	findWrong := func(mm *manga.MultiManga) *manga.Manga {
		for _, m := range mm.Mangas {
			if m.URL == wrong.URL {
				return m
			}
		}
		t.Fatal("wrong manga not found")
		return nil
	}

	got, _ := update()
	if err := got.IgnoreMangaLastReleasedChapter(findWrong(got)); err != nil {
		t.Fatal(err)
	}

	got, notify := update()
	if got.CurrentManga.URL != right.URL || notify != nil || findWrong(got).IgnoredChapter == nil {
		t.Fatalf("nothing released: current %s, notify %v, ignored %v", got.CurrentManga.URL, notify, findWrong(got).IgnoredChapter)
	}

	chapters["/right"].Store(72)
	got, notify = update()
	if got.CurrentManga.URL != right.URL || notify == nil || notify.LastReleasedChapter.Chapter != "72" {
		t.Fatalf("right chapter released: current %s, notify %v", got.CurrentManga.URL, notify)
	}
	if findWrong(got).IgnoredChapter == nil {
		t.Fatal("ignored chapter cleared before its source released another chapter")
	}

	chapters["/wrong"].Store(73)
	got, notify = update()
	if findWrong(got).IgnoredChapter != nil {
		t.Fatal("ignored chapter not cleared after its source released another chapter")
	}
	if got.CurrentManga.URL != wrong.URL || notify == nil || notify.LastReleasedChapter.Chapter != "73" {
		t.Fatalf("ignored source released 73: current %s, notify %v", got.CurrentManga.URL, notify)
	}
}
