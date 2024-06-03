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
	// 0: "release" - the chapter was released, it's representing a chapter that was released by (or scraped from) a source
	// 1: "read" - the chapter was read, it's representing a chapter that was read by the user
	Type int
)

// Chapter is the struct for a chapter.
// Chapter don't has exported methods because a chapter should be used only by a manga.
type Chapter struct {
	// URL is the URL of the chapter
	URL string
	// Chapter usually is the chapter number, but in some cases it can be a one-shot or a special chapter
	Chapter string
	// Name is the name of the chapter
	Name string
	// UpdatedAt is the time when the chapter was released or updated (read).
	// Should truncate at the second.
	// The timezone should be the default/system timezone.
	UpdatedAt time.Time
	Type      Type
}

func (c Chapter) String() string {
	return fmt.Sprintf("Chapter{URL: %s, Chapter: %s, Name: %s, UpdatedAt: %s, Type: %d}", c.URL, c.Chapter, c.Name, c.UpdatedAt, c.Type)
}

func insertChapterDB(c *Chapter, mangaID ID, tx *sql.Tx) (int, error) {
	contextError := "error inserting chapter of manga ID '%d' in the database"

	err := validateChapter(c)
	if err != nil {
		return -1, util.AddErrorContext(fmt.Sprintf(contextError, mangaID), err)
	}
	var chapterID int
	err = tx.QueryRow(`
        INSERT INTO chapters
            (manga_id, url, chapter, name, updated_at, type)
        VALUES
            ($1, $2, $3, $4, $5, $6)
        RETURNING
            id;
    `, mangaID, c.URL, c.Chapter, c.Name, c.UpdatedAt, c.Type).Scan(&chapterID)
	if err != nil {
		return -1, util.AddErrorContext(fmt.Sprintf(contextError, mangaID), err)
	}

	return chapterID, nil
}

func getChapterDB(id int, db *sql.DB) (*Chapter, error) {
	contextError := "error getting chapter with ID '%d' from the database"

	var chapter Chapter
	err := db.QueryRow(`
        SELECT
            url, chapter, name, updated_at, type
        FROM
            chapters
        WHERE
            id = $1;
    `, id).Scan(&chapter.URL, &chapter.Chapter, &chapter.Name, &chapter.UpdatedAt, &chapter.Type)
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
func upsertMangaChapter(m *Manga, chapter *Chapter, tx *sql.Tx) error {
	contextError := "error upserting manga chapter in the database"

	err := validateManga(m)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	err = validateChapter(chapter)
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}

	mangaID := m.ID
	if mangaID == 0 {
		mangaID, err = getMangaIDByURL(m.URL)
		if err != nil {
			return util.AddErrorContext(contextError, err)
		}
		m.ID = mangaID
	}

	var chapterID int
	err = tx.QueryRow(`
        INSERT INTO chapters (manga_id, url, chapter, name, updated_at, type)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT ON CONSTRAINT chapters_manga_id_type_unique
        DO UPDATE
            SET url = EXCLUDED.url, chapter = EXCLUDED.chapter, name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
        RETURNING id;
    `, m.ID, chapter.URL, chapter.Chapter, chapter.Name, chapter.UpdatedAt, chapter.Type).Scan(&chapterID)
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
	result, err = tx.Exec(query, chapterID, m.ID)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return util.AddErrorContext(contextError, err)
	}
	if rowsAffected == 0 {
		return util.AddErrorContext(contextError, errordefs.ErrMangaNotFoundDB)
	}

	return nil
}

// there is no deleteChapterDB because the chapter should
// not be deleted directly, it should be deleted when a
// manga is deleted because of DB constraints

// valdiateChapter should be used every time the API interacts with
// the mangas and chapter table in the database
func validateChapter(c *Chapter) error {
	contextError := "error validating chapter"

	if c.URL == "" {
		return util.AddErrorContext(contextError, fmt.Errorf("chapter URL is empty"))
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
