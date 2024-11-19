package comick

import (
	"fmt"
	"strings"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(mangaURL, _, chapter, chapterURL, _ string) (*manga.Chapter, error) {
	errorContext := "error while getting metadata of chapter"

	if chapter == "" && chapterURL == "" {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrChapterHasNoChapterOrURL)
	}

	returnChapter := &manga.Chapter{}
	var err error
	if chapterURL != "" {
		returnChapter, err = s.getChapterMetadataByURL(chapterURL, mangaURL)
	}
	if chapter != "" && (err != nil || chapterURL == "") {
		// not so reliable, can return the wrong chapter
		returnChapter, err = s.getChapterMetadataByChapter(mangaURL, chapter)
	}

	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return returnChapter, nil
}

// getChapterMetadataByURL scrapes the manga page and return the chapter by its URL
func (s *Source) getChapterMetadataByURL(chapterURL, mangaURL string) (*manga.Chapter, error) {
	s.checkClient()

	chapterHID, err := getChapterHID(chapterURL)
	if err != nil {
		return nil, err
	}

	mangaAPIURL := fmt.Sprintf("%s/chapter/%s", baseAPIURL, chapterHID)
	var chapterAPIResp getChapterAPIResponse
	_, err = s.client.Request("GET", mangaAPIURL, nil, &chapterAPIResp)
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, errordefs.ErrChapterNotFound
		}
		return nil, err
	}

	chapterReturn, err := getChapterFromResp(chapterAPIResp.Chapter, chapterAPIResp.Chapter.Chap, mangaURL)
	if err != nil {
		return nil, err
	}

	return chapterReturn, nil
}

type getChapterAPIResponse struct {
	Chapter chapterAPIResponse `json:"chapter"`
}

// getChapterMetadataByChapter scrapes the manga page and return the chapter by its chapter
func (s *Source) getChapterMetadataByChapter(mangaURL string, chapter string) (*manga.Chapter, error) {
	s.checkClient()

	mangaHID, err := s.getMangaHID(mangaURL)
	if err != nil {
		return nil, err
	}

	mangaAPIURL := fmt.Sprintf("%s/comic/%s/chapters?lang=en&limit=1&chap=%s", baseAPIURL, mangaHID, chapter)
	var chaptersAPIResp getChaptersAPIResponse
	_, err = s.client.Request("GET", mangaAPIURL, nil, &chaptersAPIResp)
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, errordefs.ErrChapterNotFound
		}
		return nil, err
	}

	if len(chaptersAPIResp.Chapters) == 0 {
		return nil, fmt.Errorf("chapter not found")
	}

	chapterReturn, err := getChapterFromResp(chaptersAPIResp.Chapters[0], chapter, mangaURL)
	if err != nil {
		return nil, err
	}

	return chapterReturn, nil
}

// GetLastChapterMetadata returns the last chapter of a manga by its URL
func (s *Source) GetLastChapterMetadata(mangaURL, _ string) (*manga.Chapter, error) {
	s.checkClient()

	errorContext := "error while getting last chapter metadata of manga with URL '%s'"

	mangaHID, err := s.getMangaHID(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	mangaAPIURL := fmt.Sprintf("%s/comic/%s/chapters?lang=en&limit=1", baseAPIURL, mangaHID) // default order is by chapter desc
	var chaptersAPIResp getChaptersAPIResponse
	_, err = s.client.Request("GET", mangaAPIURL, nil, &chaptersAPIResp)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	if len(chaptersAPIResp.Chapters) == 0 {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), errordefs.ErrLastReleasedChapterNotFound)
	}

	chapterReturn, err := getChapterFromResp(chaptersAPIResp.Chapters[0], chaptersAPIResp.Chapters[0].Chap, mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	return chapterReturn, nil
}

// GetChaptersMetadata returns the chapters of a manga by its URL
func (s *Source) GetChaptersMetadata(mangaURL, _ string) ([]*manga.Chapter, error) {
	s.checkClient()

	errorContext := "error while getting chapters metadata"

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
		return nil, util.AddErrorContext(errorContext, err)
	}
}

type getChaptersAPIResponse struct {
	Chapters []chapterAPIResponse `json:"chapters"`
}

type chapterAPIResponse struct {
	Chap      string `json:"chap"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	HID       string `json:"hid"`
}

// generateMangaChapters generates the chapters of a manga and sends them to the channel.
// It sends an error to the error channel if something goes wrong.
// It closes the chapters channel when there is no more chapters to send.
// It requests the mangas from the API using the chapter for ordering.
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
		var chaptersAPIResp getChaptersAPIResponse
		_, err = s.client.Request("GET", mangaAPIURL, nil, &chaptersAPIResp)
		if err != nil {
			if util.ErrorContains(err, "non-200 status code -> (404)") {
				err = util.AddErrorContext(err.Error(), errordefs.ErrMangaNotFound)
			}
			errChan <- err
			return
		}

		if len(chaptersAPIResp.Chapters) == 0 {
			break
		}

		for _, chapter := range chaptersAPIResp.Chapters {
			chapterReturn, err := getChapterFromResp(chapter, chapter.Chap, mangaURL)
			if err != nil {
				errChan <- err
				return
			}
			chaptersChan <- chapterReturn
		}

		currentPage++
	}
}

// getChapterHID returns the HID of a chapter given its URL.
// URL should be like: https://comick.io/comic/jitsu-wa-watashi-wa/PZKrW
// or https://comick.io/comic/jitsu-wa-watashi-wa/PZKrW-chapter-121-en
func getChapterHID(chapterURL string) (string, error) {
	contextError := "error while getting chapter HID"

	parts := strings.Split(chapterURL, "/")
	hid := parts[len(parts)-1]

	parts = strings.Split(hid, "-")
	hid = parts[0]

	if hid == "" {
		return "", util.AddErrorContext(contextError, fmt.Errorf("HID not found in URL"))
	}

	return hid, nil
}

func getChapterFromResp(chapterResp chapterAPIResponse, chapter string, mangaURL string) (*manga.Chapter, error) {
	chapterReturn := &manga.Chapter{}
	var err error

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
	chapterReturn.UpdatedAt, err = util.GetRFC3339Datetime(chapterResp.CreatedAt)
	if err != nil {
		return chapterReturn, err
	}

	return chapterReturn, nil
}
