//go:build db

package routes_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"testing"

	"github.com/diogovalentte/mantium/api/internal/testutil"
	"github.com/diogovalentte/mantium/api/src/manga"
)

func TestCustomMultiMangaLifecycle(t *testing.T) {
	testutil.Database(t)
	var cover bytes.Buffer
	if err := png.Encode(&cover, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"name":"Fixture Story","status":1,"cover_img":%q}`, base64.StdEncoding.EncodeToString(cover.Bytes()))
	request(t, "POST", "/v1/multimanga", body, 200)
	w := request(t, "GET", "/v1/multimangas", "", 200)
	var list struct {
		MultiMangas []*manga.MultiManga `json:"multimangas"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.MultiMangas) != 1 {
		t.Fatalf("expected one multimanga: %s", w.Body.String())
	}
	id := list.MultiMangas[0].ID
	path := fmt.Sprintf("/v1/multimanga?id=%d", id)
	get := func() *manga.MultiManga {
		t.Helper()
		w := request(t, "GET", path, "", 200)
		var got struct {
			MultiManga *manga.MultiManga `json:"multimanga"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.MultiManga == nil || got.MultiManga.CurrentManga == nil {
			t.Fatalf("missing manga: %s", w.Body.String())
		}
		return got.MultiManga
	}
	mm := get()
	if mm.CurrentManga.Name != "Fixture Story" || mm.CurrentManga.Source != manga.CustomMangaSource {
		t.Fatalf("unexpected manga: %+v", mm)
	}
	request(t, "PATCH", fmt.Sprintf("/v1/custom_manga/name?id=%d&name=Renamed", mm.CurrentManga.ID), `{"name":"Renamed"}`, 200)
	request(t, "PATCH", fmt.Sprintf("/v1/multimanga/status?id=%d", id), `{"status":4}`, 200)
	mm = get()
	if mm.Status != 4 || mm.CurrentManga.Name != "Renamed" {
		t.Fatalf("updates not persisted: %+v", mm)
	}
	request(t, "PATCH", fmt.Sprintf("/v1/multimanga/last_read_chapter?id=%d", id), `{"chapter":"12.5"}`, 200)
	mm = get()
	if mm.LastReadChapter == nil || mm.LastReadChapter.Chapter != "12.5" {
		t.Fatalf("chapter not persisted: %+v", mm)
	}
	request(t, "GET", "/v1/mangas/stats", "", 200)
	request(t, "DELETE", path, "", 200)
	w = request(t, "GET", "/v1/multimangas", "", 200)
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.MultiMangas) != 0 {
		t.Fatal("deleted multimanga still listed")
	}
}
