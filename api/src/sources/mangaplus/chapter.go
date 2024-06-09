package mangaplus

import (
	"fmt"
	"strings"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

var chapterURLBase = "https://mangaplus.shueisha.co.jp/viewer/"

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(mangaURL string, chapter string, chapterURL string) (*manga.Chapter, error) {
	errorContext := "error while getting metadata of chapter with chapter '%s' and URL '%s', and manga URL '%s'"

	if chapter == "" && chapterURL == "" {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, chapter, chapterURL, mangaURL), errordefs.ErrChapterDoesntHaveChapterAndURL)
	}

	returnChapter := &manga.Chapter{}
	var err error
	if chapter != "" {
		returnChapter, err = s.GetChapterMetadataByChapter(mangaURL, chapter)
	}
	if chapterURL != "" && (err != nil || chapter == "") {
		returnChapter, err = s.GetChapterMetadataByURL(chapterURL)
	}

	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, chapter, chapterURL, mangaURL), err)
	}

	return returnChapter, nil
}

// GetChapterMetadataByURL returns a manga chapter by its URL.
func (s *Source) GetChapterMetadataByURL(_ string) (*manga.Chapter, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetChapterMetadataByChapter returns a manga chapter by its chapter.
// Chapter is expected to be a clean chapter. For example, in the site, a chapter can be like "# 025",
// but here it should be "25". Use the function cleanChapter to clean the chapter.
func (s *Source) GetChapterMetadataByChapter(mangaURL string, chapter string) (*manga.Chapter, error) {
	s.checkClient()

	mangaID, err := getMangaID(mangaURL)
	if err != nil {
		return nil, err
	}
	_, response, err := s.client.Request(fmt.Sprintf("%s/title_detailV3?title_id=%d", baseAPIURL, mangaID))
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, errordefs.ErrMangaNotFound
		}
		return nil, err
	}

	titleChapters := response.GetSuccess().GetTitleDetailView().GetChapters()
	chapters := getChaptersFromAPIList(titleChapters)
	if len(chapters) == 0 {
		return nil, errordefs.ErrChapterNotFound
	}

	for _, mangaChapter := range chapters {
		if mangaChapter.Chapter == chapter {
			return mangaChapter, nil
		}
	}

	return nil, errordefs.ErrChapterNotFound
}

func cleanChapter(chapter string) string {
	return strings.TrimLeft(strings.TrimSpace(strings.Replace(chapter, "#", "", 1)), "0")
}

// GetLastChapterMetadata returns the manga last released chapter
func (s *Source) GetLastChapterMetadata(mangaURL string) (*manga.Chapter, error) {
	s.checkClient()

	mangaID, err := getMangaID(mangaURL)
	if err != nil {
		return nil, err
	}
	_, response, err := s.client.Request(fmt.Sprintf("%s/title_detailV3?title_id=%d", baseAPIURL, mangaID))
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, errordefs.ErrMangaNotFound
		}
		return nil, err
	}

	titleChapters := response.GetSuccess().GetTitleDetailView().GetChapters()
	chapters := getChaptersFromAPIList(titleChapters)
	if len(chapters) == 0 {
		return nil, errordefs.ErrChapterNotFound
	}

	return chapters[len(chapters)-1], nil
}

// GetChaptersMetadata returns all the chapters of a manga
func (s *Source) GetChaptersMetadata(mangaURL string) ([]*manga.Chapter, error) {
	s.checkClient()

	mangaID, err := getMangaID(mangaURL)
	if err != nil {
		return nil, err
	}
	_, response, err := s.client.Request(fmt.Sprintf("%s/title_detailV3?title_id=%d", baseAPIURL, mangaID))
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, errordefs.ErrMangaNotFound
		}
		return nil, err
	}
	titleChapters := response.GetSuccess().GetTitleDetailView().GetChapters()
	return getChaptersFromAPIList(titleChapters), nil
}
