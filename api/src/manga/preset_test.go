package manga

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const madaraPage = `<html><body><ul>
<li class="wp-manga-chapter"><a href="https://example.com/manga/x/chapter-31/">Chapter 31 - The end</a></li>
<li class="wp-manga-chapter"><a href="https://example.com/manga/x/chapter-30/">Chapter 30</a></li>
</ul></body></html>`

const themesiaPage = `<html><body><div class="eplister" id="chapterlist"><ul>
<li data-num="12.5"><div class="chbox"><div class="eph-num"><a href="/x-chapter-12-5/"><span class="chapternum">Chapter 12.5</span></a></div></div></li>
<li data-num="12"><div class="chbox"><div class="eph-num"><a href="/x-chapter-12/"><span class="chapternum">Chapter 12</span></a></div></div></li>
</ul></div></body></html>`

const titlesOnlyFeed = `<?xml version="1.0"?><rss><channel>
<item><title>Series c.88</title><description>Group</description></item>
<item><title>Series c.87</title><description>Group</description></item>
</channel></rss>`

func TestDetectPresets(t *testing.T) {
	allowPrivateURLs(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/madara":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(madaraPage))
		case "/themesia":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(themesiaPage))
		case "/feed":
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(titlesOnlyFeed))
		}
	}))
	defer server.Close()

	presets := append([]*Preset{
		{
			Name:         "Madara",
			NameSelector: &HTMLSelector{Selector: "css:li.wp-manga-chapter > a", Regex: chapterNumberRegex, GetFirst: true},
			URLSelector:  &HTMLSelector{Selector: "css:li.wp-manga-chapter > a", Attribute: "href", GetFirst: true},
		},
		{
			Name:         "MangaReader (Themesia)",
			NameSelector: &HTMLSelector{Selector: "css:#chapterlist li", Attribute: "data-num", GetFirst: true},
			URLSelector:  &HTMLSelector{Selector: "css:#chapterlist li .eph-num a", Attribute: "href", GetFirst: true},
		},
	}, BuiltinPresets...)

	tests := []struct {
		path        string
		wantPreset  string
		wantChapter string
		wantURL     string
	}{
		{"/madara", "Madara", "31", "https://example.com/manga/x/chapter-31/"},
		{"/themesia", "MangaReader (Themesia)", "12.5", server.URL + "/x-chapter-12-5/"},
		{"/feed", "RSS feed (titles only)", "88", ""},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			detections, err := DetectPresets(server.URL+test.path, presets)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			working := []*PresetDetection{}
			for _, d := range detections {
				if d.Chapter != nil {
					working = append(working, d)
				} else if d.UserMessage == "" {
					t.Errorf("preset %s failed without a message", d.Preset.Name)
				}
			}
			if len(working) != 1 || working[0].Preset.Name != test.wantPreset {
				names := []string{}
				for _, d := range working {
					names = append(names, d.Preset.Name)
				}
				t.Fatalf("want only %s to work, got %v", test.wantPreset, names)
			}
			if working[0].Chapter.Chapter != test.wantChapter || working[0].Chapter.URL != test.wantURL {
				t.Fatalf("got chapter %q URL %q, want %q %q", working[0].Chapter.Chapter, working[0].Chapter.URL, test.wantChapter, test.wantURL)
			}
		})
	}
}

func TestPresetsForURL(t *testing.T) {
	domains, err := normalizeDomains([]string{" https://www.Example.com/manga ", "", "other.org"})
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 2 || domains[0] != "example.com" || domains[1] != "other.org" {
		t.Fatalf("unexpected normalized domains: %v", domains)
	}

	presets := []*Preset{{Name: "mine", Domains: domains}, {Name: "builtin"}}
	for url, want := range map[string]int{
		"https://example.com/manga/x":     1,
		"https://www.example.com/manga/x": 1,
		"https://read.example.com/x":      1,
		"https://notexample.com/x":        0,
		"not a url":                       0,
	} {
		if got := PresetsForURL(url, presets); len(got) != want {
			t.Errorf("%s: got %d presets, want %d", url, len(got), want)
		}
	}
}

func TestNameOnlyChapterURL(t *testing.T) {
	allowPrivateURLs(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(titlesOnlyFeed))
	}))
	defer server.Close()

	nameSelector := &HTMLSelector{Selector: "xpath://item/title", Regex: chapterNumberRegex, GetFirst: true}
	get := func(path string) *Chapter {
		t.Helper()
		chapter, err := GetCustomMangaLastReleasedChapter(CustomMangaScraping{URL: server.URL + path, NameSelector: nameSelector})
		if err != nil {
			t.Fatal(err)
		}
		return chapter
	}

	first, again, other := get("/a"), get("/a"), get("/b")
	if !strings.HasPrefix(first.URL, CustomMangaURLPrefix+"/") || !strings.HasSuffix(first.URL, "/88") {
		t.Fatalf("unexpected chapter URL %q", first.URL)
	}
	if first.URL != again.URL {
		t.Fatalf("scraping the same page twice gave different chapter URLs: %q and %q", first.URL, again.URL)
	}
	if first.URL == other.URL {
		t.Fatalf("two manga pages with the same chapter got the same chapter URL %q", first.URL)
	}
}
