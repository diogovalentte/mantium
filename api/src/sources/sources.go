// Package sources implements the manga sources.
// It provides a way to get manga metadata and chapters from different sources.
// The sources should not be used directly, instead, the functions in this package should be used.
package sources

import (
	"fmt"
	"net/url"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources/comick"
	"github.com/diogovalentte/mantium/api/src/sources/mangadex"
	"github.com/diogovalentte/mantium/api/src/sources/mangahub"
	"github.com/diogovalentte/mantium/api/src/sources/mangaplus"
	"github.com/diogovalentte/mantium/api/src/util"
)

// sources is a map of all sources
var sources = map[string]Source{
	// default sources
	"mangahub.io":              &mangahub.Source{},
	"mangadex.org":             &mangadex.Source{},
	"comick.io":                &comick.Source{},
	"mangaplus.shueisha.co.jp": &mangaplus.Source{},
}

// Source is the interface for a manga source
type Source interface {
	// GetMangaMetadata returns a manga
	// ignoreGetLastChapterError is used to ignore the error when getting the last chapter of a manga by setting the last released chapter to nil. Use for mangas that don't have chapters.
	GetMangaMetadata(mangaURL string, ignoreGetLastChapterError bool) (*manga.Manga, error)
	// GetChapterMetadata returns a chapter by its chapter or URL
	GetChapterMetadata(mangaURL string, chapter string, chapterURL string) (*manga.Chapter, error)
	// GetLastChapterMetadata returns the last released chapter in the source
	GetLastChapterMetadata(mangaURL string) (*manga.Chapter, error)
	// GetChaptersMetadata returns all chapters of a manga
	GetChaptersMetadata(mangaURL string) ([]*manga.Chapter, error)
}

// RegisterSource registers a new source
func RegisterSource(domain string, source Source) {
	sources[domain] = source
}

// DeleteSource deletes a source
func DeleteSource(domain string) {
	delete(sources, domain)
}

// GetSource returns a source
func GetSource(domain string) (Source, error) {
	contextError := "error while getting source"

	value, ok := sources[domain]
	if !ok {
		return nil, util.AddErrorContext(contextError, fmt.Errorf("source '%s' not found", domain))
	}
	return value, nil
}

// GetSources returns all sources
func GetSources() map[string]Source {
	return sources
}

// GetMangaMetadata gets the metadata of a manga using a source
func GetMangaMetadata(mangaURL string, ignoreGetLastChapterError bool) (*manga.Manga, error) {
	contextError := "error while getting metadata of manga with URL '%s' from source"

	domain, err := getDomain(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL), err)
	}

	source, err := GetSource(domain)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL), err)
	}
	contextError = fmt.Sprintf("(%s) %s", domain, contextError)

	manga, err := getManga(mangaURL, source, ignoreGetLastChapterError)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL), err)
	}

	return manga, nil
}

// GetChapterMetadata gets the metadata of a chapter using a source.
// Each source has its own way to get the chapter. Some can't get the chapter by its URL/chapter,
// so they get the chapter by the chapter chapter/URL.
func GetChapterMetadata(mangaURL string, chapter string, chapterURL string) (*manga.Chapter, error) {
	contextError := "error while getting metadata of chapter with chapter '%s' and URL '%s' for manga with URL '%s' from source"

	domain, err := getDomain(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, chapter, chapterURL, mangaURL), err)
	}

	source, err := GetSource(domain)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, chapter, chapterURL, mangaURL), err)
	}
	contextError = fmt.Sprintf("(%s) %s", domain, contextError)

	chapterReturn, err := getChapter(mangaURL, chapter, chapterURL, source)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, chapter, chapterURL, mangaURL), err)
	}

	return chapterReturn, nil
}

// GetMangaChapters gets the chapters of a manga using a source
func GetMangaChapters(mangaURL string) ([]*manga.Chapter, error) {
	contextError := "error while getting manga with URL '%s' chapters from source"

	domain, err := getDomain(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL), err)
	}

	source, err := GetSource(domain)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL), err)
	}
	contextError = fmt.Sprintf("(%s) %s", domain, contextError)

	chapters, err := getChapters(mangaURL, source)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL), err)
	}

	return chapters, nil
}

func getDomain(urlString string) (string, error) {
	errorContext := "error while getting domain from URL '%s'"

	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", util.AddErrorContext(fmt.Sprintf(errorContext, urlString), err)
	}

	return parsedURL.Hostname(), nil
}

func getManga(mangaURL string, source Source, ignoreGetLastChapterError bool) (*manga.Manga, error) {
	return source.GetMangaMetadata(mangaURL, ignoreGetLastChapterError)
}

func getChapter(mangaURL string, chapter string, chapterURL string, source Source) (*manga.Chapter, error) {
	return source.GetChapterMetadata(mangaURL, chapter, chapterURL)
}

func getChapters(mangaURL string, source Source) ([]*manga.Chapter, error) {
	return source.GetChaptersMetadata(mangaURL)
}
