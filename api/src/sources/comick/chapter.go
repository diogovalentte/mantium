package comick

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(mangaURL string, chapter string, chapterURL string) (*manga.Chapter, error) {
	if chapter == "" && chapterURL == "" {
		return nil, fmt.Errorf("chapter or chapter URL is required")
	}
	return s.GetChapterMetadataByChapter(mangaURL, chapter)
}

// GetChapterMetadataByURL scrapes the manga page and return the chapter by its URL
func (s *Source) GetChapterMetadataByURL(_ string) (*manga.Chapter, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetChapterMetadataByChapter scrapes the manga page and return the chapter by its chapter
func (s *Source) GetChapterMetadataByChapter(mangaURL string, chapter string) (*manga.Chapter, error) {
	s.checkClient()

	mangaHID, err := s.getMangaHID(mangaURL)
	if err != nil {
		return nil, err
	}

	chapterReturn := &manga.Chapter{}

	mangaAPIURL := fmt.Sprintf("%s/comic/%s/chapters?lang=en&limit=1&chap=%s", baseAPIURL, mangaHID, chapter)
	resp, err := s.client.Request("GET", mangaAPIURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chaptersAPIResp getChaptersAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&chaptersAPIResp); err != nil {
		return nil, err
	}

	if len(chaptersAPIResp.Chapters) == 0 {
		return nil, fmt.Errorf("chapter not found")
	}

	err = getChapterFromResp(chaptersAPIResp.Chapters[0], chapterReturn, chapter, mangaURL)
	if err != nil {
		return nil, err
	}

	return chapterReturn, nil
}

// GetLastChapterMetadata returns the last chapter of a manga by its URL
func (s *Source) GetLastChapterMetadata(mangaURL string) (*manga.Chapter, error) {
	s.checkClient()

	mangaHID, err := s.getMangaHID(mangaURL)
	if err != nil {
		return nil, err
	}

	chapterReturn := &manga.Chapter{}

	mangaAPIURL := fmt.Sprintf("%s/comic/%s/chapters?lang=en&limit=1", baseAPIURL, mangaHID) // default order is by chapter desc
	resp, err := s.client.Request("GET", mangaAPIURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chaptersAPIResp getChaptersAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&chaptersAPIResp); err != nil {
		return nil, err
	}

	if len(chaptersAPIResp.Chapters) == 0 {
		return nil, fmt.Errorf("chapter not found")
	}

	err = getChapterFromResp(chaptersAPIResp.Chapters[0], chapterReturn, chaptersAPIResp.Chapters[0].Chap, mangaURL)
	if err != nil {
		return nil, err
	}

	return chapterReturn, nil
}

// GetChaptersMetadata returns the chapters of a manga by its URL
func (s *Source) GetChaptersMetadata(mangaURL string) ([]*manga.Chapter, error) {
	s.checkClient()

	chaptersChan := make(chan *manga.Chapter)
	errChan := make(chan error)
	done := make(chan struct{})

	go generateMangaChapters(s, mangaURL, chaptersChan, errChan)

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
		return nil, err
	}
}

type getChaptersAPIResponse struct {
	Chapters []getChapterAPIResponse `json:"chapters"`
}

type getChapterAPIResponse struct {
	Chap      string `json:"chap"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	HID       string `json:"hid"`
}

func generateMangaChapters(s *Source, mangaURL string, chaptersChan chan *manga.Chapter, errChan chan error) {
	defer close(chaptersChan)

	mangaHID, err := s.getMangaHID(mangaURL)
	if err != nil {
		errChan <- err
		return
	}

	currentPage := 1
	for {

		mangaAPIURL := fmt.Sprintf("%s/comic/%s/chapters?lang=en&page=%d", baseAPIURL, mangaHID, currentPage)
		resp, err := s.client.Request("GET", mangaAPIURL, nil)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		var chaptersAPIResp getChaptersAPIResponse
		if err = json.NewDecoder(resp.Body).Decode(&chaptersAPIResp); err != nil {
			errChan <- err
			return
		}

		if len(chaptersAPIResp.Chapters) == 0 {
			break
		}

		for _, chapter := range chaptersAPIResp.Chapters {
			chapterReturn := &manga.Chapter{}
			err = getChapterFromResp(chapter, chapterReturn, chapter.Chap, mangaURL)
			if err != nil {
				errChan <- err
				return
			}
			chaptersChan <- chapterReturn
		}

		currentPage++
	}
}

func getChapterFromResp(chapterResp getChapterAPIResponse, chapterReturn *manga.Chapter, chapter string, mangaURL string) error {
	if chapterResp.Chap == "" && chapterResp.Title == "" {
		chapterReturn.Chapter = chapter
		chapterReturn.Name = fmt.Sprintf("Ch. %s", chapter)
	} else {
		if chapterResp.Chap == "" {
			chapterReturn.Chapter = chapterResp.Title
		} else {
			chapterReturn.Chapter = chapterResp.Chap
		}

		if chapterResp.Title == "" {
			chapterReturn.Name = fmt.Sprintf("Ch. %s", chapterReturn.Chapter)
		} else {
			chapterReturn.Name = chapterResp.Title
		}
	}
	chapterReturn.URL = fmt.Sprintf("%s/%s", mangaURL, chapterResp.HID)
	chapterCreatedAt, err := util.GetRFC3339Datetime(chapterResp.CreatedAt)
	if err != nil {
		return nil
	}
	chapterCreatedAt = chapterCreatedAt.Truncate(time.Second)
	chapterReturn.UpdatedAt = chapterCreatedAt

	return nil
}
