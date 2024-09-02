package mangahub

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetChapterMetadata returns a chapter by its chapter or URL
func (s *Source) GetChapterMetadata(mangaURL, _, chapter, chapterURL, _ string) (*manga.Chapter, error) {
	errorContext := "error while getting metadata of chapter"

	if chapter == "" && chapterURL == "" {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrChapterDoesntHaveChapterAndURL)
	}

	returnChapter := &manga.Chapter{}
	var err error
	if chapter != "" {
		returnChapter, err = s.GetChapterMetadataByChapter(mangaURL, "", chapter)
	}
	if chapterURL != "" && (err != nil || chapter == "") {
		returnChapter, err = s.GetChapterMetadataByURL(chapterURL)
	}

	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return returnChapter, nil
}

// GetChapterMetadataByURL scrapes the manga page and return the chapter by its URL
func (s *Source) GetChapterMetadataByURL(_ string) (*manga.Chapter, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetChapterMetadataByChapter scrapes the manga page and return the chapter by its chapter
func (s *Source) GetChapterMetadataByChapter(mangaURL, _, chapter string) (*manga.Chapter, error) {
	s.checkClient()

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return nil, err
	}

	chapterReturn := &manga.Chapter{}

	query := `
        {"query":"{chapter(x:m01,slug:\"MANGA-SLUG\",number:CHAPTER-NUMBER){number,title,slug,date,manga{slug}}}"}
    `
	query = strings.ReplaceAll(query, "MANGA-SLUG", mangaSlug)
	query = strings.ReplaceAll(query, "CHAPTER-NUMBER", chapter)
	payload := strings.NewReader(query)

	var mangaAPIResp getChapterAPIResponse
	_, err = s.client.Request("POST", baseAPIURL, payload, &mangaAPIResp)
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, errordefs.ErrChapterNotFound
		}
		return nil, err
	}

	if len(mangaAPIResp.Errors) > 0 {
		switch mangaAPIResp.Errors[0].Message {
		case "Cannot read properties of undefined (reading 'mangaID')":
			return nil, errordefs.ErrMangaNotFound
		case "Cannot convert undefined or null to object":
			return nil, errordefs.ErrChapterNotFound
		default:
			return nil, fmt.Errorf("error while getting chapter from response: %s", mangaAPIResp.Errors[0].Message)
		}
	}

	chapterReturn, err = getChapterFromResponse(&mangaAPIResp.Data.Chapter, mangaSlug)
	if err != nil {
		return nil, err
	}

	return chapterReturn, nil
}

type getChapterAPIResponse struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Data struct {
		Chapter getMangaAPIChapter `json:"chapter"`
	} `json:"data"`
}

type getMangaAPIChapter struct {
	Number float64 `json:"number"`
	Title  string  `json:"title"`
	Slug   string  `json:"slug"`
	Date   string  `json:"date"`
	Manga  struct {
		Slug string `json:"slug"`
	} `json:"manga"`
}

// GetLastChapterMetadata scrapes the manga page and return the latest chapter
func (s *Source) GetLastChapterMetadata(mangaURL, _ string) (*manga.Chapter, error) {
	s.checkClient()

	errorContext := "error while getting last chapter metadata of manga with URL '%s'"

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	query := `
        {"query":"{manga(x:m01,slug:\"MANGA-SLUG\"){latestChapter}}"}
    `
	query = strings.ReplaceAll(query, "MANGA-SLUG", mangaSlug)
	payload := strings.NewReader(query)

	var mangaAPIResp getMangaAPIResponse
	_, err = s.client.Request("POST", baseAPIURL, payload, &mangaAPIResp)
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), errordefs.ErrMangaNotFound)
		}
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	if len(mangaAPIResp.Errors) > 0 {
		switch mangaAPIResp.Errors[0].Message {
		case "Cannot read properties of undefined (reading 'mangaID')":
			return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), errordefs.ErrMangaNotFound)
		default:
			return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), fmt.Errorf("error while getting chapter from response: %s", mangaAPIResp.Errors[0].Message))
		}
	}

	chapterReturn, err := s.GetChapterMetadataByChapter(mangaURL, "", strconv.FormatFloat(mangaAPIResp.Data.Manga.LastestChapter, 'f', -1, 64))
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	return chapterReturn, nil
}

// GetChaptersMetadata scrapes the manga page and return the chapters
func (s *Source) GetChaptersMetadata(mangaURL, _ string) ([]*manga.Chapter, error) {
	s.checkClient()

	errorContext := "error while getting chapters metadata"

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	query := `
        {"query":"{manga(x:m01,slug:\"MANGA-SLUG\"){chapters{number,title,date}}}"}
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

	chaptersLen := len(mangaAPIResp.Data.Manga.Chapters)
	chapters := make([]*manga.Chapter, 0, chaptersLen)
	for i := chaptersLen - 1; i >= 0; i-- {
		chapterReturn, err := getChapterFromResponse(mangaAPIResp.Data.Manga.Chapters[i], mangaSlug)
		if err != nil {
			return nil, util.AddErrorContext(errorContext, err)
		}
		chapters = append(chapters, chapterReturn)
	}

	return chapters, nil
}

func getChapterFromResponse(chapter *getMangaAPIChapter, mangaSlug string) (*manga.Chapter, error) {
	errorContext := "error while getting chapter from response"
	layout := time.RFC3339
	updatedAt, err := time.Parse(layout, chapter.Date)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	number := strconv.FormatFloat(chapter.Number, 'f', -1, 64)
	slug := chapter.Slug
	if slug == "" {
		slug = "chapter-" + number
	}
	title := chapter.Title
	if title == "" {
		// MangaHub uses the manga name + number when the chapter title is empty.
		// But we'll use this instead.
		title = "Chapter " + number
	}
	chapterReturn := &manga.Chapter{
		URL:       baseSiteURL + "/chapter/" + mangaSlug + "/" + slug,
		Chapter:   number,
		Name:      title,
		UpdatedAt: updatedAt,
	}

	return chapterReturn, nil
}
