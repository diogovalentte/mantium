package rawkuma

import (
	"path"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources/models"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL, _ string) (*manga.Manga, error) {
	s.resetCollector()

	errorContext := "error while getting manga metadata"

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "rawkuma.com"
	mangaReturn.URL = mangaURL

	var sharedErr error

	// manga name
	s.c.OnHTML("h1.entry-title", func(e *colly.HTMLElement) {
		name := e.Text
		mangaReturn.Name = strings.TrimSuffix(name, " Raw")
	})

	// manga cover
	s.c.OnHTML("div.thumb > img", func(e *colly.HTMLElement) {
		coverURL := e.Attr("src")

		coverImg, resized, err := util.GetImageFromURL(coverURL, 3, 1*time.Second)
		if err == nil {
			mangaReturn.CoverImgURL = coverURL
			mangaReturn.CoverImgResized = resized
			mangaReturn.CoverImg = coverImg
		}
	})

	// last released chapter
	s.c.OnHTML("ul.clstyle > li:first-child", func(e *colly.HTMLElement) {
		chapter := e.Attr("data-num")
		chapterName := e.DOM.Find("div > div > a > span.chapternum").Text()
		chapterURL, exists := e.DOM.Find("div > div > a").Attr("href")
		if !exists {
			sharedErr = errordefs.ErrChapterURLNotFound
			return
		}

		chapterDate := e.DOM.Find("div > div > a > span.chapterdate").Text()
		releaseTime, err := time.Parse("January 2, 2006", chapterDate)
		if err != nil {
			sharedErr = err
			return
		}

		mangaReturn.LastReleasedChapter = &manga.Chapter{
			URL:       chapterURL,
			Chapter:   chapter,
			Name:      chapterName,
			Type:      1,
			UpdatedAt: releaseTime,
		}
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
		}
		return nil, util.AddErrorContext(errorContext, util.AddErrorContext("error while visiting manga URL", err))
	}
	if sharedErr != nil {
		return nil, util.AddErrorContext(errorContext, sharedErr)
	}

	return mangaReturn, nil
}

func (s *Source) Search(term string, limit int) ([]*models.MangaSearchResult, error) {
	s.resetCollector()

	errorContext := "error while searching manga"

	mangaSearchResults := []*models.MangaSearchResult{}
	s.c.OnHTML("div.listupd > div > div", func(e *colly.HTMLElement) {
		mangaSearchResult := &models.MangaSearchResult{}
		mangaSearchResult.Source = "rawkuma.com"
		mangaSearchResult.URL = e.DOM.Find("a").AttrOr("href", "")
		mangaSearchResult.Description = e.DOM.Find("a > div.limit > span.type").Text()
		mangaSearchResult.Name = strings.TrimSpace(e.DOM.Find("a > div.bigor > div.tt").Text())
		mangaSearchResult.CoverURL = e.DOM.Find("a > div.limit > img").AttrOr("src", "")
		if mangaSearchResult.CoverURL == "" {
			mangaSearchResult.CoverURL = models.DefaultCoverImgURL
		}

		mangaSearchResult.LastChapter = strings.TrimPrefix(e.DOM.Find("a > div.bigor > div.adds > div.epxs").Text(), "Chapter ")
		baseURL := path.Base(strings.TrimSuffix(mangaSearchResult.URL, "/"))
		if mangaSearchResult.LastChapter != "" {
			mangaSearchResult.LastChapterURL = baseSiteURL + "/" + baseURL + "-chapter-" + mangaSearchResult.LastChapter
		}

		mangaSearchResults = append(mangaSearchResults, mangaSearchResult)
	})

	term = strings.ReplaceAll(term, " ", "+")
	mangaURL := baseSiteURL + "/?s=" + term
	err := s.c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
		}
		return nil, util.AddErrorContext(errorContext, util.AddErrorContext("error while visiting manga URL", err))
	}

	return mangaSearchResults, nil
}
