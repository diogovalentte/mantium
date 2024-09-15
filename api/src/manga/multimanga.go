package manga

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/util"
)

// A multimanga is an interface to the same manga from multiple sources.
// It is used to get the manga with the lastest chapter.
type MultiManga struct {
	CurrentManga    *Manga
	LastReadChapter *Chapter
	Mangas          []*Manga
	ID              ID
	Status          Status // All mangas in the multimanga should have the same status
}

func (mm MultiManga) String() string {
	returnStr := fmt.Sprintf("MultiManga{ID: %d, Status: %d, LastReadChapter: %s, CurrentManga: %s, Mangas: [",
		mm.ID, mm.Status, mm.LastReadChapter, mm.CurrentManga)

	for _, manga := range mm.Mangas {
		returnStr += manga.String() + ", "
	}
	returnStr = strings.TrimSuffix(returnStr, ", ")
	returnStr += "]}"

	return returnStr
}

// InsertIntoDB saves the multimanga into the database
func (mm *MultiManga) InsertIntoDB() error {
	contextError := "error inserting multimanga '%s' into DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}

	multiMangaID, err := insertMultiMangaIntoDB(mm, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm), err)
	}
	mm.ID = multiMangaID

	return nil
}

func insertMultiMangaIntoDB(mm *MultiManga, tx *sql.Tx) (ID, error) {
	err := validateMultiManga(mm)
	if err != nil {
		return -1, err
	}

	var currentMangaID ID
	var mangaIDs []ID
	for _, manga := range mm.Mangas {
		manga.Type = 2
		mangaID, err := insertMangaIntoDB(manga, tx)
		if err != nil {
			return -1, err
		}
		manga.ID = mangaID
		mangaIDs = append(mangaIDs, mangaID)
		if manga.URL == mm.CurrentManga.URL {
			currentMangaID = mangaID
		}
	}

	var multiMangaID ID
	err = tx.QueryRow(`
        INSERT INTO multimangas
            (status, current_manga)
        VALUES
            ($1, $2)
        RETURNING
            id;
    `, mm.Status, currentMangaID).Scan(&multiMangaID)
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "multimangas_pkey"` {
			return -1, errordefs.ErrMultiMangaAlreadyInDB
		}
		return -1, err
	}

	for _, mangaID := range mangaIDs {
		query := "INSERT INTO multimanga_mangas (multimanga_id, manga_id) VALUES ($1, $2)"
		_, err = tx.Exec(query, multiMangaID, mangaID)
		if err != nil {
			return -1, err
		}
	}

	if mm.LastReadChapter != nil {
		err := upsertMultiMangaChapter(multiMangaID, mm.LastReadChapter, tx)
		if err != nil {
			if err.Error() == `pq: duplicate key value violates unique constraint "chapters_pkey"` {
				return -1, fmt.Errorf("last read chapter of the multimanga you're trying to add already exists in DB")
			}
			return -1, err
		}
	}

	return multiMangaID, nil
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

// Will delete chapters and rows in the multimanga_mangas table
// because of the foreign key constraint ON DELETE CASCADE
func deleteMultiMangaDB(mm *MultiManga, tx *sql.Tx) error {
	err := validateMultiManga(mm)
	if err != nil {
		return err
	}
	if mm.ID < 1 {
		return errordefs.ErrMultiMangaHasNoID
	}

	rows, err := tx.Query(`
        SELECT manga_id
        FROM multimanga_mangas
        WHERE multimanga_id = $1;
    `, mm.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var IDs pq.Int64Array
	for rows.Next() {
		var id ID
		err = rows.Scan(&id)
		if err != nil {
			return err
		}
		IDs = append(IDs, int64(id))
	}

	// have to delete the multimanga before the mangas because of foreign key constraints
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

	result, err = tx.Exec(`
        DELETE FROM mangas
        WHERE id = ANY($1);
    `, IDs)
	if err != nil {
		return err
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != int64(len(IDs)) {
		return fmt.Errorf("not all mangas were deleted")
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
	if mm.ID < 1 {
		return errordefs.ErrMultiMangaHasNoID
	}

	err = validateStatus(status)
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

// UpsertChapterInDB updates the last read chapter in the database
// The chapter.Type field must be set to 2 (last read)
func (mm *MultiManga) UpsertChapterInDB(chapter *Chapter) error {
	contextError := "error upserting chapter '%s' to multimanga '%s' in DB"

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

// UpdateCurrentMangaInDB updates the multimanga current manga in the database.
// The manga has to be already in the DB.
func (mm *MultiManga) UpdateCurrentMangaInDB(m *Manga) error {
	contextError := "error updating multimanga '%s' current manga to manga '%s' in DB"

	db, err := db.OpenConn()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, m), err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, m), err)
	}

	err = updateMultiMangaCurrentManga(mm, m, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, m), err)
	}

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, mm, m), err)
	}
	mm.CurrentManga = m

	return nil
}

func updateMultiMangaCurrentManga(mm *MultiManga, m *Manga, tx *sql.Tx) error {
	err := validateMultiManga(mm)
	if err != nil {
		return err
	}

	err = validateManga(m)
	if err != nil {
		return err
	}
	if m.ID < 1 {
		return errordefs.ErrMangaHasNoIDOrURL
	}

	query := `
        UPDATE multimangas
        SET current_manga = $1
        WHERE id = $2;
    `
	_, err = tx.Exec(query, m.ID, mm.ID)
	if err != nil {
		return err
	}

	return nil
}

// AddManga adds a manga to the multimanga.
// It inserts the manga into the DB.
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

	mangaID, err := addMangaToMultiMangaList(mm, m, tx)
	if err != nil {
		tx.Rollback()
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}
	m.ID = mangaID
	mm.Mangas = append(mm.Mangas, m)

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	return nil
}

func addMangaToMultiMangaList(mm *MultiManga, m *Manga, tx *sql.Tx) (ID, error) {
	err := validateMultiManga(mm)
	if err != nil {
		return -1, err
	}

	m.Type = 2
	mangaID, err := insertMangaIntoDB(m, tx)
	if err != nil {
		return -1, err
	}

	query := `
        INSERT INTO multimanga_mangas (multimanga_id, manga_id) VALUES ($1, $2)
    `
	_, err = tx.Exec(query, mm.ID, mangaID)
	if err != nil {
		return -1, err
	}

	return mangaID, nil
}

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
		if manga.URL == m.URL {
			mangaIdx = i
			break
		}
	}
	if mangaIdx == -1 {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), errordefs.ErrMangaNotFoundInMultiManga)
	}

	// only deleting the manga also deletes the register from the multimanga_mangas table
	// because of cascade on delete
	err = deleteMangaDB(m, tx)
	if err != nil {
		tx.Rollback()
		if util.ErrorContains(err, `update or delete on table "mangas" violates foreign key constraint "multimangas_current_manga_fkey" on table "multimangas"`) {
			return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), errordefs.ErrAttemptedToDeleteCurrentManga)
		}
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	mm.Mangas = append(mm.Mangas[:mangaIdx], mm.Mangas[mangaIdx+1:]...)

	err = tx.Commit()
	if err != nil {
		return util.AddErrorContext(fmt.Sprintf(contextError, m, mm), err)
	}

	return nil
}

// GetMultMangaDB gets a multimanga from the database by its ID
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

// GetMultiMangasDB gets all mangas from the database
func GetMultiMangasDB() ([]*MultiManga, error) {
	contextError := "error getting multimangas from DB"

	db, err := db.OpenConn()
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}
	defer db.Close()

	query := `
        SELECT
            id
        FROM
            multimangas;
    `
	rows, err := db.Query(query)
	if err != nil {
		return nil, util.AddErrorContext(contextError, err)
	}
	defer rows.Close()

	var multiMangaIDs pq.Int64Array
	for rows.Next() {
		var multiMangaID ID
		err = rows.Scan(&multiMangaID)
		if err != nil {
			return nil, util.AddErrorContext(contextError, err)
		}
		multiMangaIDs = append(multiMangaIDs, int64(multiMangaID))
	}

	var multiMangas []*MultiManga
	for _, multiMangaID := range multiMangaIDs {
		multiManga, err := getMultiMangaFromDB(ID(multiMangaID), db)
		if err != nil {
			return nil, util.AddErrorContext(contextError, err)
		}
		multiMangas = append(multiMangas, multiManga)
	}

	return multiMangas, nil
}

func getMultiMangaFromDB(multimangaID ID, db *sql.DB) (*MultiManga, error) {
	var currentMangaID sql.NullInt64
	var lastReadChapterID sql.NullInt64
	if multimangaID < 1 {
		return nil, errordefs.ErrMultiMangaHasNoID
	}

	mm := &MultiManga{}
	mm.ID = multimangaID

	query := `
        SELECT
            id, status, current_manga, last_read_chapter
        FROM
            multimangas
        WHERE
            id = $1;
    `
	err := db.QueryRow(query, mm.ID).Scan(&mm.ID, &mm.Status, &currentMangaID, &lastReadChapterID)
	if err != nil {
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
            manga_id
        FROM
            multimanga_mangas
        WHERE
            multimanga_id = $1;
    `
	rows, err := db.Query(query, multiMangaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangaIDs pq.Int64Array
	for rows.Next() {
		var mangaID ID
		err = rows.Scan(&mangaID)
		if err != nil {
			return nil, err
		}
		mangaIDs = append(mangaIDs, int64(mangaID))
	}

	var mangas []*Manga
	for _, mangaID := range mangaIDs {
		manga, err := GetMangaDBByID(ID(mangaID))
		if err != nil {
			return nil, err
		}
		mangas = append(mangas, manga)
	}

	return mangas, nil
}

func validateMultiManga(mm *MultiManga) error {
	contextError := "error validating multimanga"

	err := validateStatus(mm.Status)
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
		if manga.URL == mm.CurrentManga.URL {
			found = true
		}
	}
	if !found {
		return util.AddErrorContext(contextError, fmt.Errorf("current manga not found in multimanga mangas"))
	}

	return nil
}
