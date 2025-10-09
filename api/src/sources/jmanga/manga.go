package jmanga

import (
	"net/url"
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
	mangaReturn.Source = "jmanga"
	mangaReturn.URL = mangaURL

	var sharedErr error

	// manga name
	s.c.OnHTML("h2.manga-name", func(e *colly.HTMLElement) {
		mangaReturn.Name = e.Text
	})

	// manga cover
	s.c.OnHTML("div.manga-poster img", func(e *colly.HTMLElement) {
		coverURL := e.Attr("data-src")

		var coverImg []byte
		var resized bool
		var err error
		coverImg, resized, err = util.GetImageFromURL(coverURL, 3, 1*time.Second)
		if err == nil {
			mangaReturn.CoverImgURL = coverURL
			mangaReturn.CoverImgResized = resized
			mangaReturn.CoverImg = coverImg
		}
	})

	// last released chapter
	s.c.OnHTML("ul#ja-chaps > li:first-child", func(e *colly.HTMLElement) {
		chapterName := e.DOM.Find("span.name > strong").Text()
		chapter, err := extractChapter(chapterName)
		if err != nil {
			sharedErr = util.AddErrorContext(errordefs.ErrMangaAttributesNotFound.Error(), err)
			return
		}
		chapterURL := e.DOM.Find("a").AttrOr("href", "")

		mangaReturn.LastReleasedChapter = &manga.Chapter{
			URL:     chapterURL,
			Chapter: chapter,
			Name:    chapterName,
			Type:    1,
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
	if mangaReturn.Name == "" {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaAttributesNotFound)
	}

	return mangaReturn, nil
}

func (s *Source) Search(term string, limit int) ([]*models.MangaSearchResult, error) {
	s.resetCollector()

	errorContext := "error while searching manga"
	mangaSearchResults := []*models.MangaSearchResult{}
	var sharedErr error
	var mangaCount int

	s.c.OnHTML("div.manga_list-sbs div.item", func(e *colly.HTMLElement) {
		if mangaCount >= limit {
			return
		}
		var err error
		var exists bool
		mangaSearchResult := &models.MangaSearchResult{}
		mangaSearchResult.Source = "jmanga"
		mangaNameEl := e.DOM.Find("h3.manga-name > a")
		mangaSearchResult.URL, exists = mangaNameEl.Attr("href")
		if !exists {
			sharedErr = errordefs.ErrMangaAttributesNotFound
			return
		}
		mangaSearchResult.Name = mangaNameEl.Text()

		genres := e.DOM.Find("div.fd-infor")
		mangaSearchResult.Description = genres.Text()
		mangaSearchResult.CoverURL = e.DOM.Find("a.manga-poster img").AttrOr("data-src", "")
		if mangaSearchResult.CoverURL == "" {
			mangaSearchResult.CoverURL = models.DefaultCoverImgURL
		}

		lastChapter := e.DOM.Find("div.manga-detail div.fd-list > div.fdl-item:first-child > div.chapter > a")
		chapter, err := extractChapter(lastChapter.Text())
		if err != nil {
			sharedErr = util.AddErrorContext(errordefs.ErrMangaAttributesNotFound.Error(), err)
			return
		}
		mangaSearchResult.LastChapter = chapter
		mangaSearchResult.LastChapterURL = lastChapter.AttrOr("href", "")

		mangaSearchResults = append(mangaSearchResults, mangaSearchResult)
		mangaCount++
	})

	term = url.QueryEscape(term)
	mangaURL := baseSiteURL + "/?q=" + term
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

	return mangaSearchResults, nil
}
