package manga

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/util"
)

// MultiManga is an interface to the same manga from multiple sources.
// It is used to get the manga with the lastest chapter.
type MultiManga struct {
	CurrentManga    *Manga
	LastReadChapter *Chapter
	Mangas          []*Manga
	// CoverImgURL is the URL of the cover image
	CoverImgURL string
	// CoverImg is the cover image of the multimanga
	CoverImg []byte
	ID       ID
	Status   Status // All mangas in the multimanga should have the same status
	// CoverImgResized is true if the cover image was resized
	CoverImgResized bool
	// CoverImgFixed is true if the cover image is fixed. If false (default) the current manga's cover image should be used.
	// Else, use the multimanga's cover image fields.
	// It's used for when the cover image is manually set by the user.
	CoverImgFixed bool
}

func (mm MultiManga) String() string {
	returnStr := fmt.Sprintf("MultiManga{ID: %d, Status: %d, CoverImg: []byte, CoverImgResized: %v, CoverImgURL: %s, CoverImgFixed: %v, LastReadChapter: %s, CurrentManga: %s, Mangas: [",
		mm.ID, mm.Status, mm.CoverImgResized, mm.CoverImgURL, mm.CoverImgFixed, mm.LastReadChapter, mm.CurrentManga)

	for _, manga := range mm.Mangas {
		returnStr += manga.String() + ", "
	}
	returnStr = strings.TrimSuffix(returnStr, ", ")
	returnStr += "]}"

	return returnStr
}

// InsertIntoDB creates the multimanga and its mangas into the database.
// It's a method to create a multimanga from scratch.
func (mm *MultiManga) InsertIntoDB() error {
	contextError := "error creating multimanga '%s' into DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}

	err = insertMultiMangaIntoDB(mm, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}

	return nil
}

// Also creates the multimanga manga list's mangas and set current manga.
func insertMultiMangaIntoDB(mm *MultiManga, tx *sql.Tx) error {
	err := validateMultiManga(mm)
	if err != nil {
		return err
	}

	var multiMangaID ID
	err = tx.QueryRow(`
        INSERT INTO multimangas
            (status, cover_img, cover_img_resized, cover_img_url, cover_img_fixed)
        VALUES
            ($1, $2, $3, $4, $5)
        RETURNING
            id;
    `, mm.Status, mm.CoverImg, mm.CoverImgResized, mm.CoverImgURL, mm.CoverImgFixed).Scan(&multiMangaID)
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "multimangas_pkey"` {
			return errordefs.ErrMultiMangaAlreadyInDB
		}
		return err
	}
	mm.ID = multiMangaID

	for _, manga := range mm.Mangas {
		manga.MultiMangaID = multiMangaID
		manga.LastReadChapter = nil
		manga.CoverImgFixed = false
		mangaID, err := insertMangaIntoDB(manga, tx)
		if err != nil {
			return err
		}
		manga.ID = mangaID
	}

	err = updateMultiMangaCurrentManga(mm, tx)
	if err != nil {
		return err
	}

	if mm.LastReadChapter != nil {
		err := upsertMultiMangaChapter(multiMangaID, mm.LastReadChapter, tx)
		if err != nil {
			if err.Error() == `pq: duplicate key value violates unique constraint "chapters_pkey"` {
				return fmt.Errorf("last read chapter of the multimanga you're trying to add already exists in DB")
			}
			return err
		}
	}

	return nil
}

// DeleteFromDB deletes the multimanga, its mangas, and its chapter from the database
func (mm *MultiManga) DeleteFromDB() error {
	contextError := "error deleting multimanga '%s' from DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}

	err = deleteMultiMangaDB(mm, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}

	return nil
}

// Will delete chapters and mangas
// because of the foreign key constraint ON DELETE CASCADE
func deleteMultiMangaDB(mm *MultiManga, tx *sql.Tx) error {
	err := validateMultiManga(mm)
	if err != nil {
		return err
	}

	result, err := tx.Exec(`
        DELETE FROM multimangas
        WHERE id = $1;
    `, mm.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMultiMangaNotFoundDB
	}

	return nil
}

// UpdateStatusInDB updates the multimanga status in the database
func (mm *MultiManga) UpdateStatusInDB(status Status) error {
	contextError := "error updating multimanga '%s' status in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}

	err = updateMultiMangaStatusDB(mm, status, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}
	mm.Status = status

	return nil
}

func updateMultiMangaStatusDB(mm *MultiManga, status Status, tx *sql.Tx) error {
	err := validateMultiManga(mm)
	if err != nil {
		return err
	}

	err = ValidateStatus(status)
	if err != nil {
		return err
	}

	var result sql.Result
	result, err = tx.Exec(`
        UPDATE multimangas
        SET status = $1
        WHERE id = $2;
    `, status, mm.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMultiMangaNotFoundDB
	}

	return nil
}

// UpdateCoverImgInDB updates the multimanga cover image in the database.
// It doesn't care if the cover image is fixed or not.
func (mm *MultiManga) UpdateCoverImgInDB(coverImg []byte, coverImgResized bool, coverImgURL string) error {
	contextError := "error updating multimanga '%s' cover image to URL '%s' or/and image with '%d' bytes in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, coverImgURL, len(coverImg)), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, coverImgURL, len(coverImg)), err)
	}

	err = updateMultiMangaCoverImg(mm, coverImg, coverImgResized, coverImgURL, mm.CoverImgFixed, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, coverImgURL, len(coverImg)), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, coverImgURL, len(coverImg)), err)
	}
	mm.CoverImg = coverImg
	mm.CoverImgResized = coverImgResized
	mm.CoverImgURL = coverImgURL

	return nil
}

func updateMultiMangaCoverImg(mm *MultiManga, coverImg []byte, coverImgResized bool, coverImgURL string, fixed bool, tx *sql.Tx) error {
	err := validateMultiManga(mm)
	if err != nil {
		return err
	}

	var result sql.Result
	result, err = tx.Exec(`
        UPDATE multimangas
        SET cover_img = $1, cover_img_resized = $2, cover_img_url = $3, cover_img_fixed = $4
        WHERE id = $5;
    `, coverImg, coverImgResized, coverImgURL, fixed, mm.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMultiMangaNotFoundDB
	}

	return nil
}

// UpsertChapterIntoDB updates the last read chapter in the database
// The chapter.Type field must be set to 2 (last read)
func (mm *MultiManga) UpsertChapterIntoDB(chapter *Chapter) error {
	contextError := "error upserting chapter '%s' to multimanga '%s' into DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, mm), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, mm), err)
	}

	err = upsertMultiMangaChapter(mm.ID, chapter, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, mm), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, chapter, mm), err)
	}
	mm.LastReadChapter = chapter

	return nil
}

// UpdateCurrentMangaInDB checks the manga is in the multimanga manga list
// and updates the current manga in the database.
func (mm *MultiManga) UpdateCurrentMangaInDB() error {
	contextError := "error updating multimanga '%s' current manga"
	m, err := GetLatestManga(mm.Mangas)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}
	contextError = "error updating multimanga '%s' current manga to manga '%s' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, m), err)
	}

	mm.CurrentManga = m
	err = updateMultiMangaCurrentManga(mm, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, m), err)
	}

	return nil
}

// updateMultiMangaCurrentManga updates the multimanga current manga in the database.
func updateMultiMangaCurrentManga(mm *MultiManga, tx *sql.Tx) error {
	err := validateMultiManga(mm)
	if err != nil {
		return err
	}

	err = validateManga(mm.CurrentManga)
	if err != nil {
		return err
	}
	mangaID := mm.CurrentManga.ID
	if mangaID < 1 {
		mangaID, err = getMangaIDByURL(mm.CurrentManga.URL)
		if err != nil {
			return err
		}
	}

	query := `
        UPDATE multimangas
        SET current_manga = $1
        WHERE id = $2;
    `
	result, err := tx.Exec(query, mangaID, mm.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errordefs.ErrMultiMangaNotFoundDB
	}

	return nil
}

// AddManga adds a manga to the multimanga.
// It also inserts the manga into the DB and updates the multimanga current manga.
// Manga should a fully valid manga.
func (mm *MultiManga) AddManga(m *Manga) error {
	contextError := "error adding manga '%s' to multimanga '%s' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	m.MultiMangaID = mm.ID
	m.LastReadChapter = nil
	m.CoverImgFixed = false
	mangaID, err := insertMangaIntoDB(m, tx)
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}
	m.ID = mangaID
	mm.Mangas = append(mm.Mangas, m)

	currentManga, err := GetLatestManga(mm.Mangas)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	mm.CurrentManga = currentManga
	err = updateMultiMangaCurrentManga(mm, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	return nil
}

// RemoveManga removes a manga from the multimanga mangas list
// and updates current manga.
func (mm *MultiManga) RemoveManga(m *Manga) error {
	contextError := "error removing manga '%s' from multimanga '%s' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	mangaIdx := -1
	for i, manga := range mm.Mangas {
		if manga == m {
			mangaIdx = i
			break
		}
	}
	if mangaIdx == -1 {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), errordefs.ErrMangaNotFoundInMultiManga)
	}
	if len(mm.Mangas) == 1 {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), errordefs.ErrAttemptedToRemoveLastMultiMangaManga)
	}

	mangas := append(mm.Mangas[:mangaIdx], mm.Mangas[mangaIdx+1:]...)

	currentManga, err := GetLatestManga(mangas)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	mm.CurrentManga = currentManga
	err = updateMultiMangaCurrentManga(mm, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, m), err)
	}

	err = deleteMangaDB(m, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	mm.Mangas = mangas

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	return nil
}

// GetMultiMangaFromDB gets a multimanga from the database by its ID
func GetMultiMangaFromDB(multimangaID ID) (*MultiManga, error) {
	contextError := "error getting multimanga with ID '%d' from DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, multimangaID), err)
	}
	defer db.Close()

	mm, err := getMultiMangaFromDB(multimangaID, db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, multimangaID), errordefs.ErrMultiMangaNotFoundDB)
		}
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, multimangaID), err)
	}

	return mm, nil
}

// GetMultiMangasDB gets all multimangas from the database.
// If getMangas is false, gets only the multimanga's current manga. Also add it to the multimanga.Mangas slice.
// If true, gets all mangas in the multimanga, and set one of them as the current manga (slow).
func GetMultiMangasDB(getMangas bool) ([]*MultiManga, error) {
	contextError := "error getting multimangas from DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}
	defer db.Close()

	var multimangas []*MultiManga
	if !getMangas {
		multimangas, err = getMultiMangasWithoutMangasDB(db)
	} else {
		multimangas, err = getMultiMangasWithMangasDB(db)
	}
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}

	return multimangas, nil
}

func getMultiMangasWithoutMangasDB(db *sql.DB) ([]*MultiManga, error) {
	query := `
        SELECT 
            mm.id AS multimanga_id,
            mm.status AS multimanga_status,
            mm.cover_img AS multimanga_cover_img,
            mm.cover_img_url AS multimanga_cover_img_url,
            mm.cover_img_resized AS multimanga_cover_img_resized,
            mm.cover_img_fixed AS multimanga_cover_img_fixed,

            -- current manga
            cm.id AS manga_id,
            cm.source,
            cm.url,
            cm.name,
            cm.internal_id,
            cm.preferred_group,
            cm.cover_img_url AS manga_cover_img_url,
            cm.cover_img AS manga_cover_img,
            cm.cover_img_resized AS manga_cover_img_resized,

            -- other mangas
            COALESCE(json_agg(DISTINCT om.name) FILTER (WHERE om.id IS NOT NULL)::TEXT, '[]') AS other_mangas,

            -- last released chapter
            last_released_chapter.url AS last_released_chapter_url,
            last_released_chapter.chapter AS last_released_chapter,
            last_released_chapter.name AS last_released_chapter_name,
            last_released_chapter.internal_id AS last_released_chapter_internal_id,
            last_released_chapter.updated_at AS last_released_chapter_updated_at,
            last_released_chapter.type AS last_released_chapter_type,

            -- last read chapter
            last_read_chapter.url AS last_read_chapter_url,
            last_read_chapter.chapter AS last_read_chapter,
            last_read_chapter.name AS last_read_chapter_name,
            last_read_chapter.internal_id AS last_read_chapter_internal_id,
            last_read_chapter.updated_at AS last_read_chapter_updated_at,
            last_read_chapter.type AS last_read_chapter_type
        FROM 
            multimangas AS mm
        LEFT JOIN 
            mangas AS cm ON cm.id = mm.current_manga
        LEFT JOIN 
            mangas AS om ON om.multimanga_id = mm.id
        LEFT JOIN 
            chapters AS last_released_chapter ON last_released_chapter.id = cm.last_released_chapter
        LEFT JOIN
            chapters AS last_read_chapter ON last_read_chapter.id = mm.last_read_chapter
        GROUP BY
            mm.id, cm.id, mm.status, mm.cover_img, mm.cover_img_url, mm.cover_img_resized, mm.cover_img_fixed,
            cm.source, cm.url, cm.name, cm.internal_id, cm.preferred_group, cm.cover_img_url, cm.cover_img, cm.cover_img_resized,
            last_released_chapter.url, last_released_chapter.chapter, last_released_chapter.name, last_released_chapter.internal_id,
            last_released_chapter.updated_at, last_released_chapter.type,
            last_read_chapter.url, last_read_chapter.chapter, last_read_chapter.name, last_read_chapter.internal_id,
            last_read_chapter.updated_at, last_read_chapter.type
    ;
    `
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var multiMangas []*MultiManga

	for rows.Next() {
		var multimanga MultiManga
		var currentManga Manga
		var lastReleasedChapter, multiLastReadChapter Chapter

		var (
			lastReleasedChapterURL, lastReleasedChapterChapter, lastReleasedChapterName, lastReleasedChapterInternalID sql.NullString
			lastReleasedChapterUpdatedAt                                                                               sql.NullTime
			lastReleasedChapterType                                                                                    sql.NullInt32

			multiLastReadChapterURL, multiLastReadChapterChapter, multiLastReadChapterName, multiLastReadChapterInternalID sql.NullString
			multiLastReadChapterUpdatedAt                                                                                  sql.NullTime
			multiLastReadChapterType                                                                                       sql.NullInt32
		)

		altNames := []byte{}
		err = rows.Scan(
			&multimanga.ID,
			&multimanga.Status,
			&multimanga.CoverImg,
			&multimanga.CoverImgURL,
			&multimanga.CoverImgResized,
			&multimanga.CoverImgFixed,
			&currentManga.ID,
			&currentManga.Source,
			&currentManga.URL,
			&currentManga.Name,
			&currentManga.InternalID,
			&currentManga.PreferredGroup,
			&currentManga.CoverImgURL,
			&currentManga.CoverImg,
			&currentManga.CoverImgResized,
			&altNames,
			&lastReleasedChapterURL,
			&lastReleasedChapterChapter,
			&lastReleasedChapterName,
			&lastReleasedChapterInternalID,
			&lastReleasedChapterUpdatedAt,
			&lastReleasedChapterType,
			&multiLastReadChapterURL,
			&multiLastReadChapterChapter,
			&multiLastReadChapterName,
			&multiLastReadChapterInternalID,
			&multiLastReadChapterUpdatedAt,
			&multiLastReadChapterType,
		)
		if err != nil {
			return nil, err
		}

		if len(altNames) > 0 {
			err = json.Unmarshal(altNames, &currentManga.SearchNames)
			if err != nil {
				return nil, err
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

		if multiLastReadChapterURL.Valid {
			multiLastReadChapter.URL = multiLastReadChapterURL.String
			multiLastReadChapter.Chapter = multiLastReadChapterChapter.String
			multiLastReadChapter.Name = multiLastReadChapterName.String
			multiLastReadChapter.InternalID = multiLastReadChapterInternalID.String
			multiLastReadChapter.UpdatedAt = multiLastReadChapterUpdatedAt.Time
			multiLastReadChapter.Type = Type(multiLastReadChapterType.Int32)
			multimanga.LastReadChapter = &multiLastReadChapter
		}

		currentManga.MultiMangaID = multimanga.ID
		currentManga.Status = multimanga.Status
		multimanga.CurrentManga = &currentManga
		multimanga.Mangas = append(multimanga.Mangas, &currentManga)

		err = validateMultiManga(&multimanga)
		if err != nil {
			return nil, err
		}

		multiMangas = append(multiMangas, &multimanga)
	}

	return multiMangas, nil
}

func getMultiMangasWithMangasDB(db *sql.DB) ([]*MultiManga, error) {
	query := `
        SELECT 
            multimangas.id AS multimanga_id,
            multimangas.status AS multimanga_status,
            multimangas.cover_img AS multimanga_cover_img,
            multimangas.cover_img_url AS multimanga_cover_img_url,
            multimangas.cover_img_resized AS multimanga_cover_img_resized,
            multimangas.cover_img_fixed AS multimanga_cover_img_fixed,
            multimangas.current_manga AS multimanga_current_manga,

            -- last read chapter
            last_read_chapter.url AS last_read_chapter_url,
            last_read_chapter.chapter AS last_read_chapter,
            last_read_chapter.name AS last_read_chapter_name,
            last_read_chapter.internal_id AS last_read_chapter_internal_id,
            last_read_chapter.updated_at AS last_read_chapter_updated_at,
            last_read_chapter.type AS last_read_chapter_type
        FROM 
            multimangas
        LEFT JOIN 
            mangas ON multimangas.current_manga = mangas.id
        LEFT JOIN
            chapters AS last_read_chapter ON last_read_chapter.id = multimangas.last_read_chapter
    ;
    `
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var multiMangas []*MultiManga

	for rows.Next() {
		var multimanga MultiManga
		var multiLastReadChapter Chapter
		var currentMangaID int

		var (
			multiLastReadChapterURL, multiLastReadChapterChapter, multiLastReadChapterName, multiLastReadChapterInternalID sql.NullString
			multiLastReadChapterUpdatedAt                                                                                  sql.NullTime
			multiLastReadChapterType                                                                                       sql.NullInt32
		)

		err = rows.Scan(
			&multimanga.ID,
			&multimanga.Status,
			&multimanga.CoverImg,
			&multimanga.CoverImgURL,
			&multimanga.CoverImgResized,
			&multimanga.CoverImgFixed,
			&currentMangaID,
			&multiLastReadChapterURL,
			&multiLastReadChapterChapter,
			&multiLastReadChapterName,
			&multiLastReadChapterInternalID,
			&multiLastReadChapterUpdatedAt,
			&multiLastReadChapterType,
		)
		if err != nil {
			return nil, err
		}

		if multiLastReadChapterURL.Valid {
			multiLastReadChapter.URL = multiLastReadChapterURL.String
			multiLastReadChapter.Chapter = multiLastReadChapterChapter.String
			multiLastReadChapter.Name = multiLastReadChapterName.String
			multiLastReadChapter.InternalID = multiLastReadChapterInternalID.String
			multiLastReadChapter.UpdatedAt = multiLastReadChapterUpdatedAt.Time
			multiLastReadChapter.Type = Type(multiLastReadChapterType.Int32)
			multimanga.LastReadChapter = &multiLastReadChapter
		}

		mangas, err := getMultiMangaMangasFromDB(multimanga.ID, db)
		if err != nil {
			return nil, err
		}
		multimanga.Mangas = mangas

		for _, manga := range mangas {
			if manga.ID == ID(currentMangaID) {
				multimanga.CurrentManga = manga
				break
			}
		}

		if multimanga.CurrentManga == nil {
			return nil, fmt.Errorf("current manga of multimanga with ID '%d' not found in DB", multimanga.ID)
		}

		err = validateMultiManga(&multimanga)
		if err != nil {
			return nil, err
		}

		multiMangas = append(multiMangas, &multimanga)
	}

	return multiMangas, nil
}

func getMultiMangaFromDB(multimangaID ID, db *sql.DB) (*MultiManga, error) {
	var currentMangaID sql.NullInt64
	var lastReadChapterID sql.NullInt64

	mm := &MultiManga{}

	query := `
        SELECT
            id, status, cover_img, cover_img_resized, cover_img_url, cover_img_fixed, current_manga, last_read_chapter
        FROM
            multimangas
        WHERE
            id = $1;
    `
	err := db.QueryRow(query, multimangaID).Scan(&mm.ID, &mm.Status, &mm.CoverImg, &mm.CoverImgResized, &mm.CoverImgURL, &mm.CoverImgFixed, &currentMangaID, &lastReadChapterID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errordefs.ErrMultiMangaNotFoundDB
		}
		return nil, err
	}

	mangas, err := getMultiMangaMangasFromDB(mm.ID, db)
	if err != nil {
		return nil, err
	}
	mm.Mangas = mangas
	for _, manga := range mm.Mangas {
		if manga.ID == ID(currentMangaID.Int64) {
			mm.CurrentManga = manga
			break
		}
	}

	if lastReadChapterID.Valid && lastReadChapterID.Int64 != 0 {
		chapter, err := getChapterDB(int(lastReadChapterID.Int64), db)
		if err != nil {
			return nil, err
		}
		mm.LastReadChapter = chapter
	}

	err = validateMultiManga(mm)
	if err != nil {
		return nil, err
	}

	return mm, nil
}

func getMultiMangaMangasFromDB(multiMangaID ID, db *sql.DB) ([]*Manga, error) {
	query := `
        SELECT 
            multimangas.status AS multimanga_status,

            mangas.id AS manga_id,
            mangas.source,
            mangas.url,
            mangas.name,
            mangas.internal_id,
            mangas.preferred_group,
            mangas.multimanga_id AS manga_multimanga_id,
            mangas.cover_img_url AS manga_cover_img_url,
            mangas.cover_img AS manga_cover_img,
            mangas.cover_img_resized AS manga_cover_img_resized,
            mangas.last_read_chapter AS manga_last_read_chapter_id,
            
            last_released_chapter.url AS last_released_chapter_url,
            last_released_chapter.chapter AS last_released_chapter,
            last_released_chapter.name AS last_released_chapter_name,
            last_released_chapter.internal_id AS last_released_chapter_internal_id,
            last_released_chapter.updated_at AS last_released_chapter_updated_at,
            last_released_chapter.type AS last_released_chapter_type
        FROM 
            multimangas
        INNER JOIN 
            mangas ON mangas.multimanga_id = multimangas.id
        LEFT JOIN 
            chapters AS last_released_chapter ON last_released_chapter.id = mangas.last_released_chapter
        WHERE 
            multimangas.id = $1
    `
	rows, err := db.Query(query, multiMangaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangas []*Manga

	for rows.Next() {
		var currentManga Manga
		var lastReleasedChapter Chapter

		var (
			lastReleasedChapterURL, lastReleasedChapterChapter, lastReleasedChapterName, lastReleasedChapterInternalID sql.NullString
			lastReleasedChapterUpdatedAt                                                                               sql.NullTime
			lastReleasedChapterType                                                                                    sql.NullInt32
		)

		err := rows.Scan(
			&currentManga.Status, &currentManga.ID, &currentManga.Source, &currentManga.URL, &currentManga.Name,
			&currentManga.InternalID, &currentManga.PreferredGroup, &currentManga.MultiMangaID, &currentManga.CoverImgURL,
			&currentManga.CoverImg, &currentManga.CoverImgResized, &currentManga.LastReadChapter,

			&lastReleasedChapterURL, &lastReleasedChapterChapter, &lastReleasedChapterName,
			&lastReleasedChapterInternalID, &lastReleasedChapterUpdatedAt, &lastReleasedChapterType,
		)
		if err != nil {
			return nil, err
		}

		currentManga.SearchNames = []string{currentManga.Name}

		if lastReleasedChapterURL.Valid {
			lastReleasedChapter.URL = lastReleasedChapterURL.String
			lastReleasedChapter.Chapter = lastReleasedChapterChapter.String
			lastReleasedChapter.Name = lastReleasedChapterName.String
			lastReleasedChapter.InternalID = lastReleasedChapterInternalID.String
			lastReleasedChapter.UpdatedAt = lastReleasedChapterUpdatedAt.Time
			lastReleasedChapter.Type = Type(lastReleasedChapterType.Int32)
			currentManga.LastReleasedChapter = &lastReleasedChapter
		}

		err = validateManga(&currentManga)
		if err != nil {
			return nil, err
		}

		mangas = append(mangas, &currentManga)
	}

	return mangas, nil
}

// TurnIntoMultiManga turns a manga into a multimanga.
// It creates the multimanga in DB, sets the manga as the current manga,
// adds the manga to the multimanga manga list, and sets the status and last read chapter.
func TurnIntoMultiManga(m *Manga) (*MultiManga, error) {
	contextError := "error turning manga '%s' into multimanga in DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	multiManga, err := turnMangaIntoMultiMangaInDB(m, tx)
	if err != nil {
		tx.Rollback()
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, m), err)
	}

	return multiManga, nil
}

func turnMangaIntoMultiMangaInDB(m *Manga, tx *sql.Tx) (*MultiManga, error) {
	err := validateManga(m)
	if err != nil {
		return nil, err
	}

	err = deleteMangaDB(m, tx)
	if err != nil {
		return nil, err
	}

	multiManga := &MultiManga{
		CurrentManga:    m,
		LastReadChapter: m.LastReadChapter,
		Mangas:          []*Manga{m},
		Status:          m.Status,
	}

	err = insertMultiMangaIntoDB(multiManga, tx)
	if err != nil {
		return nil, err
	}

	return multiManga, nil
}

func validateMultiManga(mm *MultiManga) error {
	contextError := "error validating multimanga"

	err := ValidateStatus(mm.Status)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	if mm.CurrentManga == nil {
		return util.AddErrorContext(contextError, fmt.Errorf("current manga is nil"))
	}
	if len(mm.Mangas) == 0 {
		return util.AddErrorContext(contextError, fmt.Errorf("mangas is empty"))
	}
	var found bool
	for _, manga := range mm.Mangas {
		err = validateManga(manga)
		if err != nil {
			return util.AddErrorContext(contextError, err)
		}
		if manga == mm.CurrentManga {
			found = true
		}
	}
	if !found {
		return util.AddErrorContext(contextError, fmt.Errorf("current manga not found in multimanga mangas"))
	}

	return nil
}

// GetLatestManga: tries to return the manga with the latest chapter.
func GetLatestManga(mangas []*Manga) (*Manga, error) {
	if len(mangas) == 0 {
		return nil, errordefs.ErrMultiMangaMangaListIsEmpty
	}
	if len(mangas) == 1 {
		return mangas[0], nil
	}
	currentManga := mangas[0]
	for _, manga := range mangas[1:] {
		currentChapter := currentManga.LastReleasedChapter
		newChapter := manga.LastReleasedChapter
		if currentChapter == nil {
			currentManga = manga
			continue
		}
		if newChapter == nil {
			continue
		}

		currentChapterInt, err := strconv.ParseFloat(currentChapter.Chapter, 64)
		if err != nil {
			if currentChapter.UpdatedAt.Before(newChapter.UpdatedAt) {
				currentManga = manga
			}
			continue
		}
		newChapterInt, err := strconv.ParseFloat(newChapter.Chapter, 64)
		if err != nil {
			if currentChapter.UpdatedAt.Before(newChapter.UpdatedAt) {
				currentManga = manga
			}
			continue
		}
		if currentChapterInt < newChapterInt {
			currentManga = manga
			continue
		} else if currentChapterInt > newChapterInt {
			continue
		} else if currentChapter.UpdatedAt.Before(newChapter.UpdatedAt) {
			currentManga = manga
		}
	}

	return currentManga, nil
}
