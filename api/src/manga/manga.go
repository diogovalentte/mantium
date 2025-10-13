// Package manga implements the manga and chapter structs and methods
package manga

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gocolly/colly/v2"
	netHTML "golang.org/x/net/html"

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

// InsertIntoDB saves the manga into the database
func (m *Manga) InsertIntoDB() error {
	contextError := "error inserting manga '%s' into DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	mangaID, err := insertMangaIntoDB(m, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}
	m.ID = mangaID

	return nil
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

// DeleteFromDB deletes the manga and its chapters from the database if they exists.
func (m *Manga) DeleteFromDB() error {
	contextError := "error deleting manga '%s' from DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	err = deleteMangaDB(m, tx)
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

// UpdateStatusInDB updates the manga status in the database
func (m *Manga) UpdateStatusInDB(status Status) error {
	contextError := "error updating manga '%s' status to '%d' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, status), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, status), err)
	}

	err = updateMangaStatusDB(m, status, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, status), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, status), err)
	}
	m.Status = status

	return nil
}

func updateMangaStatusDB(m *Manga, status Status, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	err = ValidateStatus(status)
	if err != nil {
		return err
	}

	var result sql.Result
	if m.ID > 0 {
		result, err = tx.Exec(`
            UPDATE mangas
            SET status = $1
            WHERE id = $2;
        `, status, m.ID)
		if err != nil {
			return err
		}
	} else if m.URL != "" {
		result, err = tx.Exec(`
            UPDATE mangas
            SET status = $1
            WHERE url = $2;
        `, status, m.URL)
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
	defer db.Close()

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

// DeleteLastReadChapterFromDB deletes the last read chapter of the manga in the database
func (m *Manga) DeleteLastReadChapterFromDB() error {
	contextError := "error deleting last read chapter '%s' of manga '%s' from DB"

	if m.LastReadChapter == nil {
		return nil
	}

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReadChapter, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReadChapter, m), err)
	}

	err = deleteMangaChapter(m.ID, m.LastReadChapter, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReadChapter, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReadChapter, m), err)
	}
	m.LastReadChapter = nil

	return nil
}

// DeleteLastReleasedChapterFromDB deletes the last released chapter of the manga from the database
func (m *Manga) DeleteLastReleasedChapterFromDB() error {
	contextError := "error deleting last released chapter '%s' of manga '%s' from DB"

	if m.LastReleasedChapter == nil {
		return nil
	}

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReleasedChapter, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReleasedChapter, m), err)
	}

	err = deleteMangaChapter(m.ID, m.LastReleasedChapter, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReleasedChapter, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReleasedChapter, m), err)
	}
	m.LastReleasedChapter = nil

	return nil
}

// DeleteChaptersFromDB deletes the last released and last read chapters of the manga from the database
func (m *Manga) DeleteChaptersFromDB() error {
	contextError := "error deleting last released chapter '%s' and last read chapter '%s' of manga '%s' from DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReleasedChapter, m.LastReadChapter, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReleasedChapter, m.LastReadChapter, m), err)
	}

	if m.LastReleasedChapter != nil {
		err = deleteMangaChapter(m.ID, m.LastReleasedChapter, tx)
		if err != nil {
			tx.Rollback()
			return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReleasedChapter, m.LastReadChapter, m), err)
		}
	}

	if m.LastReadChapter != nil {
		err = deleteMangaChapter(m.ID, m.LastReadChapter, tx)
		if err != nil {
			tx.Rollback()
			return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReleasedChapter, m.LastReadChapter, m), err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m.LastReleasedChapter, m.LastReadChapter, m), err)
	}
	m.LastReleasedChapter = nil
	m.LastReadChapter = nil

	return nil
}

// UpdateNameInDB updates the manga name in the database
func (m *Manga) UpdateNameInDB(name string) error {
	contextError := "error updating manga '%s' name to '%s' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, name), err)
	}
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
	defer db.Close()

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
                last_read_chapter.type AS last_read_chapter_type
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
			&lastReadChapterInternalID, &lastReadChapterUpdatedAt, &lastReadChapterType,
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
                last_read_chapter.type AS last_read_chapter_type
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
			&lastReadChapterInternalID, &lastReadChapterUpdatedAt, &lastReadChapterType,
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
	defer db.Close()

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

// GetMangaDBByID gets a manga from the database by its ID
func GetMangaDBByID(mangaID ID) (*Manga, error) {
	return GetMangaDB(mangaID, "")
}

// GetMangaDBByURL gets a manga from the database by its URL
func GetMangaDBByURL(url string) (*Manga, error) {
	return GetMangaDB(0, url)
}

// GetMangasWithoutMultiMangasDB gets all mangas from the database that are not part of a multimanga or are not custom mangas.
// Used only to get mangas to be transformed into multimangas.
func GetMangasWithoutMultiMangasDB() ([]*Manga, error) {
	contextError := "error getting mangas without multimangas from DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}
	defer db.Close()

	mangas, err := getMangasWithoutMultiMangasFromDB(db)
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	return mangas, nil
}

func getMangasWithoutMultiMangasFromDB(db *sql.DB) ([]*Manga, error) {
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
            last_read_chapter.type AS last_read_chapter_type
        FROM 
            mangas
        LEFT JOIN 
            chapters AS last_released_chapter ON last_released_chapter.id = mangas.last_released_chapter
        LEFT JOIN 
            chapters AS last_read_chapter ON last_read_chapter.id = mangas.last_read_chapter
        WHERE
            mangas.multimanga_id is NULL
            AND mangas.source <> $1
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
			&lastReadChapterInternalID, &lastReadChapterUpdatedAt, &lastReadChapterType,
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
			currentManga.LastReadChapter = &lastReadChapter
		}

		err = validateManga(&currentManga)
		if err != nil {
			return nil, err
		}

		mangas = append(mangas, &currentManga)
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
	defer db.Close()

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
            last_read_chapter.type AS last_read_chapter_type
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
			&lastReadChapterInternalID, &lastReadChapterUpdatedAt, &lastReadChapterType,
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
			currentManga.LastReadChapter = &lastReadChapter
		}

		err = validateManga(&currentManga)
		if err != nil {
			return nil, err
		}

		mangas = append(mangas, &currentManga)
	}

	return mangas, nil
}

func GetLibraryStats() (map[string]int, error) {
	contextError := "error getting library stats"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}
	defer db.Close()

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

	query = `
        SELECT
            COUNT(*) AS total_mangas, 
            status
        FROM
            mangas
        WHERE
            source = $1
        GROUP BY
            status
        ;
    `
	rows, err = db.Query(query, CustomMangaSource)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

		val, ok := stats[status]
		if ok {
			stats[status] = val + totalMangas
		} else {
			stats[status] = totalMangas
		}
	}

	var unread, total, read int

	query = `
        WITH unread_mm AS (
            SELECT
                COUNT(mm.*) AS total_mangas
            FROM
                multimangas AS mm
            LEFT JOIN 
                chapters AS c ON c.id = mm.last_read_chapter
            JOIN
                mangas AS m ON m.id = mm.current_manga
            LEFT JOIN
                chapters AS cc ON cc.id = m.last_released_chapter 
            WHERE
                cc.chapter <> c.chapter
        ),
        no_last_read_chapter_mm AS (
            SELECT
                COUNT(*) AS total_mangas
            FROM
                multimangas
            WHERE
                last_read_chapter IS NULL
        ),
        total_mm AS (
            SELECT
                COUNT(*)
            FROM
                multimangas
        ),
        unread_custom_mangas AS (
            SELECT
                COUNT(*) AS total_mangas
            FROM
                mangas
            WHERE
                last_released_chapter IS NULL
                AND last_read_chapter IS NOT NULL
                AND source = $1
        ),
        total_custom_mangas AS (
            SELECT
                COUNT(*)
            FROM
                mangas
            WHERE
                source = $1
        )

        SELECT
            (unread_mm.total_mangas + no_last_read_chapter_mm.total_mangas + unread_custom_mangas.total_mangas) AS unread_mangas,
            (total_mm.count + total_custom_mangas.count) AS total_mangas,
            ((total_mm.count - unread_mm.total_mangas - no_last_read_chapter_mm.total_mangas) + (total_custom_mangas.count - unread_custom_mangas.total_mangas)) AS read_mangas
        FROM
            unread_mm, no_last_read_chapter_mm, unread_custom_mangas, total_mm, total_custom_mangas
        ;
    `
	err = db.QueryRow(query, CustomMangaSource).Scan(
		&unread, &total, &read,
	)
	if err != nil {
		return nil, err
	}

	stats["Unread"] = unread
	stats["Total"] = total
	stats["Read"] = read

	return stats, nil
}

func (m *Manga) UpdateSourceInDB(source string) error {
	contextError := "error updating manga '%s' source to '%s' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, source), err)
	}
	defer db.Close()

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
	defer db.Close()

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

	err := ValidateStatus(m.Status)
	if err != nil {
		return util.AddErrorContext(contextError, err)
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

// FilterUnreadChapterMangas filters a list of mangas to return
// mangas where the last released chapter is different from the
// last read chapter
func FilterUnreadChapterMangas(mangas []*Manga) []*Manga {
	unreadChapterMangas := []*Manga{}

	for _, manga := range mangas {
		if (manga.LastReleasedChapter != nil && manga.LastReadChapter == nil) || (manga.LastReleasedChapter == nil && manga.LastReadChapter != nil) {
			unreadChapterMangas = append(unreadChapterMangas, manga)
		} else if manga.LastReleasedChapter != nil && manga.LastReadChapter != nil {
			if manga.LastReleasedChapter.Chapter != manga.LastReadChapter.Chapter {
				unreadChapterMangas = append(unreadChapterMangas, manga)
			}
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

	chapter := &Chapter{
		UpdatedAt: time.Now().Local().Truncate(time.Second),
		Type:      1,
	}
	var chapterName string
	var chapterURL string
	var err error
	if nameSelector != nil {
		if useBrowser {
			chapterName, err = getSelectorFromPageUsingBrowser(mangaURL, nameSelector)
			if err != nil {
				return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, nameSelector, URLSelector, useBrowser), err)
			}
		} else {
			chapterName, err = getSelectorFromPageUsingHTTPRequest(mangaURL, nameSelector)
			if err != nil {
				return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, nameSelector, URLSelector, useBrowser), err)
			}
		}
		chapter.Chapter = chapterName
		chapter.Name = "Chapter " + chapterName
	}
	if URLSelector != nil {
		if useBrowser {
			chapterURL, err = getSelectorFromPageUsingBrowser(mangaURL, URLSelector)
			if err != nil {
				return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, nameSelector, URLSelector, useBrowser), err)
			}
		} else {
			chapterURL, err = getSelectorFromPageUsingHTTPRequest(mangaURL, URLSelector)
			if err != nil {
				return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaURL, nameSelector, URLSelector, useBrowser), err)
			}
		}
		chapter.URL = chapterURL
		if !strings.HasPrefix(chapter.URL, "http") {
			chapter.URL = strings.TrimRight(mangaURL, "/") + "/" + strings.TrimLeft(chapter.URL, "/")
		}
		if chapter.Chapter == "" {
			chapter.Chapter = "?"
			chapter.Name = "Chapter ?"
		}
	}

	return chapter, nil
}

var (
	userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"
	timeout   = time.Second * 15
)

func getSelectorFromPageUsingBrowser(url string, selector *HTMLSelector) (string, error) {
	contextError := "error getting selector '%s' from page '%s'"

	browser := rod.New()
	err := browser.Connect()
	if err != nil {
		return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error connecting to browser", err))
	}
	defer browser.MustClose()

	page := browser.MustPage(url).MustSetUserAgent(&proto.NetworkSetUserAgentOverride{UserAgent: userAgent})
	defer page.MustClose()

	var selection string
	if after, ok := strings.CutPrefix(selector.Selector, "css:"); ok {
		el, err := page.Timeout(timeout).Element(after)
		if err != nil {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error finding element with selector", err))
		}
		err = el.Timeout(timeout).WaitVisible()
		if err != nil {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error waiting for element to be visible", err))
		}
		html, err := page.HTML()
		if err != nil {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error getting page HTML", err))
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error creating goquery document from page HTML", err))
		}
		var s *goquery.Selection
		if selector.GetFirst {
			s = doc.Find(after).First()
		} else {
			s = doc.Find(after).Last()
		}
		if selector.Attribute != "" {
			selection = s.AttrOr(selector.Attribute, "")
		} else {
			selection = s.Text()
		}
	} else if after, ok := strings.CutPrefix(selector.Selector, "xpath:"); ok {
		el, err := page.Timeout(timeout).ElementX(after)
		if err != nil {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error finding element with selector", err))
		}
		err = el.Timeout(timeout).WaitVisible()
		if err != nil {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error waiting for element to be visible", err))
		}
		html, err := page.HTML()
		if err != nil {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error getting page HTML", err))
		}

		doc, err := htmlquery.Parse(strings.NewReader(html))
		if err != nil {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error creating xpath document from page HTML", err))
		}

		nodes := htmlquery.Find(doc, after)
		if len(nodes) == 0 {
			selection = ""
		} else {
			var node *netHTML.Node
			if selector.GetFirst {
				node = nodes[0]
			} else {
				node = nodes[len(nodes)-1]
			}
			if selector.Attribute != "" {
				selection = htmlquery.SelectAttr(node, selector.Attribute)
			} else {
				selection = htmlquery.InnerText(node)
			}
		}
	} else {
		return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), fmt.Errorf("selector should start with 'css:' or 'xpath:', instead it's '%s'", url))
	}

	if selection == "" {
		return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), fmt.Errorf("selector not found in the page or is empty"))
	}

	if selector.Regex != "" {
		re, err := regexp.Compile(selector.Regex)
		if err != nil {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error compiling regex", err))
		}
		matches := re.FindStringSubmatch(selection)
		if len(matches) < 2 {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), fmt.Errorf("regex did not match"))
		}
		selection = matches[1]
	}
	selection = strings.TrimSpace(selection)

	return selection, nil
}

func getSelectorFromPageUsingHTTPRequest(url string, selector *HTMLSelector) (string, error) {
	contextError := "error getting selector '%s' from page '%s'"

	if selector == nil {
		return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), fmt.Errorf("manga.HTMLSelector is nil"))
	}
	if selector.Selector == "" {
		return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), fmt.Errorf("manga.HTMLSelector.Selector is empty"))
	}

	c := colly.NewCollector(colly.UserAgent(userAgent))

	var selection string
	if after, ok := strings.CutPrefix(selector.Selector, "css:"); ok {
		c.OnHTML(after, func(e *colly.HTMLElement) {
			if selector.GetFirst && selection != "" {
				return
			}

			if selector.Attribute != "" {
				selection = e.Attr(selector.Attribute)
			} else {
				selection = e.Text
			}
		})
	} else if after, ok := strings.CutPrefix(selector.Selector, "xpath:"); ok {
		c.OnXML(after, func(e *colly.XMLElement) {
			if selector.GetFirst && selection != "" {
				return
			}

			if selector.Attribute != "" {
				selection = e.Attr(selector.Attribute)
			} else {
				selection = e.Text
			}
		})
	} else {
		return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), fmt.Errorf("selector should start with 'css:' or 'xpath:', instead it's '%s'", url))
	}

	err := c.Visit(url)
	if err != nil {
		return "", util.AddErrorContext(contextError, util.AddErrorContext("error while visiting manga URL", err))
	}

	if selection == "" {
		return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), fmt.Errorf("selector not found in the page or is empty"))
	}

	if selector.Regex != "" {
		re, err := regexp.Compile(selector.Regex)
		if err != nil {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), util.AddErrorContext("error compiling regex", err))
		}
		matches := re.FindStringSubmatch(selection)
		if len(matches) < 2 {
			return "", util.AddErrorContext(fmt.Sprintf(contextError, selector, url), fmt.Errorf("regex did not match"))
		}
		selection = matches[1]
	}
	selection = strings.TrimSpace(selection)

	return selection, nil
}
