package models

import "github.com/diogovalentte/mantium/api/src/manga"

// Source is the interface for a manga source
type Source interface {
	// GetMangaMetadata returns a manga
	GetMangaMetadata(mangaURL, mangaInternalID string) (*manga.Manga, error)
	// GetChapterMetadata returns a chapter by its chapter or URL
	GetChapterMetadata(mangaURL, mangaInternalID, chapter, chapterURL, chapterInternalID string) (*manga.Chapter, error)
	// GetLastChapterMetadata returns the last released chapter in the source
	GetLastChapterMetadata(mangaURL, mangaInternalID string) (*manga.Chapter, error)
	// GetChaptersMetadata returns all chapters of a manga
	GetChaptersMetadata(mangaURL, mangaInternalID string) ([]*manga.Chapter, error)
	// Search searches for a manga by its name.
	Search(term string, limit int) ([]*MangaSearchResult, error)
}

type MangaSearchResult struct {
	URL            string
	Name           string
	Source         string
	CoverURL       string
	Description    string
	Status         string
	LastChapter    string
	LastChapterURL string
	InternalID     string
	Year           int
}

var DefaultCoverImgURL = "https://i.imgur.com/jMy7evE.jpeg"
