package manga

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/util"
)

func setup() error {
	err := config.SetConfigs("../../../.env.test")
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
		Name:      "Chapter 1",
		Chapter:   "15",
		UpdatedAt: time.Now(),
		Type:      1,
	},
	LastReadChapter: &Chapter{
		URL:       "https://testingsite/manga/best-manga/chapter-11",
		Name:      "Chapter 11",
		Chapter:   "11",
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

func TestInvalidMangaDBLifeCycle(t *testing.T) {
	var err error
	var mangaID ID

	// Testing manga and chapter validations
	t.Run("should insert a manga into DB", func(t *testing.T) {
		manga := getMangaCopy(mangaTest)

		manga.Status = 0

		mangaID, err = manga.InsertIntoDB()
		if err != nil {
			if util.ErrorContains(err, "Manga status should be >= 1 && <= 5") {
				manga.Status = mangaTest.Status
				manga.LastUploadChapter.Name = ""
				mangaID, err = manga.InsertIntoDB()
				if util.ErrorContains(err, "Chapter name is empty") {
					manga.LastUploadChapter.Name = mangaTest.LastUploadChapter.Name
					manga.LastUploadChapter.Type = 0
					mangaID, err = manga.InsertIntoDB()
					if util.ErrorContains(err, "Chapter type should be 1 (last upload) or 2 (last read)") {
						manga.LastUploadChapter.Type = mangaTest.LastUploadChapter.Type
						manga.LastReadChapter.URL = ""
						mangaID, err = manga.InsertIntoDB()
						if util.ErrorContains(err, "Chapter URL is empty") {
							manga.LastReadChapter.URL = mangaTest.LastReadChapter.URL
							mangaID, err = manga.InsertIntoDB()
							if err != nil {
								t.Error(err)
								return
							}
						} else {
							t.Error(err)
							return
						}
					} else {
						t.Error(err)
						return
					}
				} else {
					t.Error(err)
					return
				}
			} else {
				t.Error(err)
				return
			}
		} else {
			t.Error(fmt.Errorf("No errors while adding the invalid manga to DB"))
			return
		}
		mangaTest.ID = mangaID
	})
	t.Run("should get a manga's ID and then get the manga from DB by ID", func(t *testing.T) {
		mangaID, err := getMangaIDByURL(mangaTest.URL + "salt")
		if err != nil {
			if util.ErrorContains(err, "Manga not found in DB") {
				mangaID, err = getMangaIDByURL(mangaTest.URL)
				if err != nil {
					t.Error(err)
					return
				}
			} else {
				t.Error(err)
				return
			}
		} else {
			t.Error(fmt.Errorf("No errors while getting the invalid manga from DB"))
			return
		}

		_, err = GetMangaDBByID(mangaID - 10000)
		if err != nil {
			if util.ErrorContains(err, "Manga doesn't have an ID or URL") {
				mangaID, err = getMangaIDByURL(mangaTest.URL)
				if err != nil {
					t.Error(err)
					return
				}
			} else {
				t.Error(err)
				return
			}
		} else {
			t.Error(fmt.Errorf("No errors while getting the invalid manga from DB"))
			return
		}
		_, err = GetMangaDBByID(mangaID + 10000)
		if err != nil {
			if util.ErrorContains(err, "Manga not found in DB") {
				mangaID, err = getMangaIDByURL(mangaTest.URL)
				if err != nil {
					t.Error(err)
					return
				}
			} else {
				t.Error(err)
				return
			}
		} else {
			t.Error(fmt.Errorf("No errors while getting the invalid manga from DB"))
			return
		}
	})
	t.Run("should get a manga from DB By URL", func(t *testing.T) {
		_, err := GetMangaDBByURL(mangaTest.URL + "salt")
		if err != nil {
			if util.ErrorContains(err, "Manga not found in DB") {
				mangaID, err = getMangaIDByURL(mangaTest.URL)
				if err != nil {
					t.Error(err)
					return
				}
			} else {
				t.Error(err)
				return
			}
		} else {
			t.Error(fmt.Errorf("No errors while getting the invalid manga from DB"))
			return
		}
	})
	t.Run("should get all mangas from DB", func(t *testing.T) {
		mangas, err := GetMangasDB()
		if err != nil {
			t.Error(err)
			return
		}

		if len(mangas) != 1 {
			t.Error("DB should have only one manga, instead it has:", len(mangas))
			return
		}
	})
	t.Run("should update a manga's status in DB", func(t *testing.T) {
		manga := getMangaCopy(mangaTest)

		err := manga.UpdateStatusInDB(6)
		if err != nil {
			if util.ErrorContains(err, "Manga status should be >= 1 && <= 5") {
				err = manga.UpdateStatusInDB(5)
				if err != nil {
					t.Error(err)
					return
				}
			} else {
				t.Error(err)
				return
			}
		} else {
			t.Error(fmt.Errorf("No errors while updating the manga with an invalid status in DB"))
			return
		}
	})
	t.Run("should update a manga's last upload chapter in DB", func(t *testing.T) {
		manga := getMangaCopy(mangaTest)
		chapter := *chaptersTest["last_upload_chapter"]

		chapter.Type = 0

		err := manga.UpsertChapterInDB(&chapter)
		if err != nil {
			if util.ErrorContains(err, "Chapter type should be 1 (last upload) or 2 (last read)") {
				chapter.Type = chaptersTest["last_upload_chapter"].Type
				err = manga.UpsertChapterInDB(&chapter)
				if err != nil {
					t.Error(err)
					return
				}
			} else {

				t.Error(err)
				return
			}
		} else {
			t.Error(fmt.Errorf("No errors while updating the manga with an invalid chapter in DB"))
			return
		}
	})
	t.Run("should update a manga's last read chapter in DB", func(t *testing.T) {
		manga := getMangaCopy(mangaTest)
		chapter := *chaptersTest["last_read_chapter"]

		chapter.Type = 0

		err := manga.UpsertChapterInDB(&chapter)
		if err != nil {
			if util.ErrorContains(err, "Chapter type should be 1 (last upload) or 2 (last read)") {
				chapter.Type = chaptersTest["last_read_chapter"].Type
				err = manga.UpsertChapterInDB(&chapter)
				if err != nil {
					t.Error(err)
					return
				}
			} else {

				t.Error(err)
				return
			}
		} else {
			t.Error(fmt.Errorf("No errors while updating the manga with an invalid chapter in DB"))
			return
		}
	})
	t.Run("should delete a manga into DB", func(t *testing.T) {
		err = mangaTest.DeleteFromDB()
		if err != nil {
			t.Error(err)
			return
		}
	})
}

func getMangaCopy(source *Manga) Manga {
	manga := *source
	lastUploadChapter := *source.LastUploadChapter
	lastReadChapter := *source.LastReadChapter
	manga.LastUploadChapter = &lastUploadChapter
	manga.LastReadChapter = &lastReadChapter

	return manga
}
