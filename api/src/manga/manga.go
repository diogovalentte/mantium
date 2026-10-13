// Package manga implements the manga and chapter structs and methods
package manga

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"github.com/antchfx/xmlquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/util"
)

type (
	// ID is the ID of the manga/multimanga in the database, should not be manually set
	ID int
	// Status: user's status of the manga/multimanga, it can be:
	// 1 - Reading
	// 2 - Completed
	// 3 - On Hold
	// 4 - Dropped
	// 5 - Plan to Read
	Status int
)

const (
	// CustomMangaSource is the source of custom mangas.
	CustomMangaSource = "custom_manga"
	// UnknownChapter is the chapter of custom mangas tracked only by the chapter URL.
	UnknownChapter = "?"
	// CustomMangaURLPrefix is the prefix of the URL of custom mangas and its chapters when the URL is not provided by the user.
	CustomMangaURLPrefix = "http://custom_manga.com"
)

func (id ID) String() string {
	return fmt.Sprintf("%d", id)
}

// Manga is the interface for a manga
type Manga struct {
	// Source is the source of the manga, usually the domain of the website.
	// If source is the above CustomMangaSource const, it means the manga is a custom manga created by the user.
	// and without a source site.
	Source string
	// URL is the URL of the manga.
	// If custom manga doesn't have a URL provided by the user, it should be like above CustomMangaSource/<uuid>.
	URL string
	// Name is the name of the manga
	Name string
	// SearchNames should be the multimanga's mangas names.
	// Used for searching mangas by name.
	SearchNames []string
	// InteralID is a unique identifier for the manga in the source
	InternalID string
	// PreferredGroup is the preferred group that translates (and more) the manga.
	// Not all sources have multiple groups. Currently not used.
	PreferredGroup string
	// CoverImgURL is the URL of the cover image
	CoverImgURL string
	// LastReleasedChapter is the last chapter released by the source
	// If the custom manga has no more released chapter, it'll be equal to the LastReadChapter.
	LastReleasedChapter *Chapter
	// LastReleasedChapterNameSelector is the selector used to find the last released chapter name in the source website
	LastReleasedChapterNameSelector *HTMLSelector
	// LastReleasedChapterURLSelector is the selector used to find the last released chapter URL in the source website
	LastReleasedChapterURLSelector *HTMLSelector
	// LastReadChapter is the last chapter read by the user
	// In a custom manga, this field represents the next manga the user should read
	// or, if it's equal to the last released chapter, the manga is considered read.
	LastReadChapter *Chapter
	// CoverImg is the cover image of the manga
	CoverImg []byte
	ID       ID
	Status   Status
	// When the manga is part of a multimanga, this field should be set to the multimanga ID
	MultiMangaID ID
	// CoverImgResized is true if the cover image was resized
	CoverImgResized bool
	// CoverImgFixed is true if the cover image is fixed. If true, the cover image will not be updated when updating the manga metadata.
	// It's used for when the cover image is manually set by the user.
	CoverImgFixed bool
	// LastReleasedChapterSelectorUseBrowser is true if the LastReleasedChapterNameSelector and LastReleasedChapterURLSelector should be used with a browser (Rod).
	LastReleasedChapterSelectorUseBrowser bool
}

func (m Manga) String() string {
	return fmt.Sprintf("Manga{ID: %d, Source: %s, URL: %s, Name: %s, SearchNames: %v, InternalID: %s, Status: %d, CoverImg: []byte, CoverImgResized: %v, CoverImgURL: %s, CoverImgFixed: %v, PreferredGroup: %s, MultiMangaID: %d, LastReleasedChapter: %s, LastReadChapter: %s, LastReleasedChapterNameSelector: %s, LastReleasedChapterURLSelector: %s, LastReleasedChapterSelectorUseBrowser: %v}",
		m.ID, m.Source, m.URL, m.Name, m.SearchNames, m.InternalID, m.Status, m.CoverImgResized, m.CoverImgURL, m.CoverImgFixed, m.PreferredGroup, m.MultiMangaID, m.LastReleasedChapter, m.LastReadChapter, m.LastReleasedChapterNameSelector, m.LastReleasedChapterURLSelector, m.LastReleasedChapterSelectorUseBrowser)
}

func insertMangaIntoDB(m *Manga, tx *sql.Tx) (ID, error) {
	err := validateManga(m)
	if err != nil {
		return -1, err
	}

	var multiMangaID sql.NullInt32
	if m.MultiMangaID > 0 {
		multiMangaID = sql.NullInt32{Int32: int32(m.MultiMangaID), Valid: true}
	} else {
		multiMangaID = sql.NullInt32{Valid: false}
	}

	if m.LastReleasedChapterNameSelector == nil {
		m.LastReleasedChapterNameSelector = &HTMLSelector{}
	}
	if m.LastReleasedChapterURLSelector == nil {
		m.LastReleasedChapterURLSelector = &HTMLSelector{}
	}

	var mangaID ID
	err = tx.QueryRow(`
        INSERT INTO mangas
            (source, url, name, internal_id, status, cover_img, cover_img_resized, cover_img_url, cover_img_fixed, preferred_group, multimanga_id, last_released_chapter_name_selector, last_released_chapter_name_attribute, last_released_chapter_name_regex, last_released_chapter_name_get_first, last_released_chapter_url_selector, last_released_chapter_url_attribute, last_released_chapter_url_get_first, last_released_chapter_selector_use_browser)
        VALUES
            ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
        RETURNING
            id;
    `, m.Source, m.URL, m.Name, m.InternalID, m.Status, m.CoverImg, m.CoverImgResized, m.CoverImgURL, m.CoverImgFixed, m.PreferredGroup, multiMangaID, m.LastReleasedChapterNameSelector.Selector, m.LastReleasedChapterNameSelector.Attribute, m.LastReleasedChapterNameSelector.Regex, m.LastReleasedChapterNameSelector.GetFirst, m.LastReleasedChapterURLSelector.Selector, m.LastReleasedChapterURLSelector.Attribute, m.LastReleasedChapterURLSelector.GetFirst, m.LastReleasedChapterSelectorUseBrowser).Scan(&mangaID)
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "mangas_pkey"` {
			return -1, errordefs.ErrMangaAlreadyInDB
		}
		return -1, err
	}

	if m.LastReleasedChapter != nil {
		err := upsertMangaChapter(mangaID, m.LastReleasedChapter, tx)
		if err != nil {
			if util.ErrorContains(err, `pq: duplicate key value violates unique constraint "chapters_pkey"`) {
				return -1, errordefs.ErrMangaAlreadyInDB // trying to add the same manga with different URL's (like klmanga.rs and klmanga.is), but they have the same chapter
			}
			return -1, err
		}
	}
	if m.LastReadChapter != nil {
		err := upsertMangaChapter(mangaID, m.LastReadChapter, tx)
		if err != nil {
			if util.ErrorContains(err, `pq: duplicate key value violates unique constraint "chapters_pkey"`) {
				return -1, errordefs.ErrMangaAlreadyInDB
			}
			return -1, err
		}
	}

	return mangaID, nil
}

// Will delete chapters too if the manga has chapters
// because of the foreign key constraint ON DELETE CASCADE
func deleteMangaDB(m *Manga, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	var result sql.Result
	if m.ID > 0 {
		result, err = tx.Exec(`
            DELETE FROM mangas
            WHERE id = $1;
        `, m.ID)
		if err != nil {
			return err
		}
	} else if m.URL != "" {
		result, err = tx.Exec(`
            DELETE FROM mangas
            WHERE url = $1;
        `, m.URL)
		if err != nil {
			return err
		}
	} else {
		return errordefs.ErrMangaHasNoIDOrURL
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMangaNotFoundDB
	}

	return nil
}

// UpsertChapterIntoDB updates the last read/released chapter in the database
// The chapter.Type field must be set
func (m *Manga) UpsertChapterIntoDB(chapter *Chapter) error {
	contextError := "error upserting chapter '%s' to manga '%s' into DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, m), err)
	}

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, m), err)
	}

	err = upsertMangaChapter(m.ID, chapter, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, m), err)
	}
	if chapter.Type == 1 {
		m.LastReleasedChapter = chapter
	} else {
		m.LastReadChapter = chapter
	}

	return nil
}

// UpdateNameInDB updates the manga name in the database
func (m *Manga) UpdateNameInDB(name string) error {
	contextError := "error updating manga '%s' name to '%s' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, name), err)
	}

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, name), err)
	}

	err = updateMangaNameDB(m, name, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, name), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, name), err)
	}
	m.Name = name
	m.SearchNames = []string{name}

	return nil
}

func updateMangaNameDB(m *Manga, name string, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	var result sql.Result
	if m.ID > 0 {
		result, err = tx.Exec(`
            UPDATE mangas
            SET name = $1
            WHERE id = $2;
        `, name, m.ID)
		if err != nil {
			return err
		}
	} else if m.URL != "" {
		result, err = tx.Exec(`
            UPDATE mangas
            SET name = $1
            WHERE url = $2;
        `, name, m.URL)
		if err != nil {
			return err
		}
	} else {
		return errordefs.ErrMangaHasNoIDOrURL
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMangaNotFoundDB
	}

	return nil
}

// UpdateURLInDB updates the manga URL in the database
func (m *Manga) UpdateURLInDB(URL string) error {
	contextError := "error updating manga '%s' URL to '%s' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, URL), err)
	}

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, URL), err)
	}

	err = updateMangaURLDB(m, URL, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, URL), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, URL), err)
	}
	m.URL = URL

	return nil
}

// UpdateCustomMangaURLInDB updates the manga URL in the database
func (m *Manga) UpdateCustomMangaURLInDB(URL string) error {
	contextError := "error updating custom manga '%s' URL to '%s' in DB"

	var chapter *Chapter
	var err error

	if (m.LastReleasedChapterNameSelector != nil && m.LastReleasedChapterNameSelector.Selector != "") || (m.LastReleasedChapterURLSelector != nil && m.LastReleasedChapterURLSelector.Selector != "") {
		chapter, err = GetCustomMangaLastReleasedChapter(URL, m.LastReleasedChapterNameSelector, m.LastReleasedChapterURLSelector, m.LastReleasedChapterSelectorUseBrowser)
		if err != nil {
			return util.AddErrorContext(fmt.Sprintf(contextError, m, URL), err)
		}
	}

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, URL), err)
	}

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, URL), err)
	}

	if chapter != nil {
		err = upsertMangaChapter(m.ID, chapter, tx)
		if err != nil {
			tx.Rollback()
			return util.AddErrorContext(fmt.Sprintf(contextError, m, URL), err)
		}
	}

	err = updateMangaURLDB(m, URL, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, URL), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, URL), err)
	}
	m.URL = URL

	return nil
}

func updateMangaURLDB(m *Manga, URL string, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	var result sql.Result
	if m.ID > 0 {
		result, err = tx.Exec(`
            UPDATE mangas
            SET url = $1
            WHERE id = $2;
        `, URL, m.ID)
		if err != nil {
			return err
		}
	} else if m.URL != "" {
		result, err = tx.Exec(`
            UPDATE mangas
            SET url = $1
            WHERE url = $2;
        `, URL, m.URL)
		if err != nil {
			return err
		}
	} else {
		return errordefs.ErrMangaHasNoIDOrURL
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMangaNotFoundDB
	}

	return nil
}

// UpdateCoverImgInDB updates the manga cover image in the database.
// It doesn't care if the cover image is fixed or not.
func (m *Manga) UpdateCoverImgInDB(coverImg []byte, coverImgResized bool, coverImgURL string) error {
	contextError := "error updating manga '%s' cover image to URL '%s' or/and image with '%d' bytes in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, coverImgURL, len(coverImg)), err)
	}

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, coverImgURL, len(coverImg)), err)
	}

	err = updateMangaCoverImg(m, coverImg, coverImgResized, coverImgURL, m.CoverImgFixed, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, coverImgURL, len(coverImg)), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, coverImgURL, len(coverImg)), err)
	}
	m.CoverImg = coverImg
	m.CoverImgResized = coverImgResized
	m.CoverImgURL = coverImgURL

	return nil
}

func updateMangaCoverImg(m *Manga, coverImg []byte, coverImgResized bool, coverImgURL string, fixed bool, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	var result sql.Result
	if m.ID > 0 {
		result, err = tx.Exec(`
            UPDATE mangas
            SET cover_img = $1, cover_img_resized = $2, cover_img_url = $3, cover_img_fixed = $4
            WHERE id = $5;
        `, coverImg, coverImgResized, coverImgURL, fixed, m.ID)
		if err != nil {
			return err
		}
	} else if m.URL != "" {
		result, err = tx.Exec(`
            UPDATE mangas
            SET cover_img = $1, cover_img_resized = $2, cover_img_url = $3, cover_img_fixed = $4
            WHERE url = $5;
        `, coverImg, coverImgResized, coverImgURL, fixed, m.URL)
		if err != nil {
			return err
		}
	} else {
		return errordefs.ErrMangaHasNoIDOrURL
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMangaNotFoundDB
	}

	return nil
}

// UpdateMangaMetadataDB updates the manga metadata in the database.
// It updates: the last released chapter (and its metadata), the manga name and cover image.
// The manga argument should have the ID or URL set to identify which manga to update.
// The other fields of the manga will be the new values for the manga in the database.
func UpdateMangaMetadataDB(m *Manga) error {
	contextError := "error updating manga '%s' metadata in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	err = updateMangaMetadata(m, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	return nil
}

func updateMangaMetadata(m *Manga, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	if m.LastReleasedChapter != nil {
		err = upsertMangaChapter(m.ID, m.LastReleasedChapter, tx)
		if err != nil {
			return err
		}
	}

	err = updateMangaNameDB(m, m.Name, tx)
	if err != nil {
		return err
	}

	if !m.CoverImgFixed {
		err = updateMangaCoverImg(m, m.CoverImg, m.CoverImgResized, m.CoverImgURL, m.CoverImgFixed, tx)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetMangaDB gets a manga from the database by its ID or URL
func GetMangaDB(mangaID ID, mangaURL string) (*Manga, error) {
	contextError := "error getting manga with ID '%d' and URL '%s' from DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaID, mangaURL), err)
	}

	mangaGet, err := getMangaFromDB(mangaID, mangaURL, db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaID, mangaURL), errordefs.ErrMangaNotFoundDB)
		}
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaID, mangaURL), err)
	}

	return mangaGet, nil
}

func getMangaFromDB(mangaID ID, mangaURL string, db *sql.DB) (*Manga, error) {
	var currentManga Manga
	var lastReleasedChapter, lastReadChapter Chapter

	var (
		lastReleasedChapterNameSelector, lastReleasedChapterNameAttribute, lastReleasedChapterNameRegex sql.NullString
		lastReleasedChapterURLSelector, lastReleasedChapterURLAttribute                                 sql.NullString
		lastReleasedChapterNameGetFirst, lastReleasedChapterURLGetFirst                                 sql.NullBool

		lastReleasedChapterURL, lastReleasedChapterChapter, lastReleasedChapterName, lastReleasedChapterInternalID sql.NullString
		lastReleasedChapterUpdatedAt                                                                               sql.NullTime
		lastReleasedChapterType, multiMangaID                                                                      sql.NullInt32

		lastReadChapterURL, lastReadChapterChapter, lastReadChapterName, lastReadChapterInternalID sql.NullString
		lastReadChapterUpdatedAt                                                                   sql.NullTime
		lastReadChapterType                                                                        sql.NullInt32
		lastReadChapterFromSourceSite                                                              sql.NullBool
	)
	if mangaID > 0 {
		query := `
            SELECT 
                mangas.id AS manga_id,
                mangas.source,
                mangas.url,
                mangas.name,
                mangas.internal_id,
                mangas.preferred_group,
                mangas.cover_img_url,
                mangas.cover_img,
                mangas.cover_img_resized,
                mangas.cover_img_fixed,
                mangas.status,
                mangas.multimanga_id AS multi_manga_id,
				mangas.last_released_chapter_name_selector,
				mangas.last_released_chapter_name_attribute,
				mangas.last_released_chapter_name_regex,
				mangas.last_released_chapter_name_get_first,
				mangas.last_released_chapter_url_selector,
				mangas.last_released_chapter_url_attribute,
				mangas.last_released_chapter_url_get_first,
				mangas.last_released_chapter_selector_use_browser,
                
                last_released_chapter.url AS last_released_chapter_url,
                last_released_chapter.chapter AS last_released_chapter,
                last_released_chapter.name AS last_released_chapter_name,
                last_released_chapter.internal_id AS last_released_chapter_internal_id,
                last_released_chapter.updated_at AS last_released_chapter_updated_at,
                last_released_chapter.type AS last_released_chapter_type,

                last_read_chapter.url AS last_read_chapter_url,
                last_read_chapter.chapter AS last_read_chapter,
                last_read_chapter.name AS last_read_chapter_name,
                last_read_chapter.internal_id AS last_read_chapter_internal_id,
                last_read_chapter.updated_at AS last_read_chapter_updated_at,
                last_read_chapter.type AS last_read_chapter_type,
                last_read_chapter.from_source_site AS last_read_chapter_from_source_site
            FROM 
                mangas
            LEFT JOIN 
                chapters AS last_released_chapter ON last_released_chapter.id = mangas.last_released_chapter
            LEFT JOIN 
                chapters AS last_read_chapter ON last_read_chapter.id = mangas.last_read_chapter
            WHERE
                mangas.id = $1;
        `
		err := db.QueryRow(query, mangaID).Scan(
			&currentManga.ID, &currentManga.Source, &currentManga.URL, &currentManga.Name,
			&currentManga.InternalID, &currentManga.PreferredGroup, &currentManga.CoverImgURL,
			&currentManga.CoverImg, &currentManga.CoverImgResized, &currentManga.CoverImgFixed, &currentManga.Status, &multiMangaID,

			&lastReleasedChapterNameSelector, &lastReleasedChapterNameAttribute, &lastReleasedChapterNameRegex, &lastReleasedChapterNameGetFirst,
			&lastReleasedChapterURLSelector, &lastReleasedChapterURLAttribute, &lastReleasedChapterURLGetFirst,
			&currentManga.LastReleasedChapterSelectorUseBrowser,

			&lastReleasedChapterURL, &lastReleasedChapterChapter, &lastReleasedChapterName,
			&lastReleasedChapterInternalID, &lastReleasedChapterUpdatedAt, &lastReleasedChapterType,

			&lastReadChapterURL, &lastReadChapterChapter, &lastReadChapterName,
			&lastReadChapterInternalID, &lastReadChapterUpdatedAt, &lastReadChapterType, &lastReadChapterFromSourceSite,
		)
		currentManga.SearchNames = []string{currentManga.Name}
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errordefs.ErrMangaNotFoundDB
			}
			return nil, err
		}
	} else if mangaURL != "" {
		query := `
            SELECT 
                mangas.id AS manga_id,
                mangas.source,
                mangas.url,
                mangas.name,
                mangas.internal_id,
                mangas.preferred_group,
                mangas.cover_img_url,
                mangas.cover_img,
                mangas.cover_img_resized,
                mangas.cover_img_fixed,
                mangas.status,
                mangas.multimanga_id AS multi_manga_id,
				mangas.last_released_chapter_name_selector,
				mangas.last_released_chapter_name_attribute,
				mangas.last_released_chapter_name_regex,
				mangas.last_released_chapter_name_get_first,
				mangas.last_released_chapter_url_selector,
				mangas.last_released_chapter_url_attribute,
				mangas.last_released_chapter_url_get_first,
				mangas.last_released_chapter_selector_use_browser,
                
                last_released_chapter.url AS last_released_chapter_url,
                last_released_chapter.chapter AS last_released_chapter,
                last_released_chapter.name AS last_released_chapter_name,
                last_released_chapter.internal_id AS last_released_chapter_internal_id,
                last_released_chapter.updated_at AS last_released_chapter_updated_at,
                last_released_chapter.type AS last_released_chapter_type,

                last_read_chapter.url AS last_read_chapter_url,
                last_read_chapter.chapter AS last_read_chapter,
                last_read_chapter.name AS last_read_chapter_name,
                last_read_chapter.internal_id AS last_read_chapter_internal_id,
                last_read_chapter.updated_at AS last_read_chapter_updated_at,
                last_read_chapter.type AS last_read_chapter_type,
                last_read_chapter.from_source_site AS last_read_chapter_from_source_site
            FROM 
                mangas
            LEFT JOIN 
                chapters AS last_released_chapter ON last_released_chapter.id = mangas.last_released_chapter
            LEFT JOIN 
                chapters AS last_read_chapter ON last_read_chapter.id = mangas.last_read_chapter
            WHERE
                mangas.url = $1;
        `
		err := db.QueryRow(query, mangaURL).Scan(
			&currentManga.ID, &currentManga.Source, &currentManga.URL, &currentManga.Name,
			&currentManga.InternalID, &currentManga.PreferredGroup, &currentManga.CoverImgURL,
			&currentManga.CoverImg, &currentManga.CoverImgResized, &currentManga.CoverImgFixed, &currentManga.Status, &multiMangaID,

			&lastReleasedChapterNameSelector, &lastReleasedChapterNameAttribute, &lastReleasedChapterNameRegex, &lastReleasedChapterNameGetFirst,
			&lastReleasedChapterURLSelector, &lastReleasedChapterURLAttribute, &lastReleasedChapterURLGetFirst,
			&currentManga.LastReleasedChapterSelectorUseBrowser,

			&lastReleasedChapterURL, &lastReleasedChapterChapter, &lastReleasedChapterName,
			&lastReleasedChapterInternalID, &lastReleasedChapterUpdatedAt, &lastReleasedChapterType,

			&lastReadChapterURL, &lastReadChapterChapter, &lastReadChapterName,
			&lastReadChapterInternalID, &lastReadChapterUpdatedAt, &lastReadChapterType, &lastReadChapterFromSourceSite,
		)
		currentManga.SearchNames = []string{currentManga.Name}
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errordefs.ErrMangaNotFoundDB
			}
			return nil, err
		}
	} else {
		return nil, errordefs.ErrMangaHasNoIDOrURL
	}

	if multiMangaID.Valid {
		currentManga.MultiMangaID = ID(multiMangaID.Int32)
	}

	if lastReleasedChapterNameSelector.Valid && lastReleasedChapterNameSelector.String != "" {
		currentManga.LastReleasedChapterNameSelector = &HTMLSelector{
			Selector:  lastReleasedChapterNameSelector.String,
			Attribute: lastReleasedChapterNameAttribute.String,
			Regex:     lastReleasedChapterNameRegex.String,
			GetFirst:  lastReleasedChapterNameGetFirst.Bool,
		}
	}
	if lastReleasedChapterURLSelector.Valid && lastReleasedChapterURLSelector.String != "" {
		currentManga.LastReleasedChapterURLSelector = &HTMLSelector{
			Selector:  lastReleasedChapterURLSelector.String,
			Attribute: lastReleasedChapterURLAttribute.String,
			GetFirst:  lastReleasedChapterURLGetFirst.Bool,
		}
	}

	if lastReleasedChapterURL.Valid {
		lastReleasedChapter.URL = lastReleasedChapterURL.String
		lastReleasedChapter.Chapter = lastReleasedChapterChapter.String
		lastReleasedChapter.Name = lastReleasedChapterName.String
		lastReleasedChapter.InternalID = lastReleasedChapterInternalID.String
		lastReleasedChapter.UpdatedAt = lastReleasedChapterUpdatedAt.Time
		lastReleasedChapter.Type = Type(lastReleasedChapterType.Int32)
		currentManga.LastReleasedChapter = &lastReleasedChapter
	}

	if lastReadChapterURL.Valid {
		lastReadChapter.URL = lastReadChapterURL.String
		lastReadChapter.Chapter = lastReadChapterChapter.String
		lastReadChapter.Name = lastReadChapterName.String
		lastReadChapter.InternalID = lastReadChapterInternalID.String
		lastReadChapter.UpdatedAt = lastReadChapterUpdatedAt.Time
		lastReadChapter.Type = Type(lastReadChapterType.Int32)
		lastReadChapter.FromSourceSite = lastReadChapterFromSourceSite.Bool
		currentManga.LastReadChapter = &lastReadChapter
	}

	err := validateManga(&currentManga)
	if err != nil {
		return nil, err
	}

	return &currentManga, nil
}

func getMangaIDByURL(url string) (ID, error) {
	contextError := "error getting manga ID by URL '%s' from DB"

	db, err := db.OpenConn()
	if err != nil {
		return -1, util.AddErrorContext(fmt.Sprintf(contextError, url), err)
	}

	var mangaID ID
	err = db.QueryRow(`
        SELECT id
        FROM mangas
        WHERE url = $1;
    `, url).Scan(&mangaID)
	if err != nil {
		if err == sql.ErrNoRows {
			return -1, util.AddErrorContext(fmt.Sprintf(contextError, url), errordefs.ErrMangaNotFoundDB)
		}
		return -1, util.AddErrorContext(fmt.Sprintf(contextError, url), err)
	}

	return mangaID, nil
}

// GetMangasWithoutMultiMangasDB gets all mangas from the database that are not part of a multimanga or are not custom mangas.
// Used only to get mangas to be transformed into multimangas.
func GetMangasWithoutMultiMangasDB(customManga bool) ([]*Manga, error) {
	contextError := "error getting mangas without multimangas from DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	mangas := make([]*Manga, 0)
	if customManga {
		mangas, err = getMangasWithoutMultiMangasFromDB(db, true)
	} else {
		mangas, err = getMangasWithoutMultiMangasFromDB(db, false)
	}
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	return mangas, nil
}

func getMangasWithoutMultiMangasFromDB(db *sql.DB, customManga bool) ([]*Manga, error) {
	query := `
        SELECT 
            mangas.id AS manga_id,
            mangas.source,
            mangas.url,
            mangas.name,
            mangas.internal_id,
            mangas.preferred_group,
            mangas.cover_img_url,
            mangas.cover_img,
            mangas.cover_img_resized,
            mangas.status,
			mangas.last_released_chapter_name_selector,
			mangas.last_released_chapter_name_attribute,
			mangas.last_released_chapter_name_regex,
			mangas.last_released_chapter_name_get_first,
			mangas.last_released_chapter_url_selector,
			mangas.last_released_chapter_url_attribute,
			mangas.last_released_chapter_url_get_first,
			mangas.last_released_chapter_selector_use_browser,

            last_released_chapter.url AS last_released_chapter_url,
            last_released_chapter.chapter AS last_released_chapter,
            last_released_chapter.name AS last_released_chapter_name,
            last_released_chapter.internal_id AS last_released_chapter_internal_id,
            last_released_chapter.updated_at AS last_released_chapter_updated_at,
            last_released_chapter.type AS last_released_chapter_type,

            last_read_chapter.url AS last_read_chapter_url,
            last_read_chapter.chapter AS last_read_chapter,
            last_read_chapter.name AS last_read_chapter_name,
            last_read_chapter.internal_id AS last_read_chapter_internal_id,
            last_read_chapter.updated_at AS last_read_chapter_updated_at,
            last_read_chapter.type AS last_read_chapter_type,
			last_read_chapter.from_source_site AS last_read_chapter_from_source_site
        FROM 
            mangas
        LEFT JOIN 
            chapters AS last_released_chapter ON last_released_chapter.id = mangas.last_released_chapter
        LEFT JOIN 
            chapters AS last_read_chapter ON last_read_chapter.id = mangas.last_read_chapter
        WHERE
            mangas.multimanga_id is NULL
    `
	if customManga {
		query = query + " AND mangas.source = $1;"
	} else {
		query = query + " AND mangas.source <> $1;"
	}
	rows, err := db.Query(query, CustomMangaSource)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangas []*Manga
	for rows.Next() {
		var currentManga Manga
		var lastReadChapter, lastReleasedChapter Chapter

		var (
			lastReleasedChapterNameSelector, lastReleasedChapterNameAttribute, lastReleasedChapterNameRegex sql.NullString
			lastReleasedChapterURLSelector, lastReleasedChapterURLAttribute                                 sql.NullString
			lastReleasedChapterNameGetFirst, lastReleasedChapterURLGetFirst                                 sql.NullBool

			lastReleasedChapterURL, lastReleasedChapterChapter, lastReleasedChapterName, lastReleasedChapterInternalID sql.NullString
			lastReleasedChapterUpdatedAt                                                                               sql.NullTime
			lastReleasedChapterType                                                                                    sql.NullInt32

			lastReadChapterURL, lastReadChapterChapter, lastReadChapterName, lastReadChapterInternalID sql.NullString
			lastReadChapterUpdatedAt                                                                   sql.NullTime
			lastReadChapterType                                                                        sql.NullInt32
			lastReadChapterFromSourceSite                                                              sql.NullBool
		)

		err := rows.Scan(
			&currentManga.ID, &currentManga.Source, &currentManga.URL, &currentManga.Name,
			&currentManga.InternalID, &currentManga.PreferredGroup, &currentManga.CoverImgURL,
			&currentManga.CoverImg, &currentManga.CoverImgResized, &currentManga.Status,

			&lastReleasedChapterNameSelector, &lastReleasedChapterNameAttribute, &lastReleasedChapterNameRegex, &lastReleasedChapterNameGetFirst,
			&lastReleasedChapterURLSelector, &lastReleasedChapterURLAttribute, &lastReleasedChapterURLGetFirst,
			&currentManga.LastReleasedChapterSelectorUseBrowser,

			&lastReleasedChapterURL, &lastReleasedChapterChapter, &lastReleasedChapterName,
			&lastReleasedChapterInternalID, &lastReleasedChapterUpdatedAt, &lastReleasedChapterType,

			&lastReadChapterURL, &lastReadChapterChapter, &lastReadChapterName,
			&lastReadChapterInternalID, &lastReadChapterUpdatedAt, &lastReadChapterType, &lastReadChapterFromSourceSite,
		)
		if err != nil {
			return nil, err
		}
		currentManga.SearchNames = []string{currentManga.Name}

		if lastReleasedChapterNameSelector.Valid && lastReleasedChapterNameSelector.String != "" {
			currentManga.LastReleasedChapterNameSelector = &HTMLSelector{
				Selector:  lastReleasedChapterNameSelector.String,
				Attribute: lastReleasedChapterNameAttribute.String,
				Regex:     lastReleasedChapterNameRegex.String,
				GetFirst:  lastReleasedChapterNameGetFirst.Bool,
			}
		}
		if lastReleasedChapterURLSelector.Valid && lastReleasedChapterURLSelector.String != "" {
			currentManga.LastReleasedChapterURLSelector = &HTMLSelector{
				Selector:  lastReleasedChapterURLSelector.String,
				Attribute: lastReleasedChapterURLAttribute.String,
				GetFirst:  lastReleasedChapterURLGetFirst.Bool,
			}
		}

		if lastReleasedChapterURL.Valid {
			lastReleasedChapter.URL = lastReleasedChapterURL.String
			lastReleasedChapter.Chapter = lastReleasedChapterChapter.String
			lastReleasedChapter.Name = lastReleasedChapterName.String
			lastReleasedChapter.InternalID = lastReleasedChapterInternalID.String
			lastReleasedChapter.UpdatedAt = lastReleasedChapterUpdatedAt.Time
			lastReleasedChapter.Type = Type(lastReleasedChapterType.Int32)
			currentManga.LastReleasedChapter = &lastReleasedChapter
		}

		if lastReadChapterURL.Valid {
			lastReadChapter.URL = lastReadChapterURL.String
			lastReadChapter.Chapter = lastReadChapterChapter.String
			lastReadChapter.Name = lastReadChapterName.String
			lastReadChapter.InternalID = lastReadChapterInternalID.String
			lastReadChapter.UpdatedAt = lastReadChapterUpdatedAt.Time
			lastReadChapter.Type = Type(lastReadChapterType.Int32)
			lastReadChapter.FromSourceSite = lastReadChapterFromSourceSite.Bool
			currentManga.LastReadChapter = &lastReadChapter
		}

		err = validateManga(&currentManga)
		if err != nil {
			return nil, err
		}

		mangas = append(mangas, &currentManga)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return mangas, nil
}

// GetCustomMangasDB gets all custom mangas from the database
func GetCustomMangasDB() ([]*Manga, error) {
	contextError := "error getting custom mangas from DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	mangas, err := getCustomMangasFromDB(db)
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	return mangas, nil
}

func getCustomMangasFromDB(db *sql.DB) ([]*Manga, error) {
	query := `
        SELECT 
            mangas.id AS manga_id,
            mangas.source,
            mangas.url,
            mangas.name,
            mangas.internal_id,
            mangas.preferred_group,
            mangas.cover_img_url,
            mangas.cover_img,
            mangas.cover_img_resized,
            mangas.status,
			mangas.last_released_chapter_name_selector,
			mangas.last_released_chapter_name_attribute,
			mangas.last_released_chapter_name_regex,
			mangas.last_released_chapter_name_get_first,
			mangas.last_released_chapter_url_selector,
			mangas.last_released_chapter_url_attribute,
			mangas.last_released_chapter_url_get_first,
			mangas.last_released_chapter_selector_use_browser,
            
            last_released_chapter.url AS last_released_chapter_url,
            last_released_chapter.chapter AS last_released_chapter,
            last_released_chapter.name AS last_released_chapter_name,
            last_released_chapter.internal_id AS last_released_chapter_internal_id,
            last_released_chapter.updated_at AS last_released_chapter_updated_at,
            last_released_chapter.type AS last_released_chapter_type,

            last_read_chapter.url AS last_read_chapter_url,
            last_read_chapter.chapter AS last_read_chapter,
            last_read_chapter.name AS last_read_chapter_name,
            last_read_chapter.internal_id AS last_read_chapter_internal_id,
            last_read_chapter.updated_at AS last_read_chapter_updated_at,
            last_read_chapter.type AS last_read_chapter_type,
			last_read_chapter.from_source_site AS last_read_chapter_from_source_site
        FROM 
            mangas
        LEFT JOIN 
            chapters AS last_released_chapter ON last_released_chapter.id = mangas.last_released_chapter
        LEFT JOIN 
            chapters AS last_read_chapter ON last_read_chapter.id = mangas.last_read_chapter
        WHERE
            mangas.source = $1
        ;
    `
	rows, err := db.Query(query, CustomMangaSource)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangas []*Manga
	for rows.Next() {
		var currentManga Manga
		var lastReleasedChapter, lastReadChapter Chapter

		var (
			lastReleasedChapterNameSelector, lastReleasedChapterNameAttribute, lastReleasedChapterNameRegex sql.NullString
			lastReleasedChapterURLSelector, lastReleasedChapterURLAttribute                                 sql.NullString
			lastReleasedChapterNameGetFirst, lastReleasedChapterURLGetFirst                                 sql.NullBool

			lastReleasedChapterURL, lastReleasedChapterChapter, lastReleasedChapterName, lastReleasedChapterInternalID sql.NullString
			lastReleasedChapterUpdatedAt                                                                               sql.NullTime
			lastReleasedChapterType                                                                                    sql.NullInt32

			lastReadChapterURL, lastReadChapterChapter, lastReadChapterName, lastReadChapterInternalID sql.NullString
			lastReadChapterUpdatedAt                                                                   sql.NullTime
			lastReadChapterType                                                                        sql.NullInt32
			lastReadChapterFromSourceSite                                                              sql.NullBool
		)

		err := rows.Scan(
			&currentManga.ID, &currentManga.Source, &currentManga.URL, &currentManga.Name,
			&currentManga.InternalID, &currentManga.PreferredGroup, &currentManga.CoverImgURL,
			&currentManga.CoverImg, &currentManga.CoverImgResized, &currentManga.Status,

			&lastReleasedChapterNameSelector, &lastReleasedChapterNameAttribute, &lastReleasedChapterNameRegex, &lastReleasedChapterNameGetFirst,
			&lastReleasedChapterURLSelector, &lastReleasedChapterURLAttribute, &lastReleasedChapterURLGetFirst,
			&currentManga.LastReleasedChapterSelectorUseBrowser,

			&lastReleasedChapterURL, &lastReleasedChapterChapter, &lastReleasedChapterName,
			&lastReleasedChapterInternalID, &lastReleasedChapterUpdatedAt, &lastReleasedChapterType,

			&lastReadChapterURL, &lastReadChapterChapter, &lastReadChapterName,
			&lastReadChapterInternalID, &lastReadChapterUpdatedAt, &lastReadChapterType, &lastReadChapterFromSourceSite,
		)
		if err != nil {
			return nil, err
		}

		currentManga.SearchNames = []string{currentManga.Name}

		if lastReleasedChapterNameSelector.Valid && lastReleasedChapterNameSelector.String != "" {
			currentManga.LastReleasedChapterNameSelector = &HTMLSelector{
				Selector:  lastReleasedChapterNameSelector.String,
				Attribute: lastReleasedChapterNameAttribute.String,
				Regex:     lastReleasedChapterNameRegex.String,
				GetFirst:  lastReleasedChapterNameGetFirst.Bool,
			}
		}
		if lastReleasedChapterURLSelector.Valid && lastReleasedChapterURLSelector.String != "" {
			currentManga.LastReleasedChapterURLSelector = &HTMLSelector{
				Selector:  lastReleasedChapterURLSelector.String,
				Attribute: lastReleasedChapterURLAttribute.String,
				GetFirst:  lastReleasedChapterURLGetFirst.Bool,
			}
		}

		if lastReleasedChapterURL.Valid {
			lastReleasedChapter.URL = lastReleasedChapterURL.String
			lastReleasedChapter.Chapter = lastReleasedChapterChapter.String
			lastReleasedChapter.Name = lastReleasedChapterName.String
			lastReleasedChapter.InternalID = lastReleasedChapterInternalID.String
			lastReleasedChapter.UpdatedAt = lastReleasedChapterUpdatedAt.Time
			lastReleasedChapter.Type = Type(lastReleasedChapterType.Int32)
			currentManga.LastReleasedChapter = &lastReleasedChapter
		}

		if lastReadChapterURL.Valid {
			lastReadChapter.URL = lastReadChapterURL.String
			lastReadChapter.Chapter = lastReadChapterChapter.String
			lastReadChapter.Name = lastReadChapterName.String
			lastReadChapter.InternalID = lastReadChapterInternalID.String
			lastReadChapter.UpdatedAt = lastReadChapterUpdatedAt.Time
			lastReadChapter.Type = Type(lastReadChapterType.Int32)
			lastReadChapter.FromSourceSite = lastReadChapterFromSourceSite.Bool
			currentManga.LastReadChapter = &lastReadChapter
		}

		err = validateManga(&currentManga)
		if err != nil {
			return nil, err
		}

		mangas = append(mangas, &currentManga)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return mangas, nil
}

func GetLibraryStats() (map[string]int, error) {
	contextError := "error getting library stats"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	stats, err := getLibraryStatsFromDB(db)
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	return stats, nil
}

func getLibraryStatsFromDB(db *sql.DB) (map[string]int, error) {
	query := `
        SELECT
            COUNT(*) AS total_mangas, 
            status
        FROM
            multimangas
        GROUP BY
            status
        ;
    `
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)

	for rows.Next() {
		var totalMangas, statusInt int

		err := rows.Scan(&totalMangas, &statusInt)
		if err != nil {
			return nil, err
		}

		status, err := getStatusStr(statusInt)
		if err != nil {
			return nil, err
		}

		stats[status] = totalMangas
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Every manga, custom or not, belongs to a multimanga, so only multimangas are counted.
	query = `
        SELECT
            last_read_chapter.chapter,
            last_read_chapter.url,
            last_released_chapter.chapter,
            last_released_chapter.url
        FROM
            multimangas
        JOIN
            mangas ON mangas.id = multimangas.current_manga
        LEFT JOIN
            chapters AS last_read_chapter ON last_read_chapter.id = multimangas.last_read_chapter
        LEFT JOIN
            chapters AS last_released_chapter ON last_released_chapter.id = mangas.last_released_chapter
        ;
    `
	rows, err = db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var unread, total int
	for rows.Next() {
		var lastRead, lastReadURL, lastReleased, lastReleasedURL sql.NullString
		err := rows.Scan(&lastRead, &lastReadURL, &lastReleased, &lastReleasedURL)
		if err != nil {
			return nil, err
		}
		total++
		if IsUnreadChapter(lastRead.String, lastReadURL.String, lastReleased.String, lastReleasedURL.String) {
			unread++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats["Unread"] = unread
	stats["Total"] = total
	stats["Read"] = total - unread

	return stats, nil
}

func (m *Manga) UpdateSourceInDB(source string) error {
	contextError := "error updating manga '%s' source to '%s' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, source), err)
	}

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, source), err)
	}

	err = updateMangaSourceDB(m, source, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, source), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, source), err)
	}
	m.Source = source

	return nil
}

func updateMangaSourceDB(m *Manga, source string, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	var result sql.Result
	if m.ID > 0 {
		result, err = tx.Exec(`
            UPDATE mangas
            SET source = $1
            WHERE id = $2;
        `, source, m.ID)
		if err != nil {
			return err
		}
	} else if m.URL != "" {
		result, err = tx.Exec(`
            UPDATE mangas
            SET source = $1
            WHERE url = $2;
        `, source, m.URL)
		if err != nil {
			return err
		}
	} else {
		return errordefs.ErrMangaHasNoIDOrURL
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMangaNotFoundDB
	}

	return nil
}

func (m *Manga) UpdateLastReleasedChapterSelectorsInDB(nameSelector, URLSelector *HTMLSelector, useBrowser bool) error {
	var chapter *Chapter
	var err error

	contextError := "error updating manga '%s' chapter name selector to '%s' and URL selector to '%s' in DB"
	emptySelectors := (nameSelector == nil || nameSelector.Selector == "") &&
		(URLSelector == nil || URLSelector.Selector == "")
	if !emptySelectors {
		chapter, err = GetCustomMangaLastReleasedChapter(m.URL, nameSelector, URLSelector, useBrowser)
		if err != nil {
			return util.AddErrorContext(fmt.Sprintf(contextError, m, nameSelector, URLSelector), err)
		}
	}

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, nameSelector, URLSelector), err)
	}

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, nameSelector, URLSelector), err)
	}

	err = updateMangaLastReleasedChapterSelectorDB(m, nameSelector, URLSelector, useBrowser, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, nameSelector, URLSelector), err)
	}

	if !emptySelectors {
		err = upsertMangaChapter(m.ID, chapter, tx)
		if err != nil {
			tx.Rollback()
			return util.AddErrorContext(fmt.Sprintf(contextError, m, nameSelector, URLSelector), err)
		}
		m.LastReleasedChapter = chapter
	} else {
		if m.LastReleasedChapter != nil {
			err = deleteMangaChapter(m.ID, m.LastReleasedChapter, tx)
			if err != nil {
				tx.Rollback()
				return util.AddErrorContext(fmt.Sprintf(contextError, m, nameSelector, URLSelector), err)
			}
			m.LastReleasedChapter = nil
		}
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, nameSelector, URLSelector), err)
	}
	m.LastReleasedChapterNameSelector = nameSelector
	m.LastReleasedChapterURLSelector = URLSelector

	return nil
}

// CustomMangaUpdate holds the new values of a custom manga, see Manga.UpdateCustomMangaInDB.
type CustomMangaUpdate struct {
	Name string
	// URL is the manga URL. If empty, the manga has no URL.
	URL                       string
	NameSelector, URLSelector *HTMLSelector
	UseBrowser                bool
	CoverImg                  []byte
	CoverImgURL               string
	CoverImgResized           bool
	// UpdateCoverImg tells if CoverImg, CoverImgURL and CoverImgResized should be saved.
	UpdateCoverImg bool
}

// UpdateCustomMangaInDB updates all the custom manga fields at once.
// If the URL or selectors changed, the last released chapter is got from the page using
// the new URL and selectors before saving anything, so an invalid combination saves nothing.
func (m *Manga) UpdateCustomMangaInDB(update CustomMangaUpdate) error {
	contextError := "error updating custom manga '%s' in DB"

	if update.NameSelector != nil && update.NameSelector.Selector == "" {
		update.NameSelector = nil
	}
	if update.URLSelector != nil && update.URLSelector.Selector == "" {
		update.URLSelector = nil
	}
	if update.URLSelector != nil {
		update.URLSelector.Regex = ""
	}
	hasSelectors := update.NameSelector != nil || update.URLSelector != nil

	currentURL := m.URL
	if strings.HasPrefix(currentURL, CustomMangaURLPrefix) {
		currentURL = ""
	}
	newURL := update.URL
	if newURL == "" {
		if hasSelectors {
			return util.AddErrorContext(fmt.Sprintf(contextError, m), &errordefs.UserError{Message: "Provide the manga URL to use the last released chapter selectors"})
		}
		newURL = m.URL
		if currentURL != "" {
			newURL = CustomMangaURLPrefix + "/" + uuid.New().String()
		}
	}

	trackingChanged := update.URL != currentURL ||
		!sameSelector(update.NameSelector, m.LastReleasedChapterNameSelector) ||
		!sameSelector(update.URLSelector, m.LastReleasedChapterURLSelector) ||
		update.UseBrowser != m.LastReleasedChapterSelectorUseBrowser

	var chapter *Chapter
	var err error
	if trackingChanged && hasSelectors {
		chapter, err = GetCustomMangaLastReleasedChapter(newURL, update.NameSelector, update.URLSelector, update.UseBrowser)
		if err != nil {
			return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
		}
	}

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	err = func() error {
		if update.Name != m.Name {
			if err := updateMangaNameDB(m, update.Name, tx); err != nil {
				return err
			}
		}
		if newURL != m.URL {
			if err := updateMangaURLDB(m, newURL, tx); err != nil {
				return err
			}
		}
		if trackingChanged {
			if err := updateMangaLastReleasedChapterSelectorDB(m, update.NameSelector, update.URLSelector, update.UseBrowser, tx); err != nil {
				return err
			}
			if chapter != nil {
				if err := upsertMangaChapter(m.ID, chapter, tx); err != nil {
					return err
				}
			} else if !hasSelectors && m.LastReleasedChapter != nil {
				if err := deleteMangaChapter(m.ID, m.LastReleasedChapter, tx); err != nil {
					return err
				}
			}
		}
		if update.UpdateCoverImg {
			if err := updateMangaCoverImg(m, update.CoverImg, update.CoverImgResized, update.CoverImgURL, true, tx); err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	m.Name = update.Name
	m.SearchNames = []string{update.Name}
	m.URL = newURL
	if trackingChanged {
		m.LastReleasedChapterNameSelector = update.NameSelector
		m.LastReleasedChapterURLSelector = update.URLSelector
		m.LastReleasedChapterSelectorUseBrowser = update.UseBrowser
		if chapter != nil || !hasSelectors {
			m.LastReleasedChapter = chapter
		}
	}
	if update.UpdateCoverImg {
		m.CoverImg = update.CoverImg
		m.CoverImgURL = update.CoverImgURL
		m.CoverImgResized = update.CoverImgResized
		m.CoverImgFixed = true
	}

	return nil
}

// sameSelector tells if two selectors are equal. An empty selector is the same as no selector.
func sameSelector(a, b *HTMLSelector) bool {
	if a == nil || a.Selector == "" {
		return b == nil || b.Selector == ""
	}
	if b == nil {
		return false
	}
	return *a == *b
}

func updateMangaLastReleasedChapterSelectorDB(m *Manga, nameSelector, URLSelector *HTMLSelector, useBrowser bool, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	if nameSelector == nil {
		nameSelector = &HTMLSelector{}
	}
	if URLSelector == nil {
		URLSelector = &HTMLSelector{}
	}

	var result sql.Result
	if m.ID > 0 {
		result, err = tx.Exec(`
            UPDATE mangas
            SET
				last_released_chapter_name_selector = $1,
				last_released_chapter_name_attribute = $2,
				last_released_chapter_name_regex = $3,
				last_released_chapter_name_get_first = $4,
				last_released_chapter_url_selector = $5,
				last_released_chapter_url_attribute = $6,
				last_released_chapter_url_get_first = $7,
				last_released_chapter_selector_use_browser = $8
            WHERE id = $9;
        `, nameSelector.Selector, nameSelector.Attribute, nameSelector.Regex, nameSelector.GetFirst,
			URLSelector.Selector, URLSelector.Attribute, URLSelector.GetFirst, useBrowser, m.ID)
		if err != nil {
			return err
		}
	} else if m.URL != "" {
		result, err = tx.Exec(`
            UPDATE mangas
            SET
				last_released_chapter_name_selector = $1,
				last_released_chapter_name_attribute = $2,
				last_released_chapter_name_regex = $3,
				last_released_chapter_name_get_first = $4,
				last_released_chapter_url_selector = $5,
				last_released_chapter_url_attribute = $6,
				last_released_chapter_url_get_first = $7,
				last_released_chapter_selector_use_browser = $8
            WHERE url = $9;
        `, nameSelector.Selector, nameSelector.Attribute, nameSelector.Regex, nameSelector.GetFirst,
			URLSelector.Selector, URLSelector.Attribute, URLSelector.GetFirst, useBrowser, m.URL)
		if err != nil {
			return err
		}
	} else {
		return errordefs.ErrMangaHasNoIDOrURL
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMangaNotFoundDB
	}

	return nil
}

// validateManga should be used every time the API interacts with the mangas table in the database
func validateManga(m *Manga) error {
	contextError := "error validating manga"

	if m == nil {
		return util.AddErrorContext(contextError, fmt.Errorf("manga is nil"))
	}

	if m.Source == "" {
		return util.AddErrorContext(contextError, fmt.Errorf("manga source is empty"))
	}
	if m.Name == "" {
		return util.AddErrorContext(contextError, fmt.Errorf("manga name is empty"))
	}
	if m.CoverImg == nil {
		return util.AddErrorContext(contextError, fmt.Errorf("manga cover image is nil"))
	}

	if m.LastReleasedChapter != nil {
		err := validateChapter(m.LastReleasedChapter)
		if err != nil {
			return util.AddErrorContext(contextError+" last released chapter", err)
		}
	}
	if m.LastReadChapter != nil {
		err := validateChapter(m.LastReadChapter)
		if err != nil {
			return util.AddErrorContext(contextError+" last read chapter", err)
		}
	}

	return nil
}

func ValidateStatus(status Status) error {
	if status < 1 || status > 5 {
		return fmt.Errorf("status should be >= 1 && <= 5, instead it's %d", status)
	}

	return nil
}

// IsUnreadChapter tells if the last released chapter wasn't read yet.
// It's the same rule used by the dashboard (is_unread_chapter), keep them in sync.
func IsUnreadChapter(lastRead, lastReadURL, lastReleased, lastReleasedURL string) bool {
	if lastReleased == "" {
		return false
	}
	if lastRead == lastReleased {
		// Every chapter of a custom manga tracked only by the chapter URL has the same name
		return lastReleased == UnknownChapter && lastReadURL != lastReleasedURL
	}
	lastReadNumber, err := strconv.ParseFloat(lastRead, 64)
	if err != nil {
		return true
	}
	lastReleasedNumber, err := strconv.ParseFloat(lastReleased, 64)
	if err != nil {
		return true
	}

	return lastReadNumber < lastReleasedNumber
}

// FilterUnreadChapterMangas filters a list of mangas to return
// mangas where the last released chapter is different from the
// last read chapter
func FilterUnreadChapterMangas(mangas []*Manga) []*Manga {
	unreadChapterMangas := []*Manga{}

	for _, manga := range mangas {
		if manga.LastReleasedChapter != nil && manga.LastReadChapter != nil {
			if manga.LastReleasedChapter.Chapter != manga.LastReadChapter.Chapter ||
				// Every chapter of a custom manga tracked only by the chapter URL has the same name
				(manga.LastReleasedChapter.Chapter == UnknownChapter && manga.LastReleasedChapter.URL != manga.LastReadChapter.URL) {
				unreadChapterMangas = append(unreadChapterMangas, manga)
			}
		} else if manga.LastReleasedChapter != nil && manga.LastReadChapter == nil {
			unreadChapterMangas = append(unreadChapterMangas, manga)
		} else if (manga.LastReleasedChapter == nil && manga.LastReadChapter != nil) && manga.Source != CustomMangaSource {
			unreadChapterMangas = append(unreadChapterMangas, manga)
		}
	}

	return unreadChapterMangas
}

// SortMangasByLastReleasedChapterUpdatedAt sorts a list of mangas
// by their last released chapter updated at property, desc
func SortMangasByLastReleasedChapterUpdatedAt(mangas []*Manga) {
	sort.Slice(mangas, func(i, j int) bool {
		iChapter := mangas[i].LastReleasedChapter
		jChapter := mangas[j].LastReleasedChapter

		if iChapter == nil {
			if mangas[i].Source == CustomMangaSource {
				if mangas[i].LastReadChapter == nil {
					return false
				}
				iChapter = mangas[i].LastReadChapter
			} else {
				return false
			}
		}

		if jChapter == nil {
			if mangas[j].Source == CustomMangaSource {
				if mangas[j].LastReadChapter == nil {
					return true
				}
				jChapter = mangas[j].LastReadChapter
			} else {
				return true
			}
		}

		if iChapter.UpdatedAt.IsZero() && !jChapter.UpdatedAt.IsZero() {
			return true
		}
		if jChapter.UpdatedAt.IsZero() && !iChapter.UpdatedAt.IsZero() {
			return false
		}

		return iChapter.UpdatedAt.After(jChapter.UpdatedAt)
	})
}

func getStatusStr(status int) (string, error) {
	switch status {
	case 1:
		return "Reading", nil
	case 2:
		return "Completed", nil
	case 3:
		return "On Hold", nil
	case 4:
		return "Dropped", nil
	case 5:
		return "Plan to Read", nil
	default:
		return "i", fmt.Errorf("invalid status: %d", status)
	}
}

// HTMLSelector is used to select an element from an HTML document
type HTMLSelector struct {
	GetFirst  bool   `json:"GetFirst"`
	Selector  string `json:"Selector" binding:"required"`
	Attribute string `json:"Attribute"`
	Regex     string `json:"Regex"`
}

func (s HTMLSelector) String() string {
	return fmt.Sprintf("HTMLSelector{Selector: %s, Attr: %s, Regex: %s, GetFirst: %t}", s.Selector, s.Attribute, s.Regex, s.GetFirst)
}

// GetCustomMangaLastReleasedChapter gets the last released chapter of a custom manga
func GetCustomMangaLastReleasedChapter(mangaURL string, nameSelector, URLSelector *HTMLSelector, useBrowser bool) (*Chapter, error) {
	contextError := "error getting custom manga '%s' last released chapter with name selector '%s' and URL selector '%s' from source (browser: %t)"

	if nameSelector == nil && URLSelector == nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, nameSelector, URLSelector, useBrowser), fmt.Errorf("both manga last released chapter selectors are nil"))
	}
	for _, s := range []struct {
		selector *HTMLSelector
		field    string
	}{{nameSelector, chapterNameField}, {URLSelector, chapterURLField}} {
		if s.selector == nil {
			continue
		}
		if err := selectorPrefixError(s.selector, s.field); err != nil {
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, nameSelector, URLSelector, useBrowser), err)
		}
	}

	var page *scrapedPage
	var err error
	if useBrowser {
		page, err = getPageUsingBrowser(mangaURL, nameSelector, URLSelector)
	} else {
		page, err = getPageUsingHTTPRequest(mangaURL)
	}
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, nameSelector, URLSelector, useBrowser), err)
	}

	chapter := &Chapter{
		UpdatedAt: time.Now().Local().Truncate(time.Second),
		Type:      1,
	}
	if nameSelector != nil {
		chapterName, err := page.selectValue(nameSelector, chapterNameField)
		if err != nil {
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, nameSelector, URLSelector, useBrowser), err)
		}
		chapter.Chapter = chapterName
		chapter.Name = "Chapter " + chapterName
	}
	if URLSelector != nil {
		chapterURL, err := page.selectValue(URLSelector, chapterURLField)
		if err != nil {
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, nameSelector, URLSelector, useBrowser), err)
		}
		chapter.URL, err = page.resolveURL(chapterURL)
		if err != nil {
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, nameSelector, URLSelector, useBrowser), err)
		}
		if chapter.Chapter == "" {
			chapter.Chapter = UnknownChapter
			chapter.Name = "Chapter " + UnknownChapter
		}
	}

	return chapter, nil
}

var (
	userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"
	// timeout is the timeout to get the page using an HTTP request, and to wait for each selector's element using the browser.
	timeout = time.Second * 15
	// browserTimeout is the timeout of the whole page scraping using the browser.
	browserTimeout = time.Second * 60
)

// Names of the custom manga fields scraped by a selector, used in the messages shown to the user.
const (
	chapterNameField = "Chapter name"
	chapterURLField  = "Chapter URL"
)

// scrapedPage is a page fetched once to apply all custom manga selectors to it.
type scrapedPage struct {
	// url is the page final URL, after redirects.
	url         *url.URL
	body        []byte
	contentType string
}

// isXML tells if the page is an XML document instead of an HTML page, the same way colly does.
func (p *scrapedPage) isXML() bool {
	contentType := strings.ToLower(p.contentType)
	if strings.Contains(contentType, "html") {
		return false
	}
	path := strings.ToLower(p.url.Path)
	return strings.Contains(contentType, "xml") || strings.HasSuffix(path, ".xml") || strings.HasSuffix(path, ".xml.gz")
}

// selectValue applies a selector to the page and returns the selected value.
// If more than one element matches the selector, the last is used, or the first if selector.GetFirst is true.
func (p *scrapedPage) selectValue(selector *HTMLSelector, field string) (string, error) {
	if err := selectorPrefixError(selector, field); err != nil {
		return "", err
	}

	var selection string
	var matched bool
	if after, ok := strings.CutPrefix(selector.Selector, "css:"); ok {
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(p.body))
		if err != nil {
			return "", util.AddErrorContext("error parsing page HTML", err)
		}
		s := doc.Find(after)
		if selector.GetFirst {
			s = s.First()
		} else {
			s = s.Last()
		}
		matched = s.Length() > 0
		if selector.Attribute != "" {
			selection = s.AttrOr(selector.Attribute, "")
		} else {
			selection = s.Text()
		}
	} else {
		after := strings.TrimPrefix(selector.Selector, "xpath:")
		if p.isXML() {
			doc, err := xmlquery.Parse(bytes.NewReader(p.body))
			if err != nil {
				return "", util.AddErrorContext("error parsing page XML", err)
			}
			nodes, err := xmlquery.QueryAll(doc, after)
			if err != nil {
				return "", &errordefs.UserError{Message: fmt.Sprintf("%s selector is not a valid XPath: %s", field, err)}
			}
			matched = len(nodes) > 0
			if matched {
				node := nodes[len(nodes)-1]
				if selector.GetFirst {
					node = nodes[0]
				}
				if selector.Attribute != "" {
					selection = node.SelectAttr(selector.Attribute)
				} else {
					selection = node.InnerText()
				}
			}
		} else {
			doc, err := htmlquery.Parse(bytes.NewReader(p.body))
			if err != nil {
				return "", util.AddErrorContext("error parsing page HTML", err)
			}
			nodes, err := htmlquery.QueryAll(doc, after)
			if err != nil {
				return "", &errordefs.UserError{Message: fmt.Sprintf("%s selector is not a valid XPath: %s", field, err)}
			}
			matched = len(nodes) > 0
			if matched {
				node := nodes[len(nodes)-1]
				if selector.GetFirst {
					node = nodes[0]
				}
				if selector.Attribute != "" {
					selection = htmlquery.SelectAttr(node, selector.Attribute)
				} else {
					selection = htmlquery.InnerText(node)
				}
			}
		}
	}

	return processSelection(selection, matched, selector, field)
}

// resolveURL resolves a chapter URL found in the page, which can be relative, into an absolute URL.
// Relative URLs are resolved against the page's <base> element, if any, else against the page URL.
func (p *scrapedPage) resolveURL(rawURL string) (string, error) {
	ref, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", &errordefs.UserError{Message: fmt.Sprintf("%s selector returned an invalid URL: %s", chapterURLField, rawURL), Err: err}
	}

	base := p.url
	if !p.isXML() {
		if doc, err := htmlquery.Parse(bytes.NewReader(p.body)); err == nil {
			if baseElement := htmlquery.FindOne(doc, "//base[@href]"); baseElement != nil {
				if baseURL, err := p.url.Parse(htmlquery.SelectAttr(baseElement, "href")); err == nil {
					base = baseURL
				}
			}
		}
	}

	resolved := base.ResolveReference(ref)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", &errordefs.UserError{Message: fmt.Sprintf("%s selector returned \"%s\", which isn't an http(s) URL", chapterURLField, rawURL)}
	}

	return resolved.String(), nil
}

// selectorPrefixError returns the error for a selector that doesn't start with 'css:' or 'xpath:', or is empty after it.
func selectorPrefixError(selector *HTMLSelector, field string) error {
	if selector.Selector == "" {
		return &errordefs.UserError{Message: fmt.Sprintf("%s selector is empty", field)}
	}
	for _, prefix := range []string{"css:", "xpath:"} {
		if after, ok := strings.CutPrefix(selector.Selector, prefix); ok && strings.TrimSpace(after) == "" {
			return &errordefs.UserError{Message: fmt.Sprintf("%s selector is empty after '%s'", field, prefix)}
		}
	}
	if !strings.HasPrefix(selector.Selector, "css:") && !strings.HasPrefix(selector.Selector, "xpath:") {
		return &errordefs.UserError{Message: fmt.Sprintf("%s selector must start with 'css:' or 'xpath:'", field)}
	}

	return nil
}

// processSelection validates what a selector extracted from the page and applies the selector's regex to it.
// matched tells if the selector matched at least one element in the page.
func processSelection(selection string, matched bool, selector *HTMLSelector, field string) (string, error) {
	if !matched {
		return "", &errordefs.UserError{Message: fmt.Sprintf("%s selector didn't match any element on the page", field)}
	}
	if strings.TrimSpace(selection) == "" {
		if selector.Attribute != "" {
			return "", &errordefs.UserError{Message: fmt.Sprintf("%s selector matched, but the element has no '%s' attribute", field, selector.Attribute)}
		}
		return "", &errordefs.UserError{Message: fmt.Sprintf("%s selector matched, but the element's text is empty", field)}
	}

	if selector.Regex != "" {
		re, err := regexp.Compile(selector.Regex)
		if err != nil {
			return "", &errordefs.UserError{Message: fmt.Sprintf("%s regex is invalid: %s", field, err)}
		}
		if re.NumSubexp() == 0 {
			return "", &errordefs.UserError{Message: fmt.Sprintf(`%s regex has no capture group: wrap the chapter in ( ), e.g. Chapter (\d+)`, field)}
		}
		matches := re.FindStringSubmatch(selection)
		if len(matches) < 2 {
			text := strings.Join(strings.Fields(selection), " ")
			if runes := []rune(text); len(runes) > 80 {
				text = string(runes[:80]) + "..."
			}
			return "", &errordefs.UserError{Message: fmt.Sprintf("%s regex didn't match the text \"%s\"", field, text)}
		}
		selection = matches[1]
	}

	return strings.TrimSpace(selection), nil
}

// getPageUsingHTTPRequest gets the page with a single HTTP GET request.
func getPageUsingHTTPRequest(pageURL string) (*scrapedPage, error) {
	contextError := "error getting page '%s'"

	c := colly.NewCollector(colly.UserAgent(userAgent))
	c.SetRequestTimeout(timeout)

	var page *scrapedPage
	c.OnResponse(func(r *colly.Response) {
		page = &scrapedPage{
			url:         r.Request.URL,
			body:        r.Body,
			contentType: r.Headers.Get("Content-Type"),
		}
	})

	err := c.Visit(pageURL)
	if err != nil {
		reason := err.Error()
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			reason = fmt.Sprintf("timeout after %s", timeout)
		}
		visitError := &errordefs.UserError{Message: fmt.Sprintf("Couldn't open the manga URL: %s", reason)}
		if reason != err.Error() {
			visitError.Err = err
		}
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), util.AddErrorContext("error while visiting manga URL", visitError))
	}
	if page == nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), fmt.Errorf("no response received"))
	}

	return page, nil
}

// getPageUsingBrowser gets the page using a headless browser, so content loaded by JavaScript is included.
// It waits for the selectors' elements to exist in the page before getting the page HTML.
func getPageUsingBrowser(pageURL string, nameSelector, URLSelector *HTMLSelector) (*scrapedPage, error) {
	contextError := "error getting page '%s' using browser"

	ctx, cancel := context.WithTimeout(context.Background(), browserTimeout)
	defer cancel()

	startBrowserError := &errordefs.UserError{Message: `Couldn't start the browser used by "Use Browser"`}
	var u string
	var err error
	if config.GlobalConfigs.API.RodBrowserPath != "" {
		u, err = launcher.New().Context(ctx).Bin(config.GlobalConfigs.API.RodBrowserPath).Headless(true).NoSandbox(true).Set("disable-dev-shm-usage").Set("disable-gpu").Set("no-zygote").Set("single-process").Launch()
		if err != nil {
			startBrowserError.Err = util.AddErrorContext("error launching browser", err)
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), startBrowserError)
		}
	}

	browser := rod.New().ControlURL(u).Context(ctx)
	err = browser.Connect()
	if err != nil {
		startBrowserError.Err = util.AddErrorContext("error connecting to browser", err)
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), startBrowserError)
	}
	defer browser.Close()

	timeoutError := &errordefs.UserError{Message: fmt.Sprintf("Couldn't open the manga URL: timeout after %s (browser)", browserTimeout)}

	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), util.AddErrorContext("error creating browser page", err))
	}
	defer page.Close()
	err = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{UserAgent: userAgent})
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), util.AddErrorContext("error setting user agent", err))
	}
	err = page.Navigate(pageURL)
	if err != nil {
		if ctx.Err() != nil {
			timeoutError.Err = err
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), timeoutError)
		}
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), &errordefs.UserError{Message: fmt.Sprintf("Couldn't open the manga URL: %s", err)})
	}
	err = page.WaitLoad()
	if err != nil {
		timeoutError.Err = util.AddErrorContext("error waiting for page to load", err)
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), timeoutError)
	}

	for _, s := range []struct {
		selector *HTMLSelector
		field    string
	}{{nameSelector, chapterNameField}, {URLSelector, chapterURLField}} {
		selector, field := s.selector, s.field
		if selector == nil {
			continue
		}
		if after, ok := strings.CutPrefix(selector.Selector, "css:"); ok {
			_, err = page.Timeout(timeout).Element(after)
		} else {
			_, err = page.Timeout(timeout).ElementX(strings.TrimPrefix(selector.Selector, "xpath:"))
		}
		if err != nil {
			notFoundError := &errordefs.UserError{Message: fmt.Sprintf("%s selector: element not found within %s (browser)", field, timeout), Err: err}
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), notFoundError)
		}
	}

	html, err := page.HTML()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), util.AddErrorContext("error getting page HTML", err))
	}
	finalURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, pageURL), err)
	}
	if info, err := page.Info(); err == nil {
		if u, err := url.Parse(info.URL); err == nil {
			finalURL = u
		}
	}

	return &scrapedPage{
		url:         finalURL,
		body:        []byte(html),
		contentType: "text/html",
	}, nil
}
