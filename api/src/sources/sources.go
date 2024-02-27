// Package sources implements the manga sources.
// It provides a way to get manga metadata and chapters from different sources.
// The sources should not be used directly, instead, the functions in this package should be used.
package sources

import (
	"fmt"
	"net/url"

	"github.com/diogovalentte/manga-dashboard-api/api/src/manga"
	"github.com/diogovalentte/manga-dashboard-api/api/src/sources/comick"
	"github.com/diogovalentte/manga-dashboard-api/api/src/sources/mangadex"
	"github.com/diogovalentte/manga-dashboard-api/api/src/sources/mangahub"
	"github.com/diogovalentte/manga-dashboard-api/api/src/util"
)

// sources is a map of all sources
var sources = map[string]Source{
	// default sources
	"mangahub.io":  &mangahub.Source{},
	"mangadex.org": &mangadex.Source{},
	"comick.io":    &comick.Source{},
}

// Source is the interface for a manga source
type Source interface {
	// GetMangaMetadata returns a manga
	GetMangaMetadata(mangaURL string) (*manga.Manga, error)
	// GetChapterMetadataByURL returns a chapter by its URL
	GetChapterMetadataByURL(chapterURL string) (*manga.Chapter, error)
	// GetChapterMetadataByChapter returns a chapter by its chapter
	GetChapterMetadataByChapter(mangaURL string, chapter string) (*manga.Chapter, error)
	// GetChapterMetadata returns a chapter by its chapter or URL
	GetChapterMetadata(mangaURL string, chapter string, chapterURL string) (*manga.Chapter, error)
	// GetLastChapterMetadata returns the last uploaded chapter in the source
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
	value, ok := sources[domain]
	if !ok {
		return nil, fmt.Errorf("source %s not found", domain)
	}
	return value, nil
}

// GetSources returns all sources
func GetSources() map[string]Source {
	return sources
}

// GetMangaMetadata gets the metadata of a manga using a source
func GetMangaMetadata(mangaURL string) (*manga.Manga, error) {
	contextError := "error while getting manga metadata from source"

	domain, err := getDomain(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}

	source, err := GetSource(domain)
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}
	manga, err := getManga(mangaURL, source)
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}

	return manga, nil
}

// GetChapterMetadata gets the metadata of a chapter using a source
// If the chapterURL is not empty, it will use it to get the chapter
// If the chapterURL is empty, it will use the mangaURL and chapter to get the chapter
func GetChapterMetadata(mangaURL string, chapter string, chapterURL string) (*manga.Chapter, error) {
	contextError := "error while getting chapter metadata from source"

	domain, err := getDomain(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}

	source, err := GetSource(domain)
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}

	chapterReturn, err := getChapter(mangaURL, chapter, chapterURL, source)
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}

	return chapterReturn, nil
}

// GetMangaChapters gets the chapters of a manga using a source
func GetMangaChapters(mangaURL string) ([]*manga.Chapter, error) {
	contextError := "error while getting manga chapters metadata from source"

	domain, err := getDomain(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}

	source, err := GetSource(domain)
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}

	chapters, err := getChapters(mangaURL, source)
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}

	return chapters, nil
}

func getDomain(urlString string) (string, error) {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}

	return parsedURL.Hostname(), nil
}

func getManga(mangaURL string, source Source) (*manga.Manga, error) {
	return source.GetMangaMetadata(mangaURL)
}

func getChapter(mangaURL string, chapter string, chapterURL string, source Source) (*manga.Chapter, error) {
	return source.GetChapterMetadata(mangaURL, chapter, chapterURL)
}

func getChapters(mangaURL string, source Source) ([]*manga.Chapter, error) {
	return source.GetChaptersMetadata(mangaURL)
}
