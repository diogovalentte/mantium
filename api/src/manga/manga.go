// Package manga implements the manga and chapter structs and methods
package manga

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/diogovalentte/mantium/api/src/db"
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
	// PreferredGroup is the preferred group that translates (and more) the manga
	// Not all sources have multiple groups
	PreferredGroup string
	// LastUploadChapter is the last chapter uploaded to the source
	LastUploadChapter *Chapter
	// LastReadChapter is the last chapter read by the user
	LastReadChapter *Chapter
}

// InsertDB saves the manga into the database
func (m *Manga) InsertDB() (ID, error) {
	contextError := "error inserting manga into DB"

	db, err := db.OpenConn()
	if err != nil {
		return -1, util.AddErrorContext(err, contextError)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return -1, util.AddErrorContext(err, contextError)
	}

	mangaID, err := insertMangaDB(m, tx)
	if err != nil {
		tx.Rollback()
		return -1, util.AddErrorContext(err, contextError)
	}

	err = tx.Commit()
	if err != nil {
		return -1, util.AddErrorContext(err, contextError)
	}

	return mangaID, nil
}

func insertMangaDB(m *Manga, tx *sql.Tx) (ID, error) {
	err := validateManga(m)
	if err != nil {
		return -1, err
	}

	var mangaID ID
	err = tx.QueryRow(`
        INSERT INTO mangas
            (source, url, name, status, cover_img, cover_img_resized, cover_img_url, preferred_group)
        VALUES
            ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING
            id;
    `, m.Source, m.URL, m.Name, m.Status, m.CoverImg, m.CoverImgResized, m.CoverImgURL, m.PreferredGroup).Scan(&mangaID)
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "mangas_pkey"` {
			return -1, fmt.Errorf("manga already exists in DB")
		}
		return -1, err
	}

	if m.LastUploadChapter != nil {
		chapterID, err := insertChapterDB(m.LastUploadChapter, mangaID, tx)
		if err != nil {
			if err.Error() == `pq: duplicate key value violates unique constraint "chapters_pkey"` {
				return -1, fmt.Errorf("last upload chapter of the manga you're trying to add already exists in DB")
			}
			return -1, err
		}

		_, err = tx.Exec(`
            UPDATE mangas
            SET last_upload_chapter = $1
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

	// if the manga has chapters, also update the last_upload_chapter and last_read_chapter
	return mangaID, nil
}

// DeleteDB deletes the manga and its chapters from the database
func (m *Manga) DeleteDB() error {
	contextError := "error deleting manga from DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	err = deleteMangaDB(m, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(err, contextError)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(err, contextError)
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
		return fmt.Errorf("manga doesn't have an ID or URL")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("manga not found in DB")
	}

	return nil
}

// UpdateStatus updates the manga status in the database
func (m *Manga) UpdateStatus(status Status) error {
	contextError := "error updating manga status in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	err = updateMangaStatusDB(m, status, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(err, contextError)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	return nil
}

func updateMangaStatusDB(m *Manga, status Status, tx *sql.Tx) error {
	err := validateManga(m)
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
		return fmt.Errorf("manga doesn't have an ID or URL")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("manga not found in DB")
	}
	m.Status = status

	return nil
}

// UpsertChapter updates the last read/upload chapter in the database
// The chapter.Type field must be set
func (m *Manga) UpsertChapter(chapter *Chapter) error {
	contextError := "error updating manga chapter in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	err = upsertMangaChapter(m, chapter, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(err, contextError)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}
	m.LastUploadChapter = chapter

	return nil
}

// UpdateName updates the manga name in the database
func (m *Manga) UpdateName(name string) error {
	contextError := "error updating manga name in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	err = updateMangaName(m, name, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(err, contextError)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(err, contextError)
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
		return fmt.Errorf("manga doesn't have an ID or URL")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("manga not found in DB")
	}

	return nil
}

// UpdateCoverImg updates the manga cover image in the database
func (m *Manga) UpdateCoverImg(coverImg []byte, coverImgResized bool, coverImgURL string) error {
	contextError := "error updating manga cover image in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	err = updateMangaCoverImg(m, coverImg, coverImgResized, coverImgURL, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(err, contextError)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}
	m.CoverImg = coverImg
	m.CoverImgResized = coverImgResized
	m.CoverImgURL = coverImgURL

	return nil
}

func updateMangaCoverImg(m *Manga, coverImg []byte, coverImgResized bool, coverImgURL string, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	var result sql.Result
	if m.ID > 0 {
		result, err = tx.Exec(`
            UPDATE mangas
            SET cover_img = $1, cover_img_resized = $2, cover_img_url = $3
            WHERE id = $4;
        `, coverImg, coverImgResized, coverImgURL, m.ID)
		if err != nil {
			return err
		}
	} else if m.URL != "" {
		result, err = tx.Exec(`
            UPDATE mangas
            SET cover_img = $1, cover_img_resized = $2, cover_img_url = $3
            WHERE url = $4;
        `, coverImg, coverImgResized, coverImgURL, m.URL)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("manga doesn't have an ID or URL")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("manga not found in DB")
	}

	return nil
}

// UpdateMangaMetadataDB updates the manga metadata in the database
// It updates: the last upload chapter (and its metadata), the manga name and cover image
// The manga argument should have the ID or URL set to identify which manga to update
// The other fields of the manga argument will be the new values for the manga in the database
func UpdateMangaMetadataDB(m *Manga) error {
	contextError := "error updating manga metadata in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	err = updateMangaMetadata(m, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(err, contextError)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	return nil
}

func updateMangaMetadata(m *Manga, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	err = upsertMangaChapter(m, m.LastUploadChapter, tx)
	if err != nil {
		return err
	}

	err = updateMangaName(m, m.Name, tx)
	if err != nil {
		return err
	}

	err = updateMangaCoverImg(m, m.CoverImg, m.CoverImgResized, m.CoverImgURL, tx)
	if err != nil {
		return err
	}

	return nil
}

// GetMangaDBByID gets a manga from the database by its ID
func GetMangaDBByID(mangaID ID) (*Manga, error) {
	return GetMangaDB(mangaID, "")
}

// GetMangaDBByURL gets a manga from the database by its URL
func GetMangaDBByURL(url string) (*Manga, error) {
	return GetMangaDB(0, url)
}

// GetMangaDB gets a manga from the database by its ID or URL
func GetMangaDB(mangaID ID, mangaURL string) (*Manga, error) {
	contextError := "error getting manga from DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}
	defer db.Close()

	var mangaGet Manga
	mangaGet.ID = mangaID
	mangaGet.URL = mangaURL

	err = getMangaFromDB(&mangaGet, db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.AddErrorContext(fmt.Errorf("manga not found in DB"), contextError)
		}
		return nil, util.AddErrorContext(err, contextError)
	}

	return &mangaGet, nil
}

func getMangaFromDB(m *Manga, db *sql.DB) error {
	var lastUploadChapterID sql.NullInt64
	var lastReadChapterID sql.NullInt64
	if m.ID > 0 {
		query := `
            SELECT
                id, source, url, name, status, cover_img, cover_img_resized, cover_img_url, preferred_group, last_upload_chapter, last_read_chapter
            FROM
                mangas
            WHERE
                id = $1;
        `
		err := db.QueryRow(query, m.ID).Scan(&m.ID, &m.Source, &m.URL, &m.Name, &m.Status, &m.CoverImg, &m.CoverImgResized, &m.CoverImgURL, &m.PreferredGroup, &lastUploadChapterID, &lastReadChapterID)
		if err != nil {
			return err
		}
	} else if m.URL != "" {
		query := `
            SELECT
                id, source, url, name, status, cover_img, cover_img_resized, cover_img_url, preferred_group, last_upload_chapter, last_read_chapter
            FROM
                mangas
            WHERE
                url = $1;
        `
		err := db.QueryRow(query, m.URL).Scan(&m.ID, &m.Source, &m.URL, &m.Name, &m.Status, &m.CoverImg, &m.CoverImgResized, &m.CoverImgURL, &m.PreferredGroup, &lastUploadChapterID, &lastReadChapterID)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("manga doesn't have an ID or URL")
	}

	if lastUploadChapterID.Valid && lastUploadChapterID.Int64 != 0 {
		chapter, err := getChapterDB(int(lastUploadChapterID.Int64), db)
		if err != nil {
			return err
		}
		m.LastUploadChapter = chapter
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
	db, err := db.OpenConn()
	if err != nil {
		return -1, err
	}
	defer db.Close()

	var mangaID ID
	err = db.QueryRow(`
        SELECT id
        FROM mangas
        WHERE url = $1;
    `, url).Scan(&mangaID)
	if err != nil {
		return -1, err
	}

	return mangaID, nil
}

// GetMangasDB gets all mangas from the database
func GetMangasDB() ([]*Manga, error) {
	contextError := "error getting mangas from DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}
	defer db.Close()

	mangas, err := getMangasFromDB(db)
	if err != nil {
		return nil, util.AddErrorContext(err, contextError)
	}

	return mangas, nil
}

func getMangasFromDB(db *sql.DB) ([]*Manga, error) {
	query := `
        SELECT
            id, source, url, name, status, cover_img, cover_img_resized, cover_img_url, preferred_group, last_upload_chapter, last_read_chapter
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
		var lastUploadChapterID sql.NullInt64
		var lastReadChapterID sql.NullInt64
		err = rows.Scan(&manga.ID, &manga.Source, &manga.URL, &manga.Name, &manga.Status, &manga.CoverImg, &manga.CoverImgResized, &manga.CoverImgURL, &manga.PreferredGroup, &lastUploadChapterID, &lastReadChapterID)
		if err != nil {
			return nil, err
		}

		if lastUploadChapterID.Valid && lastUploadChapterID.Int64 != 0 {
			chapter, err := getChapterDB(int(lastUploadChapterID.Int64), db)
			if err != nil {
				return nil, err
			}
			manga.LastUploadChapter = chapter
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

// validateManga should be used every time the API interacts with
// the mangas table in the database
func validateManga(m *Manga) error {
	if m.Status < 1 || m.Status > 5 {
		return fmt.Errorf("manga status should be >= 1 && <= 5")
	}

	if m.LastUploadChapter != nil {
		err := validateChapter(m.LastUploadChapter)
		if err != nil {
			return err
		}
	}
	if m.LastReadChapter != nil {
		err := validateChapter(m.LastReadChapter)
		if err != nil {
			return err
		}
	}

	return nil
}

// FilterUnreadChapterMangas filters a list of mangas to return
// mangas where the last upload chapter is different from the
// last read chapter
func FilterUnreadChapterMangas(mangas []*Manga) []*Manga {
	unreadChapterMangas := []*Manga{}

	for _, manga := range mangas {
		if manga.LastUploadChapter.Chapter != manga.LastReadChapter.Chapter {
			unreadChapterMangas = append(unreadChapterMangas, manga)
		}
	}

	return unreadChapterMangas
}

// SortMangasByLastUploadChapterUpdatedAt sorts a list of mangas
// by their last upload chapter updated at property, desc
func SortMangasByLastUploadChapterUpdatedAt(mangas []*Manga) {
	sort.Slice(mangas, func(i, j int) bool {
		return mangas[i].LastUploadChapter.UpdatedAt.After(mangas[j].LastUploadChapter.UpdatedAt)
	})
}
