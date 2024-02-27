package manga

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func setup() error {
	err := godotenv.Load("../../../.env.test")
	if err != nil {
		return err
	}

	return nil
}

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}

var mangaTest = &Manga{
	Source:         "testing",
	URL:            "https://testingsite/manga/best-manga",
	Name:           "One Piece",
	Status:         1,
	CoverImgURL:    "https://cnd.random.best-manga.jpg", // mangahub.io
	PreferredGroup: "MangaStream",
	LastUploadChapter: &Chapter{
		URL:       "https://testingsite/manga/best-manga/chapter-15",
		Name:      "Chapter 15",
		Chapter:   "1",
		UpdatedAt: time.Now(),
		Type:      1,
	},
	LastReadChapter: &Chapter{
		URL:       "https://testingsite/manga/best-manga/chapter-11",
		Name:      "Chapter 11",
		Chapter:   "1",
		UpdatedAt: time.Now(),
		Type:      2,
	},
}

var chaptersTest = map[string]*Chapter{
	"last_upload_chapter": {
		URL:     "https://testingsite/manga/best-manga/chapter-158",
		Name:    "Chapter 158",
		Chapter: "158",
		Type:    1,
	},
	"last_read_chapter": {
		URL:     "https://testingsite/manga/best-manga/chapter-1000",
		Name:    "Chapter 1000",
		Chapter: "1000",
		Type:    2,
	},
}

func TestMangaDBLifeCycle(t *testing.T) {
	var err error
	var mangaID ID
	t.Run("should insert a manga into DB", func(t *testing.T) {
		mangaID, err = mangaTest.InsertDB()
		if err != nil {
			t.Error(err)
			return
		}
		mangaTest.ID = mangaID
	})
	t.Run("should get a manga's ID and the get the manga from DB by ID", func(t *testing.T) {
		mangaID, err := getMangaIDByURL(mangaTest.URL)
		if err != nil {
			t.Error(err)
			return
		}
		_, err = GetMangaDBByID(mangaID)
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("should get a manga from DB By URL", func(t *testing.T) {
		_, err := GetMangaDBByURL(mangaTest.URL)
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("should get all mangas from DB", func(t *testing.T) {
		_, err := GetMangasDB()
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("should update a manga's status in DB", func(t *testing.T) {
		err := mangaTest.UpdateStatus(5)
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("should update a manga's last upload chapter in DB", func(t *testing.T) {
		err := mangaTest.UpdateChapter(chaptersTest["last_upload_chapter"])
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("should update a manga's last read chapter in DB", func(t *testing.T) {
		err := mangaTest.UpdateChapter(chaptersTest["last_read_chapter"])
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("should delete a manga into DB", func(t *testing.T) {
		err = mangaTest.DeleteDB()
		if err != nil {
			t.Error(err)
			return
		}
	})
}
