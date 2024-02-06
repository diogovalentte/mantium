package mangahub

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/manga-dashboard-api/src/manga"
)

// GetChapterMetadata returns a chapter by its number or URL
func (s *Source) GetChapterMetadata(mangaURL string, chapterNumber manga.Number, _ string) (*manga.Chapter, error) {
	return s.GetChapterMetadataByNumber(mangaURL, chapterNumber)
}

// GetChapterMetadataByURL scrapes the manga page and return the chapter by its URL
// Not implemented
func (s *Source) GetChapterMetadataByURL(_ string) (*manga.Chapter, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetChapterMetadataByNumber scrapes the manga page and return the chapter by its number
func (s *Source) GetChapterMetadataByNumber(mangaURL string, chapterNumber manga.Number) (*manga.Chapter, error) {
	s.resetCollector()
	chapter := &manga.Chapter{
		Number: chapterNumber,
	}
	var sharedErr error

	chapterFound := false
	s.c.OnHTML("ul.MWqeC:first-of-type > li a._3pfyN", func(e *colly.HTMLElement) {
		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		scrapedChapterNumber, err := strconv.ParseFloat(strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1)), 32)
		if err != nil {
			sharedErr = err
			return
		}
		if manga.Number(scrapedChapterNumber) != chapterNumber {
			return
		}
		chapterFound = true

		chapter.URL = e.Attr("href")

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapter.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		chapter.UpdatedAt = uploadedTime
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}
	if !chapterFound {
		return nil, fmt.Errorf("chapter not found, is the URL or chapter number correct?")
	}

	return chapter, nil
}

// GetLastChapterMetadata scrapes the manga page and return the latest chapter
func (s *Source) GetLastChapterMetadata(mangaURL string) (*manga.Chapter, error) {
	s.resetCollector()
	chapter := &manga.Chapter{}
	var sharedErr error

	isFirstUL := true
	s.c.OnHTML("ul.MWqeC:first-of-type > li:first-child a._3pfyN", func(e *colly.HTMLElement) {
		if !isFirstUL {
			return
		}
		isFirstUL = false
		chapter.URL = e.Attr("href")

		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		chapterNumber, err := strconv.ParseFloat(strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1)), 32)
		if err != nil {
			sharedErr = err
			return
		}
		chapter.Number = manga.Number(chapterNumber)

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapter.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		chapter.UpdatedAt = uploadedTime
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}

	return chapter, nil
}

// GetChaptersMetadata scrapes the manga page and return the chapters
func (s *Source) GetChaptersMetadata(mangaURL string) ([]*manga.Chapter, error) {
	s.resetCollector()
	chapters := []*manga.Chapter{}

	var sharedErr error
	s.c.OnHTML("li._287KE a._3pfyN", func(e *colly.HTMLElement) {
		chapter := &manga.Chapter{}

		chapter.URL = e.Attr("href")

		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		chapterNumber, err := strconv.ParseFloat(strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1)), 32)
		if err != nil {
			sharedErr = err
			return
		}
		chapter.Number = manga.Number(chapterNumber)

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapter.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		chapter.UpdatedAt = uploadedTime

		chapters = append(chapters, chapter)
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
