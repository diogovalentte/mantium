package mangahub

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/manga-dashboard-api/api/src/manga"
)

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(mangaURL string, chapter string, chapterURL string) (*manga.Chapter, error) {
	if chapter == "" && chapterURL == "" {
		return nil, fmt.Errorf("chapter or chapter URL is required")
	}
	return s.GetChapterMetadataByChapter(mangaURL, chapter)
}

// GetChapterMetadataByURL scrapes the manga page and return the chapter by its URL
// Not implemented
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

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		chapterReturn.UpdatedAt = uploadedTime
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}
	if !chapterFound {
		return nil, fmt.Errorf("chapter not found, is the URL or chapter correct?")
	}

	return chapterReturn, nil
}

// GetLastChapterMetadata scrapes the manga page and return the latest chapter
func (s *Source) GetLastChapterMetadata(mangaURL string) (*manga.Chapter, error) {
	s.resetCollector()
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

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		chapterReturn.UpdatedAt = uploadedTime
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}

	return chapterReturn, nil
}

// GetChaptersMetadata scrapes the manga page and return the chapters
func (s *Source) GetChaptersMetadata(mangaURL string) ([]*manga.Chapter, error) {
	s.resetCollector()
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

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		chapterReturn.UpdatedAt = uploadedTime

		chapters = append(chapters, chapterReturn)
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}

	return chapters, nil
}
