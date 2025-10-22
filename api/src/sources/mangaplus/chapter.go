package mangaplus

import (
	"fmt"
	"strings"

	"github.com/mendoncart/mantium/api/src/errordefs"
	"github.com/mendoncart/mantium/api/src/manga"
	"github.com/mendoncart/mantium/api/src/util"
)

var chapterURLBase = "https://mangaplus.shueisha.co.jp/viewer/"

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(mangaURL, _, chapter, _, _ string) (*manga.Chapter, error) {
	errorContext := "error while getting metadata of chapter"

	if chapter == "" {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrChapterHasNoChapterOrURL)
	}

	returnChapter, err := s.GetChapterMetadataByChapter(mangaURL, "", chapter)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return returnChapter, nil
}

// GetChapterMetadataByChapter returns a manga chapter by its chapter.
// Chapter is expected to be a clean chapter. For example, in the site, a chapter can be like "# 025",
// but here it should be "25". Use the function cleanChapter to clean the chapter.
func (s *Source) GetChapterMetadataByChapter(mangaURL, _, chapter string) (*manga.Chapter, error) {
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

// GetLastChapterMetadata returns the manga last released chapter
func (s *Source) GetLastChapterMetadata(mangaURL, _ string) (*manga.Chapter, error) {
	s.checkClient()

	errorContext := "error while getting last chapter metadata of manga with URL '%s'"

	mangaID, err := getMangaID(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}
	_, response, err := s.client.Request(fmt.Sprintf("%s/title_detailV3?title_id=%d", baseAPIURL, mangaID))
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), errordefs.ErrMangaNotFound)
		}
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	titleChapters := response.GetSuccess().GetTitleDetailView().GetChapters()
	chapters := getChaptersFromAPIList(titleChapters)
	if len(chapters) == 0 {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), errordefs.ErrChapterNotFound)
	}

	return chapters[0], nil
}

// GetChaptersMetadata returns all the chapters of a manga
func (s *Source) GetChaptersMetadata(mangaURL, _ string) ([]*manga.Chapter, error) {
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

func cleanChapter(chapter string) string {
	return strings.TrimLeft(strings.TrimSpace(strings.Replace(chapter, "#", "", 1)), "0")
}
