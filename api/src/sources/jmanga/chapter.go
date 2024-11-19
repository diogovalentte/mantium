package jmanga

import (
	"fmt"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

func (s *Source) GetChapterMetadata(mangaURL, _, chapter, chapterURL, _ string) (*manga.Chapter, error) {
	errorContext := "error while getting metadata of chapter"

	if chapter == "" && chapterURL == "" {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrChapterHasNoChapterOrURL)
	}

	returnChapter := &manga.Chapter{}
	var err error
	if chapterURL != "" {
		returnChapter, err = s.getChapterMetadataByURL(chapterURL)
	}
	if chapter != "" && (err != nil || chapterURL == "") {
		returnChapter, err = s.getChapterMetadataByChapter(mangaURL, chapter)
	}

	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return returnChapter, nil
}

// GetChapterMetadataByURL scrapes the manga page and return the chapter by its URL
func (s *Source) getChapterMetadataByURL(chapterURL string) (*manga.Chapter, error) {
	s.resetCollector()
	chapterReturn := &manga.Chapter{}

	s.c.OnHTML("ul.reading-list", func(e *colly.HTMLElement) {
		chapterNum := e.Attr("data-number")
		if chapterNum == "" {
			return
		}
		chapterName := fmt.Sprintf("第%s話", chapterNum)

		chapterReturn.URL = chapterURL
		chapterReturn.Chapter = chapterNum
		chapterReturn.Name = chapterName
	})

	err := s.c.Visit(chapterURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, errordefs.ErrChapterNotFound
		}
		return nil, err
	}
	if chapterReturn.Chapter == "" {
		return nil, errordefs.ErrChapterNotFound
	}

	return chapterReturn, nil
}

// GetChapterMetadataByChapter scrapes the manga page and return the chapter by its chapter
func (s *Source) getChapterMetadataByChapter(mangaURL string, chapter string) (*manga.Chapter, error) {
	s.resetCollector()
	chapterReturn := &manga.Chapter{}
	var sharedErr error

	var chapterFound bool

	s.c.OnHTML("ul#ja-chaps > li", func(e *colly.HTMLElement) {
		if chapterFound {
			return
		}

		chapterName := e.DOM.Find("span.name > strong").Text()
		chapterNum, err := extractChapter(chapterName)
		if err != nil {
			sharedErr = err
			return
		}
		if chapterNum != chapter {
			return
		}

		chapterURL := e.DOM.Find("a").AttrOr("href", "")

		chapterReturn.URL = chapterURL
		chapterReturn.Chapter = chapterNum
		chapterReturn.Name = chapterName
		chapterFound = true
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, errordefs.ErrMangaNotFound
		}
		return nil, util.AddErrorContext("error while visiting manga URL", err)
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

	s.c.OnHTML("ul#ja-chaps > li:first-child", func(e *colly.HTMLElement) {
		chapterName := e.DOM.Find("span.name > strong").Text()
		chapter, err := extractChapter(chapterName)
		if err != nil {
			sharedErr = err
			return
		}
		chapterURL := e.DOM.Find("a").AttrOr("href", "")

		chapterReturn = &manga.Chapter{
			URL:     chapterURL,
			Chapter: chapter,
			Name:    chapterName,
		}
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
		return nil, errordefs.ErrChapterNotFound
	}

	return chapterReturn, nil
}

// GetChaptersMetadata scrapes the manga page and return the chapters
func (s *Source) GetChaptersMetadata(mangaURL, _ string) ([]*manga.Chapter, error) {
	s.resetCollector()

	errorContext := "error while getting chapters metadata"
	chapters := []*manga.Chapter{}
	var sharedErr error

	s.c.OnHTML("ul#ja-chaps > li", func(e *colly.HTMLElement) {
		chapterName := e.DOM.Find("span.name > strong").Text()
		chapter, err := extractChapter(chapterName)
		if err != nil {
			sharedErr = err
			return
		}
		chapterURL := e.DOM.Find("a").AttrOr("href", "")

		chapterAdd := &manga.Chapter{
			URL:     chapterURL,
			Chapter: chapter,
			Name:    chapterName,
			Type:    1,
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
