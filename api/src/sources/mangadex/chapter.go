package mangadex

import (
	"context"
	"fmt"
	"regexp"

	"github.com/diogovalentte/mantium/api/src/errors"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(mangaURL string, chapter string, chapterURL string) (*manga.Chapter, error) {
	errorContext := "Error while getting metadata of chapter with chapter '%s' and URL '%s', and manga URL '%s'"

	if chapter == "" && chapterURL == "" {
		return nil, util.AddErrorContext(fmt.Errorf("Chapter or chapter URL is required"), fmt.Sprintf(errorContext, chapter, chapterURL, mangaURL))
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
		return nil, util.AddErrorContext(err, fmt.Sprintf(errorContext, chapter, chapterURL, mangaURL))
	}

	return returnChapter, nil
}

// getChapterMetadataByURL scrapes the manga page and return the chapter by its URL
func (s *Source) getChapterMetadataByURL(chapterURL string) (*manga.Chapter, error) {
	s.checkClient()

	chapterReturn := &manga.Chapter{}
	chapterReturn.URL = chapterURL

	chapterID, err := getChapterID(chapterURL)
	if err != nil {
		return nil, err
	}

	chapterAPIURL := fmt.Sprintf("%s/chapter/%s", baseAPIURL, chapterID)
	var chapterAPIResp getChapterAPIResponse
	_, err = s.client.Request(context.Background(), "GET", chapterAPIURL, nil, &chapterAPIResp)
	if err != nil {
		return nil, err
	}

	attributes := &chapterAPIResp.Data.Attributes

	if attributes.Chapter == "" && attributes.Title == "" {
		chapterReturn.Chapter = attributes.Chapter
		chapterReturn.Name = attributes.Title
	} else {
		if attributes.Chapter == "" {
			chapterReturn.Chapter = attributes.Title
		} else {
			chapterReturn.Chapter = attributes.Chapter
		}

		if attributes.Title == "" {
			chapterReturn.Name = fmt.Sprintf("Ch. %s", chapterReturn.Chapter)
		} else {
			chapterReturn.Name = attributes.Title
		}
	}

	chapterReturn.UpdatedAt, err = util.GetRFC3339Datetime(attributes.PublishAt)
	if err != nil {
		return nil, err
	}

	return chapterReturn, nil
}

type getChapterAPIResponse struct {
	Result   string `json:"result"`
	Response string `json:"response"`
	Data     struct {
		ID            string                `json:"id"`
		Type          string                `json:"type"`
		Attributes    chapterAttributes     `json:"attributes"`
		Relationships []genericRelationship `json:"relationships"`
	}
}

// getChapterMetadataByChapter scrapes the manga page and return the chapter by its chapter
func (s *Source) getChapterMetadataByChapter(_ string, _ string) (*manga.Chapter, error) {
	return nil, fmt.Errorf("Not implemented")
}

// GetLastChapterMetadata returns the last chapter of a manga by its URL
func (s *Source) GetLastChapterMetadata(mangaURL string) (*manga.Chapter, error) {
	s.checkClient()

	errorContext := "Error while getting last chapter metadata"

	mangaID, err := getMangaID(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	// URL gets the last chapter of the manga
	mangaAPIURL := fmt.Sprintf("%s/manga/%s/feed?translatedLanguage[]=en&order[chapter]=desc&limit=1&offset=0", baseAPIURL, mangaID)
	var feedAPIResp getMangaFeedAPIResponse
	_, err = s.client.Request(context.Background(), "GET", mangaAPIURL, nil, &feedAPIResp)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	if len(feedAPIResp.Data) == 0 {
		return nil, util.AddErrorContext(errors.ErrLastReleasedChapterNotFound, errorContext)
	}

	chapterReturn := &manga.Chapter{}
	chapterReturn.URL = fmt.Sprintf("%s/chapter/%s", baseSiteURL, feedAPIResp.Data[0].ID)

	attributes := &feedAPIResp.Data[0].Attributes

	if attributes.Chapter == "" && attributes.Title == "" {
		chapterReturn.Chapter = attributes.Chapter
		chapterReturn.Name = attributes.Title
	} else {
		if attributes.Chapter == "" {
			chapterReturn.Chapter = attributes.Title
		} else {
			chapterReturn.Chapter = attributes.Chapter
		}

		if attributes.Title == "" {
			chapterReturn.Name = fmt.Sprintf("Ch. %s", chapterReturn.Chapter)
		} else {
			chapterReturn.Name = attributes.Title
		}
	}

	chapterReturn.UpdatedAt, err = util.GetRFC3339Datetime(attributes.PublishAt)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	return chapterReturn, nil
}

// GetChaptersMetadata returns the chapters of a manga by its URL
func (s *Source) GetChaptersMetadata(mangaURL string) ([]*manga.Chapter, error) {
	s.checkClient()

	errorContext := "Error while getting chapters metadata"

	chaptersChan := make(chan *manga.Chapter)
	errChan := make(chan error)
	done := make(chan struct{})

	go generateMangaFeed(s, mangaURL, chaptersChan, errChan)

	var chapters []*manga.Chapter
	go func() {
		occurrence := make(map[string]bool)
		for chapter := range chaptersChan {
			if occurrence[chapter.Chapter] {
				continue
			}
			occurrence[chapter.Chapter] = true
			chapters = append(chapters, chapter)
		}
		close(done)
	}()

	select {
	case <-done:
		return chapters, nil
	case err := <-errChan:
		return nil, util.AddErrorContext(err, errorContext)
	}
}

// generateMangaFeed generates the chapters of a manga and sends them to the channel.
// It sends an error to the error channel if something goes wrong.
// It closes the chapters channel when there is no more chapters to send.
// It requests the mangas from the API using the chapter for ordering.
func generateMangaFeed(s *Source, mangaURL string, chaptersChan chan<- *manga.Chapter, errChan chan<- error) {
	defer close(chaptersChan)

	mangaID, err := getMangaID(mangaURL)
	if err != nil {
		errChan <- err
		return
	}

	requestLimit := 500
	requestOffset := 0
	totalChapters := 1

	for totalChapters >= requestOffset {
		mangaAPIURL := fmt.Sprintf("%s/manga/%s/feed?translatedLanguage[]=en&order[chapter]=desc&limit=%d&offset=%d", baseAPIURL, mangaID, requestLimit, requestOffset)
		var feedAPIResp getMangaFeedAPIResponse
		_, err = s.client.Request(context.Background(), "GET", mangaAPIURL, nil, &feedAPIResp)
		if err != nil {
			errChan <- err
			return
		}

		totalChapters = feedAPIResp.Total
		requestOffset += requestLimit
		for _, chapterReq := range feedAPIResp.Data {
			chapterReturn := &manga.Chapter{}
			chapterReturn.URL = fmt.Sprintf("%s/chapter/%s", baseSiteURL, chapterReq.ID)

			attributes := &chapterReq.Attributes

			if attributes.Chapter == "" && attributes.Title == "" {
				chapterReturn.Chapter = attributes.Chapter
				chapterReturn.Name = attributes.Title
			} else {
				if attributes.Chapter == "" {
					chapterReturn.Chapter = attributes.Title
				} else {
					chapterReturn.Chapter = attributes.Chapter
				}

				if attributes.Title == "" {
					chapterReturn.Name = fmt.Sprintf("Ch. %s", chapterReturn.Chapter)
				} else {
					chapterReturn.Name = attributes.Title
				}
			}

			chapterReturn.UpdatedAt, err = util.GetRFC3339Datetime(attributes.PublishAt)
			if err != nil {
				errChan <- err
				return
			}

			chaptersChan <- chapterReturn
		}
	}
}

type getMangaFeedAPIResponse struct {
	Result   string `json:"result"`
	Response string `json:"response"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Total    int    `json:"total"`
	Data     []struct {
		ID            string                `json:"id"`
		Type          string                `json:"type"`
		Attributes    chapterAttributes     `json:"attributes"`
		Relationships []genericRelationship `json:"relationships"`
	}
}

// getChapterID returns the ID of a chapter given its URL.
// URL should be like: https://mangadex.org/chapter/87ebd557-8394-4f16-8afe-a8644e555ddc
func getChapterID(chapterURL string) (string, error) {
	errorContext := "Error while getting chapter ID"

	pattern := `https://mangadex\.org/chapter/([0-9a-fA-F-]+)`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", util.AddErrorContext(err, errorContext)
	}

	matches := re.FindStringSubmatch(chapterURL)
	if len(matches) < 2 {
		return "", util.AddErrorContext(fmt.Errorf("Chapter ID not found in URL"), errorContext)
	}

	return matches[1], nil
}
