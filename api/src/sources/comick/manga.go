package comick

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata returns the metadata of a manga given its URL.
func (s *Source) GetMangaMetadata(mangaURL string) (*manga.Manga, error) {
	s.checkClient()

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "comick.xyz"
	mangaReturn.URL = mangaURL

	mangaID, err := getMangaSlug(mangaURL)
	if err != nil {
		return nil, err
	}

	mangaAPIURL := fmt.Sprintf("%s/comic/%s", baseAPIURL, mangaID)
	resp, err := s.client.Request("GET", mangaAPIURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mangaAPIResp getMangaAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&mangaAPIResp); err != nil {
		return nil, err
	}

	comic := &mangaAPIResp.Comic

	mangaReturn.Name = comic.Title

	lastUploadChapter, err := s.GetLastChapterMetadata(mangaURL)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("cover art not found")
	}
	coverURL := fmt.Sprintf("%s/%s", baseUploadsURL, coverFileName)
	mangaReturn.CoverImgURL = coverURL

	coverImg, err := util.GetImageFromURL(coverURL)
	if err != nil {
		return nil, err
	}
	resizedCoverImg, err := util.ResizeImage(coverImg, uint(util.DefaultImageWidth), uint(util.DefaultImageHeight))
	if err != nil {
		// JPEG format that has an unsupported subsampling ratio
		// It's a valid image but the standard library doesn't support it
		// And other libraries use the standard library under the hood
		if err.Error() == "unsupported JPEG feature: luma/chroma subsampling ratio" {
			resizedCoverImg = coverImg
		} else {
			err = fmt.Errorf("error resizing image: %s", err)
			return nil, err
		}
	} else {
		mangaReturn.CoverImgResized = true
	}
	mangaReturn.CoverImg = resizedCoverImg

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

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "comick.xyz"
	mangaReturn.URL = mangaURL

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return "", err
	}

	mangaAPIURL := fmt.Sprintf("%s/comic/%s", baseAPIURL, mangaSlug)
	resp, err := s.client.Request("GET", mangaAPIURL, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var mangaAPIResp getMangaAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&mangaAPIResp); err != nil {
		return "", err
	}

	return mangaAPIResp.Comic.HID, nil
}

// getMangaSlug returns the slug of a manga given its URL.
// URL should be like: https://comick.xyz/comic/00-jujutsu-kaisen
func getMangaSlug(mangaURL string) (string, error) {
	pattern := `^https?://comick\.[^/]+/comic/([^/]+)(?:/.*)?$`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	matches := re.FindStringSubmatch(mangaURL)
	if len(matches) < 2 {
		return "", fmt.Errorf("manga Slug not found in URL")
	}

	return matches[1], nil
}
