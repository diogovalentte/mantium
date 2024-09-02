package mangaupdates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources/models"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata returns the metadata of a manga given its URL.
func (s *Source) GetMangaMetadata(mangaURL, mangaInternalID string, ignoreGetLastChapterError bool) (*manga.Manga, error) {
	s.checkClient()

	errorContext := "error while getting manga metadata"
	var err error
	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "mangaupdates"

	if mangaInternalID == "" {
		mangaInternalID, err = s.getMangaIDFromURL(mangaURL)
		if err != nil {
			return nil, util.AddErrorContext(errorContext, err)
		}
	}

	mangaAPIURL := fmt.Sprintf("%s/v1/series/%s", baseAPIURL, mangaInternalID)
	var mangaAPIResp seriesAPIResp
	_, err = s.client.Request("GET", mangaAPIURL, nil, &mangaAPIResp)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	mangaReturn.Name = mangaAPIResp.Title
	mangaReturn.URL = mangaAPIResp.URL
	mangaReturn.InternalID = strconv.Itoa(mangaAPIResp.ID)

	lastReleasedChapter, err := s.GetLastChapterMetadata(mangaURL, mangaInternalID)
	if err != nil {
		if !(ignoreGetLastChapterError && util.ErrorContains(err, errordefs.ErrLastReleasedChapterNotFound.Message)) {
			return nil, util.AddErrorContext(errorContext, err)
		}
	} else {
		lastReleasedChapter.Type = 1
		mangaReturn.LastReleasedChapter = lastReleasedChapter
	}

	coverURL, ok := mangaAPIResp.Image.URL["original"]
	if !ok {
		for _, url := range mangaAPIResp.Image.URL {
			coverURL = url
			break
		}
	}
	if coverURL != "" {
		mangaReturn.CoverImgURL = coverURL

		coverImg, resized, err := util.GetImageFromURL(coverURL, 3, 1*time.Second)
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

func (s *Source) Search(term string, limit int) ([]*models.MangaSearchResult, error) {
	s.checkClient()

	errorContext := "error while searching manga"

	body, payload := map[string]interface{}{
		"search":  term,
		"perpage": limit, // possible values: 5,10,15,25,30,40,50,75,100. 5 is the minimum, 100 the maximum, 25 the default
		"orderby": "score",
	}, new(bytes.Buffer)
	json.NewEncoder(payload).Encode(body)

	searchURL := fmt.Sprintf("%s/v1/series/search", baseAPIURL)
	var searchResp searchResultResponse
	_, err := s.client.Request("POST", searchURL, payload, &searchResp)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	results := make([]*models.MangaSearchResult, 0, len(searchResp.Results))
	for _, result := range searchResp.Results {
		year := 0
		if result.Record.Year != "" {
			year, _ = strconv.Atoi(result.Record.Year)
		}
		coverURL, ok := result.Record.Image.URL["original"]
		if !ok {
			for _, url := range result.Record.Image.URL {
				coverURL = url
				break
			}
		}
		results = append(results, &models.MangaSearchResult{
			InternalID:  strconv.Itoa(result.Record.ID),
			URL:         result.Record.URL,
			Name:        result.Record.Title,
			Source:      "mangaupdates",
			CoverURL:    coverURL,
			Description: result.Record.Description,
			Year:        year,
		})
	}

	return results, nil
}

type searchResultResponse struct {
	Results []struct {
		Record seriesAPIResp `json:"record"`
	} `json:"results"`
	TotalHits int `json:"total_hits"`
	Page      int `json:"page"`
	PerPage   int `json:"per_page"`
}

type seriesAPIResp struct {
	Image struct {
		URL map[string]string `json:"url"`
	} `json:"image"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Year        string `json:"year"`
	ID          int    `json:"series_id"`
}

func (s *Source) getMangaIDFromURL(mangaURL string) (string, error) {
	errorContext := "error while getting manga ID from URL"

	c := colly.NewCollector(
		colly.AllowURLRevisit(),
	)

	var sharedErr error
	var mangaID string
	c.OnHTML("a[href^='https://api.mangaupdates.com/v1/series']", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		href = strings.Split(href, "https://api.mangaupdates.com/v1/series/")[1]
		mangaID = strings.Split(href, "/")[0]
	})

	c.OnError(func(_ *colly.Response, err error) {
		sharedErr = err
	})

	err := c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return "", util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
		}
		return "", util.AddErrorContext(errorContext, err)
	}
	if sharedErr != nil {
		return "", util.AddErrorContext(errorContext, sharedErr)
	}
	if mangaID == "" {
		return "", util.AddErrorContext(errorContext, fmt.Errorf("manga ID not found in the page"))
	}

	return mangaID, nil
}
