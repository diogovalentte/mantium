package mangahub

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

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

// GetChapterMetadataByChapter returns the chapter by its chapter
func (s *Source) GetChapterMetadataByChapter(mangaURL, _, chapter string) (*manga.Chapter, error) {

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return nil, err
	}

	// The API's chapter(...) query is refused without a per client encryption
	// handshake that is rate limited per IP, so the chapter is looked up in the
	// manga's chapter list instead.
	query := `
        {"query":"{manga(x:m01,slug:\"MANGA-SLUG\"){chapters{number,title,slug,date}}}"}
    `
	query = strings.ReplaceAll(query, "MANGA-SLUG", mangaSlug)
	payload := strings.NewReader(query)

	var mangaAPIResp getMangaAPIResponse
	_, err = mangahubClient.Request("POST", baseAPIURL, payload, &mangaAPIResp)
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, errordefs.ErrMangaNotFound
		}
		return nil, err
	}

	if len(mangaAPIResp.Errors) > 0 {
		switch mangaAPIResp.Errors[0].Message {
		case "Cannot read properties of undefined (reading 'mangaID')":
			return nil, errordefs.ErrMangaNotFound
		default:
			return nil, fmt.Errorf("error while getting chapter from response: %s", mangaAPIResp.Errors[0].Message)
		}
	}

	return findChapterInResponse(mangaAPIResp.Data.Manga.Chapters, chapter, mangaSlug)
}

// findChapterInResponse returns the chapter with the given number from a manga's chapter list
func findChapterInResponse(chapters []*getMangaAPIChapter, chapter, mangaSlug string) (*manga.Chapter, error) {
	number, err := strconv.ParseFloat(chapter, 64)
	if err != nil {
		return nil, errordefs.ErrChapterNotFound
	}

	for _, c := range chapters {
		if c.Number == number {
			return getChapterFromResponse(c, mangaSlug)
		}
	}

	return nil, errordefs.ErrChapterNotFound
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

// GetLastChapterMetadata returns the latest chapter
func (s *Source) GetLastChapterMetadata(mangaURL, _ string) (*manga.Chapter, error) {

	errorContext := "error while getting last chapter metadata of manga with URL '%s'"

	mangaSlug, err := getMangaSlug(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	query := `
        {"query":"{manga(x:m01,slug:\"MANGA-SLUG\"){latestChapter,chapters{number,title,slug,date}}}"}
    `
	query = strings.ReplaceAll(query, "MANGA-SLUG", mangaSlug)
	payload := strings.NewReader(query)

	var mangaAPIResp getMangaAPIResponse
	_, err = mangahubClient.Request("POST", baseAPIURL, payload, &mangaAPIResp)
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

	chapterReturn, err := findChapterInResponse(mangaAPIResp.Data.Manga.Chapters, strconv.FormatFloat(mangaAPIResp.Data.Manga.LastestChapter, 'f', -1, 64), mangaSlug)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, mangaURL), err)
	}

	return chapterReturn, nil
}

// GetChaptersMetadata returns the manga chapters
func (s *Source) GetChaptersMetadata(mangaURL, _ string) ([]*manga.Chapter, error) {

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
	_, err = mangahubClient.Request("POST", baseAPIURL, payload, &mangaAPIResp)
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
	updatedAt, err := util.GetRFC3339Datetime(chapter.Date)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, util.AddErrorContext(errordefs.ErrChapterAttributesNotFound.Message, err))
	}

	number := strconv.FormatFloat(chapter.Number, 'f', -1, 64)
	var slug string
	if number != "" {
		slug = "chapter-" + number
	} else {
		slug = chapter.Slug
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
