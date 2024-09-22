package manga

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/util"
)

type (
	// Type is the type of the chapter, it can be:
	// 1: "release" - the chapter was released, it's representing a chapter that was released by (or scraped from) a source
	// 2: "read" - the chapter was read, it's representing a chapter that was read by the user
	Type int
)

// Chapter is the struct for a chapter.
// Chapter don't has exported methods because a chapter should be used only by a manga.
type Chapter struct {
	// UpdatedAt is the time when the chapter was released or updated (read).
	// Should truncate at the second.
	// The timezone should be the default/system timezone.
	UpdatedAt time.Time
	// URL is the URL of the chapter
	// If custom manga chapter doesn't have a URL provided by the user, it should be like custom_manga_<uuid>.
	URL string
	// Chapter usually is the chapter number, but in some cases it can be a one-shot or a special chapter
	Chapter string
	// Name is the name of the chapter
	Name string
	// InteralID is a unique identifier for the chapter in the source
	InternalID string
	Type       Type
}

func (c Chapter) String() string {
	return fmt.Sprintf("Chapter{URL: %s, Chapter: %s, Name: %s, InternalID: %s, UpdatedAt: %s, Type: %d}", c.URL, c.Chapter, c.Name, c.InternalID, c.UpdatedAt, c.Type)
}

func getChapterDB(id int, db *sql.DB) (*Chapter, error) {
	contextError := "error getting chapter with ID '%d' from the database"

	var chapter Chapter
	err := db.QueryRow(`
        SELECT
            url, chapter, name, internal_id, updated_at, type
        FROM
            chapters
        WHERE
            id = $1;
    `, id).Scan(&chapter.URL, &chapter.Chapter, &chapter.Name, &chapter.InternalID, &chapter.UpdatedAt, &chapter.Type)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.AddErrorContext(fmt.Sprintf(contextError, id), errordefs.ErrChapterNotFoundDB)
		}
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, id), err)
	}

	err = validateChapter(&chapter)
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(contextError, id), err)
	}

	return &chapter, nil
}

// upsertMangaChapter updates the last released or last read chapter of a manga
// if the manga doesn't exist in the database, it will be inserted
func upsertMangaChapter(mangaID ID, chapter *Chapter, tx *sql.Tx) error {
	contextError := "error upserting manga chapter in the database"

	err := validateChapter(chapter)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	var chapterID int
	err = tx.QueryRow(`
        INSERT INTO chapters (manga_id, url, chapter, name, internal_id, updated_at, type)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT ON CONSTRAINT chapters_manga_id_type_unique
        DO UPDATE
            SET url = EXCLUDED.url, chapter = EXCLUDED.chapter, name = EXCLUDED.name, internal_id = EXCLUDED.internal_id, updated_at = EXCLUDED.updated_at
        RETURNING id;
    `, mangaID, chapter.URL, chapter.Chapter, chapter.Name, chapter.InternalID, chapter.UpdatedAt, chapter.Type).Scan(&chapterID)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	var query string
	if chapter.Type == 1 {
		query = `
            UPDATE mangas
            SET last_released_chapter = $1
            WHERE id = $2;
        `
	} else {
		query = `
            UPDATE mangas
            SET last_read_chapter = $1
            WHERE id = $2;
        `
	}

	var result sql.Result
	result, err = tx.Exec(query, chapterID, mangaID)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	if rowsAffected == 0 {
		return util.AddErrorContext(contextError, errordefs.ErrMangaNotFoundDB)
	}

	return nil
}

// upsertMangaChapter updates the last released or last read chapter of a manga
// if the manga doesn't exist in the database, it will be inserted
func upsertMultiMangaChapter(multiMangaID ID, chapter *Chapter, tx *sql.Tx) error {
	contextError := "error upserting multimanga chapter in the database"

	err := validateChapter(chapter)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	if chapter.Type != 2 {
		return util.AddErrorContext(contextError, fmt.Errorf("chapter type should be 2 (last read), instead it's %d", chapter.Type))
	}

	var chapterID int
	err = tx.QueryRow(`
        INSERT INTO chapters (multimanga_id, url, chapter, name, internal_id, updated_at, type)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT ON CONSTRAINT chapters_multimanga_id_type_unique
        DO UPDATE
            SET url = EXCLUDED.url, chapter = EXCLUDED.chapter, name = EXCLUDED.name, internal_id = EXCLUDED.internal_id, updated_at = EXCLUDED.updated_at
        RETURNING id;
    `, multiMangaID, chapter.URL, chapter.Chapter, chapter.Name, chapter.InternalID, chapter.UpdatedAt, chapter.Type).Scan(&chapterID)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	query := `
        UPDATE multimangas
        SET last_read_chapter = $1
        WHERE id = $2;
    `
	var result sql.Result
	result, err = tx.Exec(query, chapterID, multiMangaID)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	if rowsAffected == 0 {
		return util.AddErrorContext(contextError, errordefs.ErrMultiMangaNotFoundDB)
	}

	return nil
}

func deleteMangaChapter(mangaID ID, chapter *Chapter, tx *sql.Tx) error {
	contextError := "error deleting chapter in the database"

	err := validateChapter(chapter)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	var query string
	if chapter.Type == 1 {
		query = `
        UPDATE mangas
        SET last_released_chapter = NULL
        WHERE id = $1;
    `
	} else {
		query = `
        UPDATE mangas
        SET last_read_chapter = NULL
        WHERE id = $1;
    `
	}

	result, err := tx.Exec(query, mangaID)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	if rowsAffected == 0 {
		return util.AddErrorContext(contextError, errordefs.ErrMangaNotFoundDB)
	}

	query = `
        DELETE FROM chapters
        WHERE manga_id = $1 AND url = $2 AND type = $3;
    `
	result, err = tx.Exec(query, mangaID, chapter.URL, chapter.Type)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	if rowsAffected == 0 {
		return util.AddErrorContext(contextError, errordefs.ErrChapterNotFoundDB)
	}

	return nil
}

// valdiateChapter should be used every time the API interacts with
// the mangas and chapter table in the database
func validateChapter(c *Chapter) error {
	contextError := "error validating chapter"

	if c == nil {
		return util.AddErrorContext(contextError, fmt.Errorf("chapter is nil"))
	}
	if c.Chapter == "" {
		return util.AddErrorContext(contextError, fmt.Errorf("chapter chapter is empty"))
	}
	if c.Name == "" {
		return util.AddErrorContext(contextError, fmt.Errorf("chapter name is empty"))
	}
	if c.Type != 1 && c.Type != 2 {
		return util.AddErrorContext(contextError, fmt.Errorf("chapter type should be 1 (last release) or 2 (last read), instead it's %d", c.Type))
	}

	return nil
}
