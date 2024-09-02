package comick

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

// GetMangaMetadata returns the metadata of a manga given its URL.
func (s *Source) GetMangaMetadata(mangaURL, _ string, ignoreGetLastChapterError bool) (*manga.Manga, error) {
	s.checkClient()

	errorContext := "error while getting manga metadata"

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "comick.xyz"
	mangaReturn.URL = mangaURL

	mangaID, err := getMangaSlug(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	mangaAPIURL := fmt.Sprintf("%s/comic/%s", baseAPIURL, mangaID)
	var mangaAPIResp getMangaAPIResponse
	_, err = s.client.Request("GET", mangaAPIURL, nil, &mangaAPIResp)
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, errordefs.ErrMangaNotFound
		}
		return nil, util.AddErrorContext(errorContext, err)
	}

	comic := &mangaAPIResp.Comic

	mangaReturn.Name = comic.Title

	lastReleasedChapter, err := s.GetLastChapterMetadata(mangaURL, "")
	if err != nil {
		if !(ignoreGetLastChapterError && util.ErrorContains(err, errordefs.ErrLastReleasedChapterNotFound.Message)) {
			return nil, util.AddErrorContext(errorContext, err)
		}
	} else {
		lastReleasedChapter.Type = 1
		mangaReturn.LastReleasedChapter = lastReleasedChapter
	}

	// Get cover img
	var coverFileName string
	for _, cover := range comic.MDCovers {
		if cover.B2Key != "" {
			coverFileName = cover.B2Key
			break
		}
	}
	if coverFileName != "" {
		coverURL := fmt.Sprintf("%s/%s", baseUploadsURL, coverFileName)
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

type getMangaAPIResponse struct {
	Comic comic `json:"comic"`
}

type comic struct {
	HID         string    `json:"hid"`
	Title       string    `json:"title"`
	Description string    `json:"desc"`
	MDCovers    []mdCover `json:"md_covers"`
	LastChapter float64   `json:"last_chapter"` // It seems to be the last english translated chapter released
	ID          int       `json:"id"`
	Year        int       `json:"year"`
	Status      int       `json:"status"`
}

type mdCover struct {
	B2Key string `json:"b2key"`
}

func (s *Source) Search(term string, limit int) ([]*models.MangaSearchResult, error) {
	s.checkClient()

	errorContext := "error while searching manga"

	term = strings.ReplaceAll(term, " ", "+")
	searchURL := fmt.Sprintf("%s/v1.0/search?q=%s&type=comic&page=1&limit=%d&sort=view&showall=true", baseAPIURL, term, limit)
	var searchAPIResp []*comic
	_, err := s.client.Request("GET", searchURL, nil, &searchAPIResp)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	mangaSearchResults := make([]*models.MangaSearchResult, 0, len(searchAPIResp))
	for _, comic := range searchAPIResp {
		mangaSearchResult := &models.MangaSearchResult{}
		mangaSearchResult.Source = "comick.xyz"
		mangaSearchResult.URL = fmt.Sprintf("%s/comic/%s", baseSiteURL, comic.HID)
		mangaSearchResult.Description = comic.Description
		mangaSearchResult.Year = comic.Year
		mangaSearchResult.Name = comic.Title
		mangaSearchResult.LastChapter = strconv.FormatFloat(comic.LastChapter, 'f', -1, 64)
		if mangaSearchResult.LastChapter == "0" || mangaSearchResult.LastChapter == "" {
			mangaSearchResult.LastChapter = "N/A"
		}
		mangaSearchResult.Status = getMangaStatus(comic.Status)
		if len(comic.MDCovers) == 0 {
			mangaSearchResult.CoverURL = models.DefaultCoverImgURL
		} else {
			mangaSearchResult.CoverURL = fmt.Sprintf("%s/%s", baseUploadsURL, comic.MDCovers[0].B2Key)
		}
		mangaSearchResults = append(mangaSearchResults, mangaSearchResult)
	}

	return mangaSearchResults, nil
}

// getMangaHID returns the HID of a manga given its URL.
// URL should be like: https://comick.xyz/comic/00-jujutsu-kaisen
func (s *Source) getMangaHID(mangaURL string) (string, error) {
	s.checkClient()

	errorContext := "error while getting manga HID"

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return "", util.AddErrorContext(errorContext, err)
	}

	mangaAPIURL := fmt.Sprintf("%s/comic/%s", baseAPIURL, mangaSlug)
	var mangaAPIResp getMangaAPIResponse
	_, err = s.client.Request("GET", mangaAPIURL, nil, &mangaAPIResp)
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return "", errordefs.ErrMangaNotFound
		}
		return "", util.AddErrorContext(errorContext, err)
	}

	return mangaAPIResp.Comic.HID, nil
}

// getMangaSlug returns the slug of a manga given its URL.
// URL should be like: https://comick.xyz/comic/00-jujutsu-kaisen
func getMangaSlug(mangaURL string) (string, error) {
	errorContext := "error while getting manga slug from URL"

	pattern := `^https?://comick\.[^/]+/comic/([^/]+)(?:/.*)?$`
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

func getMangaStatus(status int) string {
	switch status {
	case 1:
		return "Ongoing"
	case 2:
		return "Completed"
	case 3:
		return "Cancelled"
	case 4:
		return "Hiatus"
	default:
		return "Unknown"
	}
}
