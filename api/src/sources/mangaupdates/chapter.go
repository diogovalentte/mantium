package mangaupdates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(mangaURL, mangaInternalID, chapter, _, chapterInternalID string) (*manga.Chapter, error) {
	s.checkClient()

	errorContext := "error while getting metadata of chapter"
	var err error

	if mangaInternalID == "" {
		mangaInternalID, err = s.getMangaIDFromURL(mangaURL)
		if err != nil {
			return nil, util.AddErrorContext(errorContext, err)
		}
	}

	chaptersChan := make(chan *manga.Chapter)
	errChan := make(chan error)
	done := make(chan struct{})

	go s.generateMangaChapters(mangaInternalID, chaptersChan, errChan, done)

	var returnChapter *manga.Chapter
	if chapterInternalID != "" {
		go func() {
			for chap := range chaptersChan {
				if chap.InternalID == chapterInternalID {
					returnChapter = chap
					break
				}
			}
			close(done)
		}()
	} else if chapter != "" {
		go func() {
			for chap := range chaptersChan {
				if chap.Chapter == chapter {
					returnChapter = chap
					break
				}
			}
			close(done)
		}()
	} else {
		close(done)
		return nil, util.AddErrorContext(errorContext, err)
	}

	select {
	case <-done:
		if returnChapter == nil {
			return nil, util.AddErrorContext(errorContext, errordefs.ErrChapterNotFound)
		}
		return returnChapter, nil
	case err := <-errChan:
		return nil, util.AddErrorContext(errorContext, err)
	}
}

func (s *Source) GetLastChapterMetadata(mangaURL, mangaInternalID string) (*manga.Chapter, error) {
	s.checkClient()

	errorContext := "error while getting last chapter metadata"
	var err error

	if mangaInternalID == "" {
		mangaInternalID, err = s.getMangaIDFromURL(mangaURL)
		if err != nil {
			return nil, util.AddErrorContext(errorContext, err)
		}
	}

	chaptersChan := make(chan *manga.Chapter)
	errChan := make(chan error)
	done := make(chan struct{})

	go s.generateMangaChapters(mangaInternalID, chaptersChan, errChan, done)

	var returnChapter *manga.Chapter
	go func() {
		for chap := range chaptersChan {
			returnChapter = chap
			break
		}
		close(done)
	}()

	select {
	case <-done:
		if returnChapter == nil {
			return nil, util.AddErrorContext(errorContext, errordefs.ErrLastReleasedChapterNotFound)
		}
		return returnChapter, nil
	case err := <-errChan:
		return nil, util.AddErrorContext(errorContext, err)
	}
}

// GetChaptersMetadata returns the chapters of a manga
func (s *Source) GetChaptersMetadata(mangaURL, mangaInternalID string) ([]*manga.Chapter, error) {
	s.checkClient()

	errorContext := "error while getting chapters metadata"
	var err error

	if mangaInternalID == "" {
		mangaInternalID, err = s.getMangaIDFromURL(mangaURL)
		if err != nil {
			return nil, util.AddErrorContext(errorContext, err)
		}
	}

	chaptersChan := make(chan *manga.Chapter)
	errChan := make(chan error)
	done := make(chan struct{})

	go s.generateMangaChapters(mangaInternalID, chaptersChan, errChan, done)

	var chapters []*manga.Chapter
	go func() {
		for chapter := range chaptersChan {
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

// generateMangaChapters generates the chapters of a manga and sends them to the channel.
// It sends an error to the error channel if something goes wrong.
// It closes the chapters channel when there is no more chapters to send.
func (s *Source) generateMangaChapters(mangaInternalID string, chaptersChan chan *manga.Chapter, errChan chan error, done chan struct{}) {
	defer close(chaptersChan)

	releasesAPIURL := fmt.Sprintf("%s/v1/releases/search", baseAPIURL)
	generatedChaptersNumber := 0
	currentPage := 1
	for {
		select {
		case <-done:
			return
		default:
		}

		body, payload := map[string]interface{}{
			"search":      mangaInternalID,
			"search_type": "series",
			"orderby":     "date",
			"asc":         "desc",
			"perpage":     100, // possible values: 5,10,15,25,30,40,50,75,100. 5 is the minimum, 100 the maximum, 25 the default
			"page":        currentPage,
		}, new(bytes.Buffer)
		json.NewEncoder(payload).Encode(body)

		var chaptersAPIResp getChaptersAPIResponse
		_, err := s.client.Request("POST", releasesAPIURL, payload, &chaptersAPIResp)
		if err != nil {
			errChan <- err
			return
		}

		for _, record := range chaptersAPIResp.Results {
			chaptersChan <- getChapterFromResp(record.Record, mangaInternalID)
		}

		generatedChaptersNumber += len(chaptersAPIResp.Results)
		if generatedChaptersNumber >= chaptersAPIResp.TotalHits {
			break
		}

		currentPage++
	}
}

type getChaptersAPIResponse struct {
	Results []struct {
		Record releaseAPIResp `json:"record"`
	} `json:"results"`
	TotalHits int `json:"total_hits"`
	Page      int `json:"page"`
	PerPage   int `json:"per_page"`
}

type releaseAPIResp struct {
	Title       string `json:"title"`
	Chapter     string `json:"chapter"`
	ReleaseDate string `json:"release_date"`
	ID          int    `json:"id"`
}

func getChapterFromResp(release releaseAPIResp, mangaInternalID string) *manga.Chapter {
	var releaseDate time.Time
	var err error
	if release.ReleaseDate != "" {
		releaseDate, err = time.Parse("2006-01-02", release.ReleaseDate)
		if err == nil {
			releaseDate = releaseDate.In(time.Local)
		}
	}
	url := fmt.Sprintf("https://www.mangaupdates.com/releases.html?stype=series&search=%s", mangaInternalID)

	chapter := &manga.Chapter{
		Name:       release.Title,
		Chapter:    release.Chapter,
		UpdatedAt:  releaseDate,
		URL:        url,
		InternalID: strconv.Itoa(release.ID),
	}

	return chapter
}
