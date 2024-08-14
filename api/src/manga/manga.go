// Package manga implements the manga and chapter structs and methods
package manga

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/util"
)

type (
	// ID is the ID of the manga in the database, should not be manually set
	ID int
	// Status is the status of the manga, it can be:
	// 1 - Reading
	// 2 - Completed
	// 3 - On Hold
	// 4 - Dropped
	// 5 - Plan to Read
	Status int
)

// Manga is the interface for a manga
type Manga struct {
	ID ID
	// Source is the source of the manga, usually the domain of the website
	Source string
	// URL is the URL of the manga
	URL string
	// Name is the name of the manga
	Name   string
	Status Status
	// CoverImg is the cover image of the manga
	CoverImg []byte
	// CoverImgResized is true if the cover image was resized
	CoverImgResized bool
	// CoverImgURL is the URL of the cover image
	CoverImgURL string
	// CoverImgFixed is true if the cover image is fixed. If true, the cover image will not be updated when updating the manga metadata.
	// It's used for when the cover image is manually set by the user.
	CoverImgFixed bool
	// PreferredGroup is the preferred group that translates (and more) the manga
	// Not all sources have multiple groups
	PreferredGroup string
	// LastReleasedChapter is the last chapter released by the source
	LastReleasedChapter *Chapter
	// LastReadChapter is the last chapter read by the user
	LastReadChapter *Chapter
}

func (m Manga) String() string {
	return fmt.Sprintf("Manga{ID: %d, Source: %s, URL: %s, Name: %s, Status: %d, CoverImg: []byte, CoverImgResized: %v, CoverImgURL: %s, CoverImgFixed: %v, PreferredGroup: %s, LastReleasedChapter: %s, LastReadChapter: %s}",
		m.ID, m.Source, m.URL, m.Name, m.Status, m.CoverImgResized, m.CoverImgURL, m.CoverImgFixed, m.PreferredGroup, m.LastReleasedChapter, m.LastReadChapter)
}

// InsertIntoDB saves the manga into the database
func (m *Manga) InsertIntoDB() (ID, error) {
	contextError := "error inserting manga '%s' into DB"

	db, err := db.OpenConn()
	if err != nil {
		return -1, util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return -1, util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	mangaID, err := insertMangaIntoDB(m, tx)
	if err != nil {
		tx.Rollback()
		return -1, util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return -1, util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	return mangaID, nil
}

func insertMangaIntoDB(m *Manga, tx *sql.Tx) (ID, error) {
	err := validateManga(m)
	if err != nil {
		return -1, err
	}

	var mangaID ID
	err = tx.QueryRow(`
        INSERT INTO mangas
            (source, url, name, status, cover_img, cover_img_resized, cover_img_url, cover_img_fixed, preferred_group)
        VALUES
            ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING
            id;
    `, m.Source, m.URL, m.Name, m.Status, m.CoverImg, m.CoverImgResized, m.CoverImgURL, m.CoverImgFixed, m.PreferredGroup).Scan(&mangaID)
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "mangas_pkey"` {
			return -1, errordefs.ErrMangaAlreadyInDB
		}
		return -1, err
	}

	if m.LastReleasedChapter != nil {
		chapterID, err := insertChapterDB(m.LastReleasedChapter, mangaID, tx)
		if err != nil {
			if err.Error() == `pq: duplicate key value violates unique constraint "chapters_pkey"` {
				return -1, fmt.Errorf("last released chapter of the manga you're trying to add already exists in DB")
			}
			return -1, err
		}

		_, err = tx.Exec(`
            UPDATE mangas
            SET last_released_chapter = $1
            WHERE id = $2;
        `, chapterID, mangaID)
		if err != nil {
			return -1, err
		}
	}
	if m.LastReadChapter != nil {
		chapterID, err := insertChapterDB(m.LastReadChapter, mangaID, tx)
		if err != nil {
			if err.Error() == `pq: duplicate key value violates unique constraint "chapters_pkey"` {
				return -1, fmt.Errorf("last read chapter of the manga you're trying to add already exists in DB")
			}
			return -1, err
		}

		_, err = tx.Exec(`
            UPDATE mangas
            SET last_read_chapter = $1
            WHERE id = $2;
        `, chapterID, mangaID)
		if err != nil {
			return -1, err
		}
	}

	// if the manga has chapters, also update the last_released_chapter and last_read_chapter
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
		return errordefs.ErrMangaDoesntHaveIDAndURL
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
	contextError := "error updating manga '%s' status in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	err = updateMangaStatusDB(m, status, tx)
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

func updateMangaStatusDB(m *Manga, status Status, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	err = validateStatus(status)
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
		return errordefs.ErrMangaDoesntHaveIDAndURL
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMangaNotFoundDB
	}
	m.Status = status

	return nil
}

// UpsertChapterInDB updates the last read/released chapter in the database
// The chapter.Type field must be set
func (m *Manga) UpsertChapterInDB(chapter *Chapter) error {
	contextError := "error upserting chapter '%s' to manga '%s' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, m), err)
	}

	err = upsertMangaChapter(m, chapter, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, m), err)
	}
	m.LastReleasedChapter = chapter

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

	err = updateMangaName(m, name, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, name), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, name), err)
	}
	m.Name = name

	return nil
}

func updateMangaName(m *Manga, name string, tx *sql.Tx) error {
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
		return errordefs.ErrMangaDoesntHaveIDAndURL
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
		return errordefs.ErrMangaDoesntHaveIDAndURL
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
		err = upsertMangaChapter(m, m.LastReleasedChapter, tx)
		if err != nil {
			return err
		}
	}

	err = updateMangaName(m, m.Name, tx)
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

	var mangaGet Manga
	mangaGet.ID = mangaID
	mangaGet.URL = mangaURL

	err = getMangaFromDB(&mangaGet, db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaID, mangaURL), errordefs.ErrMangaNotFoundDB)
		}
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, mangaID, mangaURL), err)
	}

	return &mangaGet, nil
}

func getMangaFromDB(m *Manga, db *sql.DB) error {
	var lastReleasedChapterID sql.NullInt64
	var lastReadChapterID sql.NullInt64
	if m.ID > 0 {
		query := `
            SELECT
                id, source, url, name, status, cover_img, cover_img_resized, cover_img_url, cover_img_fixed, preferred_group, last_released_chapter, last_read_chapter
            FROM
                mangas
            WHERE
                id = $1;
        `
		err := db.QueryRow(query, m.ID).Scan(&m.ID, &m.Source, &m.URL, &m.Name, &m.Status, &m.CoverImg, &m.CoverImgResized, &m.CoverImgURL, &m.CoverImgFixed, &m.PreferredGroup, &lastReleasedChapterID, &lastReadChapterID)
		if err != nil {
			return err
		}
	} else if m.URL != "" {
		query := `
            SELECT
                id, source, url, name, status, cover_img, cover_img_resized, cover_img_url, cover_img_fixed, preferred_group, last_released_chapter, last_read_chapter
            FROM
                mangas
            WHERE
                url = $1;
        `
		err := db.QueryRow(query, m.URL).Scan(&m.ID, &m.Source, &m.URL, &m.Name, &m.Status, &m.CoverImg, &m.CoverImgResized, &m.CoverImgURL, &m.CoverImgFixed, &m.PreferredGroup, &lastReleasedChapterID, &lastReadChapterID)
		if err != nil {
			return err
		}
	} else {
		return errordefs.ErrMangaDoesntHaveIDAndURL
	}

	if lastReleasedChapterID.Valid && lastReleasedChapterID.Int64 != 0 {
		chapter, err := getChapterDB(int(lastReleasedChapterID.Int64), db)
		if err != nil {
			return err
		}
		m.LastReleasedChapter = chapter
	}
	if lastReadChapterID.Valid && lastReadChapterID.Int64 != 0 {
		chapter, err := getChapterDB(int(lastReadChapterID.Int64), db)
		if err != nil {
			return err
		}
		m.LastReadChapter = chapter
	}

	err := validateManga(m)
	if err != nil {
		return err
	}

	return nil
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

// GetMangasDB gets all mangas from the database
func GetMangasDB() ([]*Manga, error) {
	contextError := "error getting mangas from DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}
	defer db.Close()

	mangas, err := getMangasFromDB(db)
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	return mangas, nil
}

func getMangasFromDB(db *sql.DB) ([]*Manga, error) {
	query := `
        SELECT
            id, source, url, name, status, cover_img, cover_img_resized, cover_img_url, cover_img_fixed, preferred_group, last_released_chapter, last_read_chapter
        FROM
            mangas;
    `
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangas []*Manga
	for rows.Next() {
		var manga Manga
		var lastReleasedChapterID sql.NullInt64
		var lastReadChapterID sql.NullInt64
		err = rows.Scan(&manga.ID, &manga.Source, &manga.URL, &manga.Name, &manga.Status, &manga.CoverImg, &manga.CoverImgResized, &manga.CoverImgURL, &manga.CoverImgFixed, &manga.PreferredGroup, &lastReleasedChapterID, &lastReadChapterID)
		if err != nil {
			return nil, err
		}

		if lastReleasedChapterID.Valid && lastReleasedChapterID.Int64 != 0 {
			chapter, err := getChapterDB(int(lastReleasedChapterID.Int64), db)
			if err != nil {
				return nil, err
			}
			manga.LastReleasedChapter = chapter
		}
		if lastReadChapterID.Valid && lastReadChapterID.Int64 != 0 {
			chapter, err := getChapterDB(int(lastReadChapterID.Int64), db)
			if err != nil {
				return nil, err
			}
			manga.LastReadChapter = chapter
		}

		err := validateManga(&manga)
		if err != nil {
			return nil, err
		}

		mangas = append(mangas, &manga)
	}

	return mangas, nil
}

// validateManga should be used every time the API interacts with the mangas table in the database
func validateManga(m *Manga) error {
	contextError := "error validating manga"

	err := validateStatus(m.Status)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	if m.Source == "" {
		return util.AddErrorContext(contextError, fmt.Errorf("manga source is empty"))
	}
	if m.URL == "" {
		return util.AddErrorContext(contextError, fmt.Errorf("manga URL is empty"))
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

func validateStatus(status Status) error {
	if status < 1 || status > 5 {
		return fmt.Errorf("manga status should be >= 1 && <= 5, instead it's %d", status)
	}

	return nil
}

// FilterUnreadChapterMangas filters a list of mangas to return
// mangas where the last released chapter is different from the
// last read chapter
func FilterUnreadChapterMangas(mangas []*Manga) []*Manga {
	unreadChapterMangas := []*Manga{}

	for _, manga := range mangas {
		if manga.LastReleasedChapter != nil && manga.LastReadChapter == nil {
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
		if mangas[i].LastReleasedChapter == nil {
			return false
		}
		if mangas[j].LastReleasedChapter == nil {
			return true
		}
		return mangas[i].LastReleasedChapter.UpdatedAt.After(mangas[j].LastReleasedChapter.UpdatedAt)
	})
}
