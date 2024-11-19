package rawkuma

import (
	"fmt"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(mangaURL, _, chapter, chapterURL, _ string) (*manga.Chapter, error) {
	errorContext := "error while getting metadata of chapter"

	if chapter == "" && chapterURL == "" {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrChapterHasNoChapterOrURL)
	}

	returnChapter := &manga.Chapter{}
	var err error
	if chapter != "" {
		returnChapter, err = s.getChapterMetadataByChapter(mangaURL, chapter)
	}
	if chapterURL != "" && (err != nil || chapter == "") {
		returnChapter, err = s.getChapterMetadataByURL(chapterURL)
	}

	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return returnChapter, nil
}

// GetChapterMetadataByURL scrapes the manga page and return the chapter by its URL
func (s *Source) getChapterMetadataByURL(_ string) (*manga.Chapter, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetChapterMetadataByChapter scrapes the manga page and return the chapter by its chapter
func (s *Source) getChapterMetadataByChapter(mangaURL string, chapter string) (*manga.Chapter, error) {
	s.resetCollector()
	chapterReturn := &manga.Chapter{}
	var sharedErr error

	var chapterFound bool
	s.c.OnHTML("ul.clstyle > li", func(e *colly.HTMLElement) {
		chapterNum := e.Attr("data-num")
		if chapterNum != chapter || chapterFound {
			return
		}
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
		releaseTime = releaseTime.Truncate(time.Second)

		chapterFound = true

		chapterReturn.URL = chapterURL
		chapterReturn.Chapter = chapterNum
		chapterReturn.Name = chapterName
		chapterReturn.UpdatedAt = releaseTime
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, errordefs.ErrMangaNotFound
		}
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}
	if !chapterFound {
		return nil, errordefs.ErrChapterNotFound
	}

	return chapterReturn, nil
}

// GetLastChapterMetadata scrapes the manga page and return the latest chapter
func (s *Source) GetLastChapterMetadata(mangaURL string, _ string) (*manga.Chapter, error) {
	s.resetCollector()

	errorContext := "error while getting last chapter metadata"
	chapterReturn := &manga.Chapter{}
	var sharedErr error

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
		releaseTime = releaseTime.Truncate(time.Second)

		chapterReturn.URL = chapterURL
		chapterReturn.Chapter = chapter
		chapterReturn.Name = chapterName
		chapterReturn.UpdatedAt = releaseTime
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
		}
		return nil, util.AddErrorContext(errorContext, err)
	}
	if sharedErr != nil {
		return nil, util.AddErrorContext(errorContext, sharedErr)
	}
	if chapterReturn.Chapter == "" {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrChapterNotFound)
	}

	return chapterReturn, nil
}

// GetChaptersMetadata scrapes the manga page and return the chapters
func (s *Source) GetChaptersMetadata(mangaURL, _ string) ([]*manga.Chapter, error) {
	s.resetCollector()

	errorContext := "error while getting chapters metadata"
	chapters := []*manga.Chapter{}
	var sharedErr error

	s.c.OnHTML("ul.clstyle > li", func(e *colly.HTMLElement) {
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
		releaseTime = releaseTime.Truncate(time.Second)

		chapterAdd := &manga.Chapter{
			URL:       chapterURL,
			Chapter:   chapter,
			Name:      chapterName,
			Type:      1,
			UpdatedAt: releaseTime,
		}

		chapters = append(chapters, chapterAdd)
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
		}
		return nil, util.AddErrorContext(errorContext, err)
	}
	if sharedErr != nil {
		return nil, util.AddErrorContext(errorContext, sharedErr)
	}

	return chapters, nil
}
