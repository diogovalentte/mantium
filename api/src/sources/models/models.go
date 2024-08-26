package models

import "github.com/diogovalentte/mantium/api/src/manga"

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
	// Search searches for a manga by its name.
	Search(term string) ([]*MangaSearchResult, error)
}

type MangaSearchResult struct {
	URL         string
	Name        string
	Source      string
	CoverURL    string
	Description string
	Status      string
	LastChapter string
	Year        int
}
