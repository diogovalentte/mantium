package mangadex

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/diogovalentte/manga-dashboard-api/src/manga"
)

// GetChapterMetadata returns a chapter by its number or URL
func (s *Source) GetChapterMetadata(_ string, _ manga.Number, chapterURL string) (*manga.Chapter, error) {
	return s.GetChapterMetadataByURL(chapterURL)
}

// GetChapterMetadataByURL scrapes the manga page and return the chapter by its URL
func (s *Source) GetChapterMetadataByURL(chapterURL string) (*manga.Chapter, error) {
	s.checkClient()

	chapterReturn := &manga.Chapter{}
	chapterReturn.URL = chapterURL

	chapterID, err := getChapterID(chapterURL)
	if err != nil {
		return nil, err
	}

	chapterAPIURL := fmt.Sprintf("%s/chapter/%s", baseAPIURL, chapterID)
	resp, err := s.client.Request(context.Background(), "GET", chapterAPIURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chapterAPIResp getChapterAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&chapterAPIResp); err != nil {
		return nil, err
	}

	attributes := &chapterAPIResp.Data.Attributes

	chapterNumber, err := strconv.ParseFloat(attributes.Chapter, 32)
	if err != nil {
		return nil, err
	}
	chapterReturn.Number = manga.Number(chapterNumber)

	chapterReturn.Name = attributes.Title
	if chapterReturn.Name == "" {
		chapterReturn.Name = fmt.Sprintf("Ch. %v", chapterReturn.Number)
	}

	chapterCreatedAt, err := getDatetime(attributes.PublishAt)
	chapterReturn.UpdatedAt = chapterCreatedAt

	return chapterReturn, nil
}

// GetChapterMetadataByNumber scrapes the manga page and return the chapter by its number
func (s *Source) GetChapterMetadataByNumber(_ string, _ manga.Number) (*manga.Chapter, error) {
	return nil, fmt.Errorf("not implemented")
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

func (s *Source) GetLastChapterMetadata(mangaURL string) (*manga.Chapter, error) {
	s.checkClient()

	mangaID, err := getMangaID(mangaURL)
	if err != nil {
		return nil, err
	}

	// URL gets the last chapter of the manga
	mangaAPIURL := fmt.Sprintf("%s/manga/%s/feed?translatedLanguage[]=en&order[chapter]=desc&limit=1&offset=0", baseAPIURL, mangaID)
	resp, err := s.client.Request(context.Background(), "GET", mangaAPIURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var feedAPIResp getMangaFeedAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&feedAPIResp); err != nil {
		return nil, err
	}

	chapterReturn := &manga.Chapter{}
	chapterReturn.URL = fmt.Sprintf("%s/chapter/%s", baseSiteURL, feedAPIResp.Data[0].ID)

	attributes := &feedAPIResp.Data[0].Attributes

	chapterNumber, err := strconv.ParseFloat(attributes.Chapter, 32)
	if err != nil {
		return nil, err
	}
	chapterReturn.Number = manga.Number(chapterNumber)

	chapterReturn.Name = attributes.Title
	if chapterReturn.Name == "" {
		chapterReturn.Name = fmt.Sprintf("Ch. %v", chapterReturn.Number)
	}

	chapterCreatedAt, err := getDatetime(attributes.PublishAt)
	if err != nil {
		return nil, err
	}
	chapterReturn.UpdatedAt = chapterCreatedAt

	return chapterReturn, nil
}

func (s *Source) GetChaptersMetadata(mangaURL string) ([]*manga.Chapter, error) {
	s.checkClient()

	chaptersChan := make(chan *manga.Chapter)
	errChan := make(chan error)
	done := make(chan struct{})

	go generateMangaFeed(s, mangaURL, chaptersChan, errChan)

	var chapters []*manga.Chapter
	go func() {
		occurrence := make(map[manga.Number]bool)
		for chapter := range chaptersChan {
			if occurrence[chapter.Number] {
				continue
			} else {
				occurrence[chapter.Number] = true
				chapters = append(chapters, chapter)
			}
		}
		close(done)
	}()

	select {
	case <-done:
		return chapters, nil
	case err := <-errChan:
		return nil, err
	}
}

// generateMangaFeed generates the chapters of a manga and sends them to the channel
// It sends an error to the error channel if something goes wrong
// It closes the chapters channel when there is no more chapters to send
// It requests the mangas from the API using the chapter number for ordering
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
		resp, err := s.client.Request(context.Background(), "GET", mangaAPIURL, nil)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		var feedAPIResp getMangaFeedAPIResponse
		if err = json.NewDecoder(resp.Body).Decode(&feedAPIResp); err != nil {
			errChan <- err
			return
		}

		totalChapters = feedAPIResp.Total
		requestOffset += requestLimit
		for _, chapterReq := range feedAPIResp.Data {
			chapterReturn := &manga.Chapter{}
			chapterReturn.URL = fmt.Sprintf("%s/chapter/%s", baseSiteURL, chapterReq.ID)

			attributes := &chapterReq.Attributes

			chapterNumber, err := strconv.ParseFloat(attributes.Chapter, 32)
			if err != nil {
				errChan <- err
				return
			}
			chapterReturn.Number = manga.Number(chapterNumber)

			chapterReturn.Name = attributes.Title
			if chapterReturn.Name == "" {
				chapterReturn.Name = fmt.Sprintf("Ch. %v", chapterReturn.Number)
			}

			chapterCreatedAt, err := getDatetime(attributes.PublishAt)
			if err != nil {
				errChan <- err
				return
			}
			chapterReturn.UpdatedAt = chapterCreatedAt

			chaptersChan <- chapterReturn
		}
	}
}

type chapterAttributes struct {
	Title              string `json:"title"`
	Volume             string `json:"volume"`
	Chapter            string `json:"chapter"`
	Pages              int    `json:"pages"`
	TranslatedLanguage string `json:"translatedLanguage"`
	Uploader           string `json:"uploader"`
	ExternalURL        string `json:"externalURL"`
	Version            int    `json:"version"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
	PublishAt          string `json:"publishAt"`
	ReadableAt         string `json:"readableAt"`
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

// getChapterID returns the ID of a chapter given its URL
// URL should be like: https://mangadex.org/chapter/87ebd557-8394-4f16-8afe-a8644e555ddc
func getChapterID(chapterURL string) (string, error) {
	pattern := `https://mangadex\.org/chapter/([0-9a-fA-F-]+)`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	matches := re.FindStringSubmatch(chapterURL)
	if len(matches) < 2 {
		return "", fmt.Errorf("chapter ID not found in URL")
	}

	return matches[1], nil
}
