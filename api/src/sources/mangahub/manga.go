package mangahub

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL string, ignoreGetLastChapterError bool) (*manga.Manga, error) {
	s.checkClient()

	errorContext := "error while getting manga metadata"

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "mangahub.io"
	mangaReturn.URL = mangaURL

	query := `
        {"query":"{manga(x:m01,slug:\"MANGA-SLUG\"){title,image,latestChapter}}"}
    `
	query = strings.ReplaceAll(query, "MANGA-SLUG", mangaSlug)
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
			return nil, errordefs.ErrMangaNotFound
		default:
			return nil, fmt.Errorf("error while getting chapter from response: %s", mangaAPIResp.Errors[0].Message)
		}
	}

	mangaReturn.Name = mangaAPIResp.Data.Manga.Title

	// Cover Image
	if mangaAPIResp.Data.Manga.Image != "" {
		mangaReturn.CoverImgURL = baseUploadsURL + "/" + mangaAPIResp.Data.Manga.Image
		coverImg, resized, err := util.GetImageFromURL(mangaReturn.CoverImgURL, 3, 1*time.Second)
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

	// Last Release Chapter
	if mangaAPIResp.Data.Manga.LastestChapter != 0 {
		mangaReturn.LastReleasedChapter, err = s.GetChapterMetadataByChapter(mangaURL, strconv.FormatFloat(mangaAPIResp.Data.Manga.LastestChapter, 'f', -1, 64))
		if err != nil {
			return nil, util.AddErrorContext(errorContext, err)
		}
		mangaReturn.LastReleasedChapter.Type = 1
	} else if !ignoreGetLastChapterError {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrLastReleasedChapterNotFound)
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
		} `json:"manga"`
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
