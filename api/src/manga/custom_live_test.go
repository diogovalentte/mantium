//go:build live

package manga

import "testing"

func TestCustomMangaLastReleasedChapter(t *testing.T) {
	customMangaNoBrowserHTML := &Manga{
		Source: CustomMangaSource,
		URL:    "https://klmanga.today/manga-raw/%E3%83%9D%E3%83%B3%E3%82%B3%E3%83%84%E9%A2%A8%E7%B4%80%E5%A7%94%E5%93%A1%E3%81%A8%E3%82%B9%E3%82%AB%E3%83%BC%E3%83%88%E4%B8%88%E3%81%8C%E4%B8%8D%E9%81%A9%E5%88%87%E3%81%AAjk%E3%81%AE%E8%A9%B1-raw/",
		LastReleasedChapterNameSelector: &HTMLSelector{
			Selector: "css:div.chapter-box > h4:first-child > a span:first-child",
			Regex:    `【第(\d+(\.\d+)?)話】`,
		},
		LastReleasedChapterURLSelector: &HTMLSelector{
			Selector:  "css:div.chapter-box > h4:first-child > a",
			Attribute: "href",
		},
	}
	customMangaNoBrowserXML := &Manga{
		Source: CustomMangaSource,
		URL:    "https://tonarinoyj.jp/atom/series/3270296674379186980",
		LastReleasedChapterNameSelector: &HTMLSelector{
			GetFirst: true,
			Selector: "xpath://feed/entry/title",
			Regex:    `第(\d+(\.\d+)?)話`,
		},
		LastReleasedChapterURLSelector: &HTMLSelector{
			GetFirst:  true,
			Selector:  "xpath://feed/entry/link",
			Attribute: "href",
		},
	}
	customMangaBrowser := &Manga{
		Source: CustomMangaSource,
		URL:    "https://mangaplus.shueisha.co.jp/titles/100291",
		LastReleasedChapterNameSelector: &HTMLSelector{
			Selector: "xpath://p[contains(@class, 'ChapterListItem-module_title')]",
			Regex:    `Chapter (.+)`,
		},
		LastReleasedChapterSelectorUseBrowser: true,
	}

	t.Run("Should get custom manga last released chapter without browser", func(t *testing.T) {
		chapter, err := GetCustomMangaLastReleasedChapter(customMangaNoBrowserHTML.URL, customMangaNoBrowserHTML.LastReleasedChapterNameSelector, customMangaNoBrowserHTML.LastReleasedChapterURLSelector, customMangaNoBrowserHTML.LastReleasedChapterSelectorUseBrowser)
		if err != nil {
			t.Fatalf("Error getting custom manga last released chapter (without browser): %v", err)
		}
		if chapter.Name == "" {
			t.Fatal("Expected chapter name to be not empty")
		}
		if chapter.URL == "" {
			t.Fatal("Expected chapter URL to be not empty")
		}
	})
	t.Run("Should get custom manga last released chapter without browser using XML path", func(t *testing.T) {
		chapter, err := GetCustomMangaLastReleasedChapter(customMangaNoBrowserXML.URL, customMangaNoBrowserXML.LastReleasedChapterNameSelector, customMangaNoBrowserXML.LastReleasedChapterURLSelector, customMangaNoBrowserXML.LastReleasedChapterSelectorUseBrowser)
		if err != nil {
			t.Fatalf("Error getting custom manga last released chapter (without browser): %v", err)
		}
		if chapter.Name == "" {
			t.Fatal("Expected chapter name to be not empty")
		}
		if chapter.URL == "" {
			t.Fatal("Expected chapter URL to be not empty")
		}
	})
	t.Run("Should get custom manga last released chapter with browser", func(t *testing.T) {
		chapter, err := GetCustomMangaLastReleasedChapter(customMangaBrowser.URL, customMangaBrowser.LastReleasedChapterNameSelector, customMangaBrowser.LastReleasedChapterURLSelector, customMangaBrowser.LastReleasedChapterSelectorUseBrowser)
		if err != nil {
			t.Fatalf("Error getting custom manga last released chapter (with browser): %v", err)
		}
		if chapter.Name == "" {
			t.Fatal("Expected chapter name to be not empty")
		}
	})
}
