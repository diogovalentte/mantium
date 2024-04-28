package comick

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata returns the metadata of a manga given its URL.
func (s *Source) GetMangaMetadata(mangaURL string) (*manga.Manga, error) {
	s.checkClient()

	errorContext := "Error while getting manga metadata"

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "comick.xyz"
	mangaReturn.URL = mangaURL

	mangaID, err := getMangaSlug(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	mangaAPIURL := fmt.Sprintf("%s/comic/%s", baseAPIURL, mangaID)
	resp, err := s.client.Request("GET", mangaAPIURL, nil)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}
	defer resp.Body.Close()

	var mangaAPIResp getMangaAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&mangaAPIResp); err != nil {
		return nil, util.AddErrorContext(err, util.AddErrorContext(fmt.Errorf("Error decoding JSON body response"), errorContext).Error())
	}

	comic := &mangaAPIResp.Comic

	mangaReturn.Name = comic.Title

	lastUploadChapter, err := s.GetLastChapterMetadata(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}
	lastUploadChapter.Type = 1
	mangaReturn.LastUploadChapter = lastUploadChapter

	// Get cover img
	var coverFileName string
	for _, cover := range comic.MDCovers {
		if cover.B2Key != "" {
			coverFileName = cover.B2Key
			break
		}
	}
	if coverFileName == "" {
		return nil, util.AddErrorContext(fmt.Errorf("Cover image not found"), errorContext)
	}
	coverURL := fmt.Sprintf("%s/%s", baseUploadsURL, coverFileName)
	mangaReturn.CoverImgURL = coverURL

	coverImg, resized, err := util.GetImageFromURL(coverURL, 3, 1*time.Second)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}
	mangaReturn.CoverImgResized = resized
	mangaReturn.CoverImg = coverImg

	return mangaReturn, nil
}

type getMangaAPIResponse struct {
	Comic comic `json:"comic"`
}

type comic struct {
	Title       string    `json:"title"`
	ID          int       `json:"id"`
	LastChapter float32   `json:"last_chapter"` // It seems to be the last english translated chapter uploaded
	MDCovers    []mdCover `json:"md_covers"`
	HID         string    `json:"hid"`
}

type mdCover struct {
	B2Key string `json:"b2key"`
}

// getMangaHID returns the HID of a manga given its URL.
// URL should be like: https://comick.xyz/comic/00-jujutsu-kaisen
func (s *Source) getMangaHID(mangaURL string) (string, error) {
	s.checkClient()

	errorContext := "Error while getting manga HID"

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "comick.xyz"
	mangaReturn.URL = mangaURL

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return "", util.AddErrorContext(err, errorContext)
	}

	mangaAPIURL := fmt.Sprintf("%s/comic/%s", baseAPIURL, mangaSlug)
	resp, err := s.client.Request("GET", mangaAPIURL, nil)
	if err != nil {
		return "", util.AddErrorContext(err, errorContext)
	}
	defer resp.Body.Close()

	var mangaAPIResp getMangaAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&mangaAPIResp); err != nil {
		return "", util.AddErrorContext(err, util.AddErrorContext(fmt.Errorf("Error decoding JSON body response"), errorContext).Error())
	}

	return mangaAPIResp.Comic.HID, nil
}

// getMangaSlug returns the slug of a manga given its URL.
// URL should be like: https://comick.xyz/comic/00-jujutsu-kaisen
func getMangaSlug(mangaURL string) (string, error) {
	errorContext := "Error while getting manga slug from URL"

	pattern := `^https?://comick\.[^/]+/comic/([^/]+)(?:/.*)?$`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", util.AddErrorContext(err, errorContext)
	}

	matches := re.FindStringSubmatch(mangaURL)
	if len(matches) < 2 {
		return "", util.AddErrorContext(fmt.Errorf("manga Slug not found in URL"), errorContext)
	}

	return matches[1], nil
}
