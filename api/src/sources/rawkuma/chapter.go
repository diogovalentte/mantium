package rawkuma

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(_, _, _, chapterURL, _ string) (*manga.Chapter, error) {
	errorContext := "error while getting metadata of chapter"

	if chapterURL == "" {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrChapterHasNoChapterOrURL)
	}

	returnChapter, err := s.getChapterMetadataByURL(chapterURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return returnChapter, nil
}

// GetChapterMetadataByURL scrapes the manga page and return the chapter by its URL
func (s *Source) getChapterMetadataByURL(chapterURL string) (*manga.Chapter, error) {
	s.resetCollector()
	chapterReturn := &manga.Chapter{}
	chapterReturn.URL = chapterURL
	var sharedErr error

	s.c.OnHTML("time[itemprop='dateCreated']", func(e *colly.HTMLElement) {
		releaseTime, err := util.GetRFC3339Datetime(e.Attr("datetime"))
		if err != nil {
			sharedErr = err
			return
		}
		chapterReturn.UpdatedAt = releaseTime

		chapterReturn.Chapter = strings.TrimSpace(e.DOM.Parent().Find("div").Text())
		chapterReturn.Chapter = strings.Split(chapterReturn.Chapter, "Chapter ")[1]
		chapterReturn.Name = "Chapter " + chapterReturn.Chapter
	})

	err := s.c.Visit(chapterURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, errordefs.ErrChapterNotFound
		}
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}
	if chapterReturn.Name == "" {
		return nil, errordefs.ErrChapterNotFound
	}

	return chapterReturn, nil
}

// GetLastChapterMetadata scrapes the manga page and return the latest chapter
func (s *Source) GetLastChapterMetadata(_, mangaInternalID string) (*manga.Chapter, error) {
	errorContext := "error while getting last chapter metadata"

	chapters, err := s.GetChaptersMetadata("", mangaInternalID)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	if len(chapters) == 0 {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrChapterListNotFound)
	}
	chapters[0].Type = 0

	return chapters[0], nil
}

// GetChaptersMetadata scrapes the manga page and return the chapters
func (s *Source) GetChaptersMetadata(mangaURL, mangaInternalID string) ([]*manga.Chapter, error) {
	errorContext := "error while getting chapters metadata"

	if mangaInternalID == "" {
		s.resetCollector()
		var sharedErr error

		s.c.OnResponse(func(r *colly.Response) {
			body := string(r.Body)
			re := regexp.MustCompile(`wp-admin/admin-ajax\.php\?manga_id=(\d+)(?:&|$)`)
			HTMLMangaID := re.FindStringSubmatch(body)
			if len(HTMLMangaID) <= 1 {
				sharedErr = fmt.Errorf("manga ID not found in HTML response")
				return
			}
			mangaInternalID = HTMLMangaID[1]
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
	}

	chapters, err := getChapterList(mangaInternalID)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return chapters, nil
}

func getChapterList(internalMangaID string) ([]*manga.Chapter, error) {
	s := Source{}
	s.resetCollector()

	mangaID, err := strconv.Atoi(internalMangaID)
	if err != nil {
		return nil, errordefs.ErrMangaHasNoIDOrURL
	}
	if mangaID == 0 {
		return nil, util.AddErrorContext("error while getting manga chapter list", errordefs.ErrMangaHasNoIDOrURL)
	}

	chapterListURL := baseSiteURL + "/wp-admin/admin-ajax.php?page=1&action=chapter_list&manga_id="
	chapters := []*manga.Chapter{}
	var sharedErr error

	s.c.OnHTML("div#chapter-list > div > a", func(e *colly.HTMLElement) {
		chapter := &manga.Chapter{}
		chapter.URL = e.Attr("href")
		chapter.Name = e.DOM.Find("span").Text()
		chapter.Chapter = strings.Split(chapter.Name, "Chapter ")[1]
		chapter.Type = 1

		uploadedAtStr := e.DOM.Find("time").AttrOr("datetime", "")
		if uploadedAtStr != "" {
			uploadedAt, err := util.GetRFC3339Datetime(uploadedAtStr)
			if err != nil {
				sharedErr = util.AddErrorContext("error parsing chapter uploaded at datetime", err)
				return
			}
			chapter.UpdatedAt = uploadedAt
		}

		chapters = append(chapters, chapter)
	})

	err = s.c.Visit(fmt.Sprintf("%s%d", chapterListURL, mangaID))
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, errordefs.ErrChapterListNotFound
		}
		return nil, util.AddErrorContext("error while visiting chapter list URL", err)
	}
	if sharedErr != nil {
		return nil, sharedErr
	}

	return chapters, nil
}
