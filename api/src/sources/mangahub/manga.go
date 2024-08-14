package mangahub

import (
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL string, ignoreGetLastChapterError bool) (*manga.Manga, error) {
	s.resetCollector()

	errorContext := "error while getting manga metadata"

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "mangahub.io"
	mangaReturn.URL = mangaURL

	var sharedErr error

	// manga name
	s.c.OnHTML("h1._3xnDj", func(e *colly.HTMLElement) {
		// The h1 tag with the manga's name
		// has a small tag inside it with the
		// manga description that we don't want.
		// It can also have an <a> tag with the
		// manga's name and the word "Hot".
		name := e.Text
		smallTagValue := e.DOM.Find("small").Text()
		aTagValue := e.DOM.Find("a").Text()
		name = strings.ReplaceAll(name, smallTagValue, "")
		name = util.RemoveLastOccurrence(name, aTagValue)

		mangaReturn.Name = name
	})

	// manga cover
	s.c.OnHTML("img.manga-thumb", func(e *colly.HTMLElement) {
		mangaReturn.CoverImgURL = e.Attr("src")
	})

	// last released chapter
	isFirstUL := true
	s.c.OnHTML("ul.MWqeC:first-of-type > li:first-child a._3pfyN", func(e *colly.HTMLElement) {
		if !isFirstUL {
			return
		}
		isFirstUL = false

		chapterURL := e.Attr("href")

		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		chapter := strings.TrimSpace(strings.ReplaceAll(chapterStr, "#", ""))

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapterName = strings.TrimSpace(strings.ReplaceAll(chapterName, "- ", ""))

		releasedAt := e.DOM.Find("small.UovLc").Text()
		releaseTime, err := getMangaReleaseTime(releasedAt)
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

	if mangaReturn.LastReleasedChapter == nil && !ignoreGetLastChapterError {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrLastReleasedChapterNotFound)
	}

	// get cover image
	if mangaReturn.CoverImgURL != "" {
		coverImg, resized, err := util.GetImageFromURL(mangaReturn.CoverImgURL, 3, 1*time.Second)
		if err != nil {
			return nil, util.AddErrorContext(errorContext, err)
		}
		mangaReturn.CoverImgResized = resized
		mangaReturn.CoverImg = coverImg
	} else {
		mangaReturn.CoverImg, err = util.GetDefaultCoverImg()
		if err != nil {
			return nil, util.AddErrorContext(errorContext, err)
		}
		mangaReturn.CoverImgURL = ""
		mangaReturn.CoverImgResized = true
	}

	return mangaReturn, nil
}
