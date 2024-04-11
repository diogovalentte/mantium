package mangahub

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL string) (*manga.Manga, error) {
	s.resetCollector()

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "mangahub.io"
	mangaReturn.URL = mangaURL

	lastChapter := &manga.Chapter{
		Type: 1,
	}
	mangaReturn.LastUploadChapter = lastChapter

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
		name = strings.Replace(name, smallTagValue, "", -1)
		name = util.RemoveLastOccurrence(name, aTagValue)

		mangaReturn.Name = name
	})

	// manga cover
	s.c.OnHTML("img.manga-thumb", func(e *colly.HTMLElement) {
		mangaReturn.CoverImgURL = e.Attr("src")
	})

	// last chapter
	isFirstUL := true
	s.c.OnHTML("ul.MWqeC:first-of-type > li:first-child a", func(e *colly.HTMLElement) {
		if !isFirstUL {
			return
		}
		isFirstUL = false
		lastChapter.URL = e.Attr("href")

		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		chapter := strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1))
		lastChapter.Chapter = chapter

		chapterName := e.DOM.Find("span._2IG5P").Text()
		lastChapter.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		lastChapter.UpdatedAt = uploadedTime
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, fmt.Errorf("manga not found, is the URL correct?")
		}
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}

	// get cover image
	coverImg, err := s.getCoverImg(mangaReturn.CoverImgURL)
	if err != nil {
		return nil, err
	}
	resizedCoverImg, err := util.ResizeImage(coverImg, uint(util.DefaultImageWidth), uint(util.DefaultImageHeight))
	if err != nil {
		// JPEG format that has an unsupported subsampling ratio
		// It's a valid image but the standard library doesn't support it
		// And other libraries use the standard library under the hood
		if err.Error() == "unsupported JPEG feature: luma/chroma subsampling ratio" {
			resizedCoverImg = coverImg
		} else {
			err = fmt.Errorf("error resizing image: %s", err)
			return nil, err
		}
	} else {
		mangaReturn.CoverImgResized = true
	}

	mangaReturn.CoverImg = resizedCoverImg

	return mangaReturn, nil
}
