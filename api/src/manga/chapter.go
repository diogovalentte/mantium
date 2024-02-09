package manga

import (
	"database/sql"
	"fmt"
	"time"
)

type (
	// Type is the type of the chapter, it can be:
	// 0: "upload" - the chapter was uploaded, it's representing a chapter that was uploaded to (scraped from) a source
	// 1: "read" - the chapter was read, it's representing a chapter that was read by the user
	Type int
)

// Chapter is the struct for a chapter
// Chapter don't has exported methods because
// a chapter should be used only by a manga
type Chapter struct {
	// URL is the URL of the chapter
	URL string
	// Chapter usually is the chapter number, but in some cases it can be a one-shot or a special chapter
	Chapter string
	// Name is the name of the chapter
	Name string
	// UpdatedAt is the time when the chapter was uploaded or updated (read)
	UpdatedAt time.Time
	Type      Type
}

func insertChapterDB(c *Chapter, mangaID ID, tx *sql.Tx) (int, error) {
	err := validateChapter(c)
	if err != nil {
		return -1, err
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
		return -1, err
	}

	return chapterID, nil
}

func getChapterDB(id int, db *sql.DB) (*Chapter, error) {
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
			return nil, fmt.Errorf("chapter not found, is the ID correct?")
		}
		return nil, err
	}

	err = validateChapter(&chapter)
	if err != nil {
		return nil, err
	}

	return &chapter, nil
}

// updateMangaChapter updates the last upload or last read chapter of a manga
// if the manga doesn't exist in the database, it will be inserted
func updateMangaChapter(m *Manga, chapter *Chapter, tx *sql.Tx) error {
	err := validateManga(m)
	if err != nil {
		return err
	}

	mangaID := m.ID
	if mangaID == 0 {
		mangaID, err = getMangaIDByURL(m.URL)
		if err != nil {
			return err
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
		return err
	}

	var query string
	if chapter.Type == 1 {
		query = `
            UPDATE mangas
            SET last_upload_chapter = $1
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
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("manga not found in DB")
	}

	return nil
}

// there is no deleteChapterDB because the chapter should
// not be deleted directly, it should be deleted when a
// manga is deleted because of DB constraints

// valdiateChapter should be used every time the API interacts with
// the mangas and chapter table in the database
func validateChapter(c *Chapter) error {
	if c.URL == "" {
		return fmt.Errorf("chapter URL is empty")
	}
	if c.Chapter == "" {
		return fmt.Errorf("chapter chapter is empty")
	}
	if c.Name == "" {
		return fmt.Errorf("chapter name is empty")
	}
	if c.Type != 1 && c.Type != 2 {
		return fmt.Errorf("chapter type should be 1 (last upload) or 2 (last read)")
	}

	return nil
}
