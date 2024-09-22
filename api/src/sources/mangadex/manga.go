package mangadex

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources/models"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata returns the metadata of a manga given its URL
func (s *Source) GetMangaMetadata(mangaURL, _ string) (*manga.Manga, error) {
	s.checkClient()

	errorContext := "error while getting manga metadata"

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "mangadex.org"
	mangaReturn.URL = mangaURL

	mangadexMangaID, err := getMangaID(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	mangaAPIURL := fmt.Sprintf("%s/manga/%s?includes[]=cover_art", baseAPIURL, mangadexMangaID)
	var mangaAPIResp getMangaAPIResponse
	_, err = s.client.Request("GET", mangaAPIURL, nil, &mangaAPIResp)
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
		}
		return nil, util.AddErrorContext(errorContext, err)
	}

	attributes := &mangaAPIResp.Data.Attributes

	mangaReturn.Name = attributes.Title["en"]
	if mangaReturn.Name == "" {
		mangaReturn.Name = attributes.Title["ja"]
		if mangaReturn.Name == "" {
			mangaReturn.Name = attributes.Title["ja-ro"]
			if mangaReturn.Name == "" {
				for _, title := range attributes.Title {
					mangaReturn.Name = title
					break
				}
			}
		}
	}

	lastReleasedChapter, err := s.GetLastChapterMetadata(mangaURL, "")
	if err != nil {
		if !util.ErrorContains(err, errordefs.ErrLastReleasedChapterNotFound.Message) {
			return nil, util.AddErrorContext(errorContext, err)
		}
		mangaReturn.LastReadChapter = nil
	} else {
		lastReleasedChapter.Type = 1
		mangaReturn.LastReleasedChapter = lastReleasedChapter
	}

	// Get cover img
	var coverFileName string
	for _, relationship := range mangaAPIResp.Data.Relationships {
		if relationship.Type == "cover_art" {
			attCoverFileName, ok := relationship.Attributes["fileName"]
			if ok {
				coverFileName = attCoverFileName.(string)
				break
			}
		}
	}
	if coverFileName != "" {
		coverURL := fmt.Sprintf("%s/covers/%s/%s", baseUploadsURL, mangadexMangaID, coverFileName)
		coverImg, resized, err := util.GetImageFromURL(coverURL, 3, 1*time.Second)
		if err == nil {
			mangaReturn.CoverImgURL = coverURL
			mangaReturn.CoverImgResized = resized
			mangaReturn.CoverImg = coverImg
		}
	}

	return mangaReturn, nil
}

type getMangaAPIResponse struct {
	Result   string `json:"result"`
	Response string `json:"response"`
	Data     struct {
		ID            string                `json:"id"`
		Type          string                `json:"type"`
		Relationships []genericRelationship `json:"relationships"`
		Attributes    mangaAttributes       `json:"attributes"`
	}
}

func (s *Source) Search(term string, limit int) ([]*models.MangaSearchResult, error) {
	s.checkClient()

	errorContext := "error while searching manga"

	term = strings.ReplaceAll(term, " ", "+")
	searchURL := fmt.Sprintf("%s/manga?title=%s&includes[]=cover_art&limit=%d&offset=0&order[relevance]=desc&contentRating[]=safe&contentRating[]=suggestive&contentRating[]=erotica&contentRating[]=pornographic", baseAPIURL, term, limit)
	var searchAPIResp searchMangaAPIResponse
	_, err := s.client.Request("GET", searchURL, nil, &searchAPIResp)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	mangaSearchResults := make([]*models.MangaSearchResult, 0, len(searchAPIResp.Data))
	for _, mangaData := range searchAPIResp.Data {
		mangaSearchResult := &models.MangaSearchResult{}
		mangaSearchResult.Source = "mangadex.org"
		mangaSearchResult.URL = fmt.Sprintf("%s/title/%s", baseSiteURL, mangaData.ID)
		mangaSearchResult.Description = mangaData.Attributes.Description.get()
		mangaSearchResult.Status = mangaData.Attributes.Status
		mangaSearchResult.Year = mangaData.Attributes.Year
		mangaSearchResult.LastChapter = mangaData.Attributes.LastChapter
		if mangaSearchResult.LastChapter == "0" || mangaSearchResult.LastChapter == "" {
			mangaSearchResult.LastChapter = "N/A"
		}

		mangaSearchResult.Name = mangaData.Attributes.Title.get()
		if mangaSearchResult.Name == "" {
			if len(mangaData.Attributes.AltTitles) > 0 {
				mangaSearchResult.Name = mangaData.Attributes.AltTitles[0].get()
			} else {
				return nil, util.AddErrorContext(errorContext, fmt.Errorf("manga name not found"))
			}
		}

		var coverFileName string
		for _, relationship := range mangaData.Relationships {
			if relationship.Type == "cover_art" {
				attCoverFileName, ok := relationship.Attributes["fileName"]
				if ok {
					coverFileName = attCoverFileName.(string)
					break
				}
			}
		}
		if coverFileName != "" {
			coverURL := fmt.Sprintf("%s/covers/%s/%s", baseUploadsURL, mangaData.ID, coverFileName)
			mangaSearchResult.CoverURL = coverURL
		} else {
			mangaSearchResult.CoverURL = models.DefaultCoverImgURL
		}
		mangaSearchResults = append(mangaSearchResults, mangaSearchResult)
	}

	return mangaSearchResults, nil
}

type searchMangaAPIResponse struct {
	Result   string `json:"result"`
	Response string `json:"response"`
	Data     []*struct {
		ID            string                `json:"id"`
		Type          string                `json:"type"`
		Relationships []genericRelationship `json:"relationships"`
		Attributes    mangaAttributes       `json:"attributes"`
	}
	Limit  int
	Offset int
	Total  int
}

// getMangaID returns the ID of a manga given its URL
// URL should be like: https://mangadex.org/title/87ebd557-8394-4f16-8afe-a8644e555ddc/hirayasumi
func getMangaID(mangaURL string) (string, error) {
	errorContext := "error while getting manga ID from URL"

	pattern := `/title/([0-9a-fA-F-]+)(?:/.*)?$`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", util.AddErrorContext(errorContext, err)
	}

	matches := re.FindStringSubmatch(mangaURL)
	if len(matches) < 2 {
		return "", util.AddErrorContext(errorContext, fmt.Errorf("manga ID not found"))
	}

	return matches[1], nil
}
