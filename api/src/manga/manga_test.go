package manga

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/testutil"
)

func TestModelStringsAndStatus(t *testing.T) {
	m := &Manga{Name: "Fixture Story"}
	mm := &MultiManga{CurrentManga: m, Mangas: []*Manga{m}, Status: 1}
	if m.String() == "" || mm.String() == "" {
		t.Fatal("empty model string")
	}
	for status := Status(-1); status <= 6; status++ {
		err := ValidateStatus(status)
		if (err == nil) != (status >= 1 && status <= 5) {
			t.Errorf("status %d: %v", status, err)
		}
	}
}

func TestCustomChapterSelectors(t *testing.T) {
	server := testutil.Fixtures(t, map[string]string{"/manga": "custom.html"})
	for _, selector := range []string{"css:a.chapter", "xpath://a[@class='chapter']"} {
		t.Run(selector, func(t *testing.T) {
			c, err := GetCustomMangaLastReleasedChapter(server.URL+"/manga", &HTMLSelector{Selector: selector, Regex: `第(\d+(\.\d+)?)話`, GetFirst: true}, &HTMLSelector{Selector: selector, Attribute: "href", GetFirst: true}, false)
			if err != nil {
				t.Fatal(err)
			}
			if c.Name != "Chapter 12.5" || c.Chapter != "12.5" || c.URL != server.URL+"/chapter-12.5" {
				t.Fatalf("unexpected chapter: %+v", c)
			}
		})
	}
}
