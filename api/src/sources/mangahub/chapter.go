package mangahub

import (
	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetChapterMetadata returns a chapter by its chapter, or else by its URL
func (s *Source) GetChapterMetadata(mangaURL, _, chapter, chapterURL, _ string) (*manga.Chapter, error) {
	errorContext := "error while getting metadata of chapter"

	if chapter == "" && chapterURL == "" {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrChapterHasNoChapterOrURL)
	}

	chapters, err := s.GetChaptersMetadata(mangaURL, "")
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	returnChapter, err := findChapter(chapters, chapter, chapterURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return returnChapter, nil
}

// GetLastChapterMetadata returns the latest chapter
func (s *Source) GetLastChapterMetadata(mangaURL, _ string) (*manga.Chapter, error) {
	errorContext := "error while getting last chapter metadata"

	chapters, err := s.GetChaptersMetadata(mangaURL, "")
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	if len(chapters) == 0 {
		return nil, util.AddErrorContext(errorContext, errNoChapters)
	}

	return chapters[0], nil
}

// GetChaptersMetadata returns the manga chapters, newest first
func (s *Source) GetChaptersMetadata(mangaURL, _ string) ([]*manga.Chapter, error) {
	errorContext := "error while getting chapters metadata"

	formattedURL, err := GetFormattedMangaURL(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	page, err := getMangaPage(formattedURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	return page.chapters, nil
}
