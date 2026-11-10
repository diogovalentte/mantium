package mangahub

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources/models"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL, _ string) (*manga.Manga, error) {
	errorContext := "error while getting manga metadata"

	formattedURL, err := GetFormattedMangaURL(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	page, err := getMangaPage(formattedURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	mangaReturn := &manga.Manga{
		Source: "mangahub",
		URL:    formattedURL,
		Name:   page.name,
	}

	if page.coverURL != "" {
		coverImg, resized, err := util.GetImageFromURL(page.coverURL, 3, 1*time.Second)
		if err == nil {
			mangaReturn.CoverImgURL = page.coverURL
			mangaReturn.CoverImgResized = resized
			mangaReturn.CoverImg = coverImg
		}
	}

	if len(page.chapters) > 0 {
		lastReleasedChapter := *page.chapters[0]
		lastReleasedChapter.Type = 1
		mangaReturn.LastReleasedChapter = &lastReleasedChapter
	}

	return mangaReturn, nil
}

// Search searches the site for mangas, in the order the site returns them
func (s *Source) Search(term string, limit int) ([]*models.MangaSearchResult, error) {
	errorContext := "error while searching manga"

	c := newCollector()
	results := []*models.MangaSearchResult{}

	c.OnHTML("div.media-manga", func(e *colly.HTMLElement) {
		if len(results) >= limit {
			return
		}
		link := e.DOM.Find("h4.media-heading > a").First()
		href, _ := link.Attr("href")
		name := strings.TrimSpace(link.Text())
		if href == "" || name == "" {
			return
		}

		result := &models.MangaSearchResult{
			Source:   "mangahub",
			URL:      e.Request.AbsoluteURL(href),
			Name:     name,
			CoverURL: models.DefaultCoverImgURL,
		}
		if formattedURL, err := GetFormattedMangaURL(result.URL); err == nil {
			result.URL = formattedURL
		}
		if cover := e.ChildAttr("img.manga-thumb", "src"); cover != "" {
			result.CoverURL = e.Request.AbsoluteURL(cover)
		}

		// Like "#112 chapters published (Completed)", with the last chapter in the link
		info := e.DOM.Find("div.media-body > span").First()
		lastChapter := info.Find("a").First()
		result.LastChapter = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lastChapter.Text()), "#"))
		if chapterURL, ok := lastChapter.Attr("href"); ok {
			result.LastChapterURL = e.Request.AbsoluteURL(chapterURL)
		}
		if status := searchStatusRe.FindStringSubmatch(info.Text()); status != nil {
			result.Status = strings.ToLower(status[1])
		}

		results = append(results, result)
	})

	searchURL := fmt.Sprintf("%s/search?q=%s", baseSiteURL, url.QueryEscape(term))
	if err := c.Visit(searchURL); err != nil {
		return nil, util.AddErrorContext(errorContext, visitError(err))
	}

	return results, nil
}

var searchStatusRe = regexp.MustCompile(`\(([^)]+)\)\s*$`)

var mangaSlugRe = regexp.MustCompile(`/manga/([0-9a-zA-Z_-]+)(?:/.*)?$`)

// getMangaSlug returns the slug of a manga given its URL.
// URL should be like: https://mangahub.io/manga/super-no-ura-de-yani-suu-hanashi
func getMangaSlug(mangaURL string) (string, error) {
	matches := mangaSlugRe.FindStringSubmatch(mangaURL)
	if len(matches) < 2 {
		return "", util.AddErrorContext("error while getting manga slug from URL", fmt.Errorf("manga Slug not found in URL"))
	}

	return matches[1], nil
}

func GetFormattedMangaURL(mangaURL string) (string, error) {
	errorContext := "error while getting formatted manga URL '%s'"

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return "", util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	return fmt.Sprintf("%s/manga/%s", baseSiteURL, mangaSlug), nil
}
