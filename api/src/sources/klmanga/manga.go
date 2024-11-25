package klmanga

import (
	"fmt"
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
	mangaReturn.Source = "klmanga"
	mangaReturn.URL = mangaURL

	var sharedErr error

	// manga name
	s.c.OnHTML("h1.name", func(e *colly.HTMLElement) {
		name := e.Text
		name = strings.TrimSuffix(name, " (RAW – Free)")
		name = strings.TrimSuffix(name, " (RAW - Free)")
		name = strings.TrimSuffix(name, " (Raw – Free)")
		mangaReturn.Name = strings.TrimSuffix(name, " (Raw - Free)")
	})

	// manga cover
	s.c.OnHTML("div.main-thumb > img", func(e *colly.HTMLElement) {
		coverURL := e.Attr("src")

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
	s.c.OnHTML("div.chapter-box > h4:first-child > a", func(e *colly.HTMLElement) {
		chapterName := strings.TrimSpace(e.DOM.Find("span").Text())
		chapter, err := extractChapter(chapterName)
		if err != nil {
			sharedErr = err
			return
		}
		chapterURL := e.Attr("href")

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
		return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
	}

	return mangaReturn, nil
}

func (s *Source) Search(term string, limit int) ([]*models.MangaSearchResult, error) {
	errorContext := "error while searching manga"
	mangaSearchResults := []*models.MangaSearchResult{}
	term = strings.ReplaceAll(term, " ", "+")
	pageNumber := 1
	var mangaCount int

	for mangaCount < limit {
		s.resetCollector()
		var sharedErr error
		nextPage := false

		s.c.OnHTML("div.row > div.col-sm-4 > div.entry", func(e *colly.HTMLElement) {
			if mangaCount >= limit {
				return
			}
			mangaSearchResult := &models.MangaSearchResult{}
			mangaSearchResult.Source = "klmanga"
			mangaSearchResult.URL = e.DOM.Find("h2 > a").AttrOr("href", "")

			name := e.DOM.Find("h2 > a").Text()
			name = strings.TrimSuffix(name, " (RAW – Free)")
			name = strings.TrimSuffix(name, " (RAW - Free)")
			name = strings.TrimSuffix(name, " (Raw – Free)")
			mangaSearchResult.Name = strings.TrimSuffix(name, " (Raw - Free)")
			mangaSearchResult.Description = e.DOM.Find("div.genres").Text()
			mangaSearchResult.CoverURL = e.DOM.Find("div.thumb > a.thumb > img").AttrOr("src", "")
			if mangaSearchResult.CoverURL == "" {
				mangaSearchResult.CoverURL = models.DefaultCoverImgURL
			}

			lastChapter := e.DOM.Find("div.thumb > a.meta-info")
			chapter, err := extractChapter(lastChapter.Text())
			if err != nil {
				sharedErr = err
				return
			}
			mangaSearchResult.LastChapter = chapter
			mangaSearchResult.LastChapterURL = lastChapter.AttrOr("href", "")

			mangaSearchResults = append(mangaSearchResults, mangaSearchResult)
			mangaCount++
		})

		s.c.OnHTML("span.page-numbers.current + a.page-numbers", func(_ *colly.HTMLElement) {
			nextPage = true
		})

		mangaURL := fmt.Sprintf("%s/page/%d/?s=%s", baseSiteURL, pageNumber, term)
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
		if !nextPage {
			break
		}
		pageNumber++
	}

	return mangaSearchResults, nil
}
