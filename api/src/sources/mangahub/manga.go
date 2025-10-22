package mangahub

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources/models"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL, _ string) (*manga.Manga, error) {
	s.checkClient()

	errorContext := "error while getting manga metadata"

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "mangahub"

	query := `
        {"query":"{manga(x:m01,slug:\"%s\"){title,image,latestChapter}}"}
    `
	query = fmt.Sprintf(query, mangaSlug)
	payload := strings.NewReader(query)

	var mangaAPIResp getMangaAPIResponse
	_, err = s.client.Request("POST", baseAPIURL, payload, &mangaAPIResp)
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
		}
		return nil, util.AddErrorContext(errorContext, err)
	}

	if len(mangaAPIResp.Errors) > 0 {
		switch mangaAPIResp.Errors[0].Message {
		case "Cannot read properties of undefined (reading 'mangaID')":
			return nil, errordefs.ErrMangaAttributesNotFound
		default:
			return nil, fmt.Errorf("error while getting chapter from response: %s", mangaAPIResp.Errors[0].Message)
		}
	}

	mangaReturn.Name = mangaAPIResp.Data.Manga.Title

	// Cover Image
	if mangaAPIResp.Data.Manga.Image != "" {
		coverImgURL := baseUploadsURL + "/" + mangaAPIResp.Data.Manga.Image
		coverImg, resized, err := util.GetImageFromURL(coverImgURL, 3, 1*time.Second)
		if err == nil {
			mangaReturn.CoverImgURL = coverImgURL
			mangaReturn.CoverImgResized = resized
			mangaReturn.CoverImg = coverImg
		}
	}

	// Last Released Chapter
	if mangaAPIResp.Data.Manga.LastestChapter != 0 {
		lastReleasedChapter, err := s.GetChapterMetadataByChapter(mangaURL, "", strconv.FormatFloat(mangaAPIResp.Data.Manga.LastestChapter, 'f', -1, 64))
		if err != nil {
			if !util.ErrorContains(err, errordefs.ErrChapterNotFound.Message) {
				return nil, util.AddErrorContext(errorContext, err)
			}
			mangaReturn.LastReadChapter = nil
		} else {
			lastReleasedChapter.Type = 1
			mangaReturn.LastReleasedChapter = lastReleasedChapter
		}
	}

	mangaReturn.URL, err = GetFormattedMangaURL(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return mangaReturn, nil
}

type getMangaAPIResponse struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Data struct {
		Manga struct {
			Title          string                `json:"title"`
			Image          string                `json:"image"`
			LastestChapter float64               `json:"latestChapter"`
			Chapters       []*getMangaAPIChapter `json:"chapters"`
			Slug           string                `json:"slug"`
		} `json:"manga"`
	} `json:"data"`
}

func (s *Source) Search(term string, limit int) ([]*models.MangaSearchResult, error) {
	s.checkClient()

	errorContext := "error while getting manga metadata"

	query := `
        {"query":"{search(x:m01,q:\"%s\",mod:POPULAR,limit:%d,offset:0,count:true){rows{title,slug,image,latestChapter,status},count}}"}
    `
	query = fmt.Sprintf(query, term, limit)
	payload := strings.NewReader(query)

	var searchAPIResp searchAPIResponse
	_, err := s.client.Request("POST", baseAPIURL, payload, &searchAPIResp)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	if len(searchAPIResp.Errors) > 0 {
		return nil, fmt.Errorf("error while getting manga from response: %s", searchAPIResp.Errors[0].Message)
	}

	mangaSearchResults := make([]*models.MangaSearchResult, 0, len(searchAPIResp.Data.Search.Rows))
	for _, row := range searchAPIResp.Data.Search.Rows {
		mangaSearchResult := &models.MangaSearchResult{
			Source:         "mangahub",
			URL:            baseSiteURL + "/manga/" + row.Slug,
			Name:           row.Title,
			Status:         row.Status,
			LastChapter:    strconv.FormatFloat(row.LastestChapter, 'f', -1, 64),
			LastChapterURL: fmt.Sprintf("%s/chapter/%s/chapter-%s", baseSiteURL, row.Slug, strconv.FormatFloat(row.LastestChapter, 'f', -1, 64)),
		}
		mangaSearchResult.CoverURL = baseUploadsURL + "/" + row.Image
		if row.Image != "" {
			mangaSearchResult.CoverURL = baseUploadsURL + "/" + row.Image
		} else {
			mangaSearchResult.CoverURL = models.DefaultCoverImgURL
		}

		mangaSearchResults = append(mangaSearchResults, mangaSearchResult)
	}

	return mangaSearchResults, nil
}

type searchAPIResponse struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Data struct {
		Search struct {
			Rows []struct {
				Title          string                `json:"title"`
				Image          string                `json:"image"`
				Slug           string                `json:"slug"`
				Status         string                `json:"status"`
				Chapters       []*getMangaAPIChapter `json:"chapters"`
				LastestChapter float64               `json:"latestChapter"`
			} `json:"rows"`
		} `json:"search"`
	} `json:"data"`
}

// getMangaSlug returns the slug of a manga given its URL.
// URL should be like: https://mangahub.io/manga/super-no-ura-de-yani-suu-hanashi
func getMangaSlug(mangaURL string) (string, error) {
	errorContext := "error while getting manga slug from URL"

	pattern := `/manga/([0-9a-zA-Z_-]+)(?:/.*)?$`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", util.AddErrorContext(errorContext, err)
	}

	matches := re.FindStringSubmatch(mangaURL)
	if len(matches) < 2 {
		return "", util.AddErrorContext(errorContext, fmt.Errorf("manga Slug not found in URL"))
	}

	return matches[1], nil
}

func GetFormattedMangaURL(mangaURL string) (string, error) {
	errorContext := "error while getting formatted manga URL '%s'"

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return "", util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	return fmt.Sprintf("%s/manga/%s", baseSiteURL, mangaSlug), nil
}
