// Package sources implements the manga sources.
// It provides a way to get manga metadata and chapters from different sources.
// The sources should not be used directly, instead, the functions in this package should be used.
package sources

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources/comick"
	"github.com/diogovalentte/mantium/api/src/sources/jmanga"
	"github.com/diogovalentte/mantium/api/src/sources/klmanga"
	"github.com/diogovalentte/mantium/api/src/sources/mangadex"
	"github.com/diogovalentte/mantium/api/src/sources/mangahub"
	"github.com/diogovalentte/mantium/api/src/sources/mangaplus"
	"github.com/diogovalentte/mantium/api/src/sources/mangaupdates"
	"github.com/diogovalentte/mantium/api/src/sources/models"
	"github.com/diogovalentte/mantium/api/src/sources/rawkuma"
	"github.com/diogovalentte/mantium/api/src/util"
)

var sources = map[string]models.Source{
	"mangadex":     &mangadex.Source{},
	"comick":       &comick.Source{},
	"mangahub":     &mangahub.Source{},
	"mangaplus":    &mangaplus.Source{},
	"mangaupdates": &mangaupdates.Source{},
	"rawkuma":      &rawkuma.Source{},
	"klmanga":      &klmanga.Source{},
	"jmanga":       &jmanga.Source{},
}

// RegisterSource registers a new source
func RegisterSource(domain string, source models.Source) {
	sources[domain] = source
}

// DeleteSource deletes a source
func DeleteSource(domain string) {
	delete(sources, domain)
}

// GetSource returns a source
func GetSource(mangaURL string) (models.Source, error) {
	contextError := "error while getting source"

	source, err := urlToSource(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	value, ok := sources[source]
	if !ok {
		return nil, util.AddErrorContext(contextError, fmt.Errorf("source '%s' not found", source))
	}
	return value, nil
}

// GetSources returns all sources
func GetSources() map[string]models.Source {
	return sources
}

// GetMangaMetadata gets the metadata of a manga using a source
func GetMangaMetadata(mangaURL, internalID string) (*manga.Manga, error) {
	contextError := "error while getting metadata of manga with URL '%s' and internal ID '%s' from source"

	source, err := GetSource(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, internalID), err)
	}
	contextError = fmt.Sprintf("(%s) %s", source.GetName(), contextError)

	manga, err := getManga(mangaURL, internalID, source)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, internalID), err)
	}

	return manga, nil
}

// SearchManga searches for a manga using a source
func SearchManga(term, sourceSiteURL string, limit int) ([]*models.MangaSearchResult, error) {
	contextError := "error while searching '%s' in '%s'"

	source, err := GetSource(sourceSiteURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, term, sourceSiteURL), err)
	}
	contextError = fmt.Sprintf("(%s) %s", source.GetName(), contextError)

	results, err := searchManga(term, limit, source)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, term, sourceSiteURL), err)
	}

	return results, nil
}

// GetChapterMetadata gets the metadata of a chapter using a source.
// Each source has its own way to get the chapter. Some can't get the chapter by its URL/chapter,
// so they get the chapter by the chapter chapter/URL.
func GetChapterMetadata(mangaURL, mangaInternalID, chapter, chapterURL, chapterInternalID string) (*manga.Chapter, error) {
	contextError := "error while getting metadata of chapter with manga URL '%s', internal ID '%s', chapter '%s', chapter URL '%s', chapter internal ID '%s'"

	source, err := GetSource(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, mangaInternalID, chapter, chapterURL, chapterInternalID), err)
	}
	contextError = fmt.Sprintf("(%s) %s", source.GetName(), contextError)

	chapterReturn, err := getChapter(mangaURL, mangaInternalID, chapter, chapterURL, chapterInternalID, source)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, mangaInternalID, chapter, chapterURL, chapterInternalID), err)
	}

	return chapterReturn, nil
}

// GetMangaChapters gets the chapters of a manga using a source
func GetMangaChapters(mangaURL, mangaInternalID string) ([]*manga.Chapter, error) {
	contextError := "error while getting chapters from manga with URL '%s' and internal ID '%s' from source"

	source, err := GetSource(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, mangaInternalID), err)
	}
	contextError = fmt.Sprintf("(%s) %s", source.GetName(), contextError)

	chapters, err := getChapters(mangaURL, mangaInternalID, source)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, mangaInternalID), err)
	}

	return chapters, nil
}

func urlToSource(urlString string) (string, error) {
	errorContext := "error while getting source from URL '%s'"

	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", util.AddErrorContext(fmt.Sprintf(errorContext, urlString), err)
	}
	domain := parsedURL.Hostname()

	for source := range sources {
		if strings.Contains(domain, source) {
			return source, nil
		}
	}

	return "", util.AddErrorContext(fmt.Sprintf(errorContext, urlString), fmt.Errorf("source not found"))
}

func getManga(mangaURL, mangaInternalID string, source models.Source) (*manga.Manga, error) {
	return source.GetMangaMetadata(mangaURL, mangaInternalID)
}

func searchManga(term string, limit int, source models.Source) ([]*models.MangaSearchResult, error) {
	return source.Search(term, limit)
}

func getChapter(mangaURL, mangaInternalID, chapter, chapterURL, chapterInternalID string, source models.Source) (*manga.Chapter, error) {
	return source.GetChapterMetadata(mangaURL, mangaInternalID, chapter, chapterURL, chapterInternalID)
}

func getChapters(mangaURL, mangaInternalID string, source models.Source) ([]*manga.Chapter, error) {
	return source.GetChaptersMetadata(mangaURL, mangaInternalID)
}
