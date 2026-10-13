package mangaplus

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestFixtureMetadata(t *testing.T) {
	data, err := os.ReadFile("testdata/title.json")
	if err != nil {
		t.Fatal(err)
	}
	var response Response
	if err := protojson.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	body, err := proto.Marshal(&response)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/title_list/allV2" && (r.URL.Path != "/title_detailV3" || r.URL.Query().Get("title_id") != "123") {
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	old := baseAPIURL
	baseAPIURL = server.URL
	t.Cleanup(func() { baseAPIURL = old })
	s := &Source{}
	target := "https://mangaplus.shueisha.co.jp/titles/123"
	m, err := s.GetMangaMetadata(target, "")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Fixture Story" || m.LastReleasedChapter == nil || m.LastReleasedChapter.Chapter != "12.5" {
		t.Fatalf("unexpected manga: %+v", m)
	}
	chapters, err := s.GetChaptersMetadata(target, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 1 || chapters[0].Name != "The Return" || chapters[0].UpdatedAt.Unix() != 1704164645 {
		t.Fatalf("unexpected chapters: %+v", chapters)
	}
	last, err := s.GetLastChapterMetadata(target, "")
	if err != nil {
		t.Fatal(err)
	}
	if last.Chapter != "12.5" {
		t.Fatalf("unexpected last chapter: %+v", last)
	}
	chapter, err := s.GetChapterMetadata(target, "", "12.5", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if chapter.URL != "https://mangaplus.shueisha.co.jp/viewer/456" {
		t.Fatalf("unexpected chapter: %+v", chapter)
	}
	if _, err := s.GetChapterMetadata(target, "", "999", "", ""); err == nil {
		t.Fatal("expected missing chapter error")
	}
	results, err := s.Search("fixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "Fixture Story" || results[0].URL != target {
		t.Fatalf("unexpected search: %+v", results)
	}
	results, err = s.Search("missing", 1)
	if err != nil || len(results) != 0 {
		t.Fatalf("unmatched search: %+v, %v", results, err)
	}
}
