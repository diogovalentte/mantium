package mangahub

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(mangaURL string, chapter string, chapterURL string) (*manga.Chapter, error) {
	errorContext := "error while getting metadata of chapter with chapter '%s' and URL '%s', and manga URL '%s'"

	if chapter == "" && chapterURL == "" {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, chapter, chapterURL, mangaURL), errordefs.ErrChapterDoesntHaveChapterAndURL)
	}

	returnChapter := &manga.Chapter{}
	var err error
	if chapter != "" {
		returnChapter, err = s.GetChapterMetadataByChapter(mangaURL, chapter)
	}
	if chapterURL != "" && (err != nil || chapter == "") {
		returnChapter, err = s.GetChapterMetadataByURL(chapterURL)
	}

	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, chapter, chapterURL, mangaURL), err)
	}

	return returnChapter, nil
}

// GetChapterMetadataByURL scrapes the manga page and return the chapter by its URL
func (s *Source) GetChapterMetadataByURL(_ string) (*manga.Chapter, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetChapterMetadataByChapter scrapes the manga page and return the chapter by its chapter
func (s *Source) GetChapterMetadataByChapter(mangaURL string, chapter string) (*manga.Chapter, error) {
	s.resetCollector()
	chapterReturn := &manga.Chapter{
		Chapter: chapter,
	}
	var sharedErr error

	chapterFound := false
	s.c.OnHTML("ul.MWqeC:first-of-type > li a._3pfyN", func(e *colly.HTMLElement) {
		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		scrapedChapter := strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1))
		if scrapedChapter != chapter {
			return
		}
		chapterFound = true

		chapterReturn.URL = e.Attr("href")

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapterReturn.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		releasedAt := e.DOM.Find("small.UovLc").Text()
		releaseTime, err := getMangaReleaseTime(releasedAt)
		if err != nil {
			sharedErr = err
			return
		}
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
func (s *Source) GetLastChapterMetadata(mangaURL string) (*manga.Chapter, error) {
	s.resetCollector()

	errorContext := "error while getting last chapter metadata"
	chapterReturn := &manga.Chapter{}
	var sharedErr error

	isFirstUL := true
	s.c.OnHTML("ul.MWqeC:first-of-type > li:first-child a._3pfyN", func(e *colly.HTMLElement) {
		if !isFirstUL {
			return
		}
		isFirstUL = false
		chapterReturn.URL = e.Attr("href")

		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		chapter := strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1))
		chapterReturn.Chapter = chapter

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapterReturn.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		releaseddAt := e.DOM.Find("small.UovLc").Text()
		releaseTime, err := getMangaReleaseTime(releaseddAt)
		if err != nil {
			sharedErr = err
			return
		}
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

	return chapterReturn, nil
}

// GetChaptersMetadata scrapes the manga page and return the chapters
func (s *Source) GetChaptersMetadata(mangaURL string) ([]*manga.Chapter, error) {
	s.resetCollector()

	errorContext := "error while getting chapters metadata"
	chapters := []*manga.Chapter{}
	var sharedErr error

	s.c.OnHTML("li._287KE a._3pfyN", func(e *colly.HTMLElement) {
		chapterReturn := &manga.Chapter{}

		chapterReturn.URL = e.Attr("href")

		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		chapter := strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1))
		chapterReturn.Chapter = chapter

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapterReturn.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		releasedAt := e.DOM.Find("small.UovLc").Text()
		releaseTime, err := getMangaReleaseTime(releasedAt)
		if err != nil {
			sharedErr = err
			return
		}
		chapterReturn.UpdatedAt = releaseTime

		chapters = append(chapters, chapterReturn)
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
