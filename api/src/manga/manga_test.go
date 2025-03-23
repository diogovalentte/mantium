package manga

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/errordefs"
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
	CoverImg:       []byte{},
	PreferredGroup: "MangaStream",
	LastReleasedChapter: &Chapter{
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
	SearchNames: []string{"one piece", "one piece manga"},
}

var chaptersTest = map[string]*Chapter{
	"last_released_chapter": {
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

func TestMangaString(t *testing.T) {
	t.Run("TestString", func(t *testing.T) {
		ss := mangaTest.String()
		if ss == "" {
			t.Errorf("Error: expected string to not be empty")
		}
	})
}

func TestMangaDBLifeCycle(t *testing.T) {
	var err error
	manga := getMangaCopy(mangaTest)

	// Testing manga and chapter validations
	t.Run("Should insert a manga into DB", func(t *testing.T) {
		manga.Status = 0
		err = manga.InsertIntoDB()
		if err != nil {
			if util.ErrorContains(err, "status should be >= 1 && <= 5") {
				manga.Status = mangaTest.Status
				manga.LastReleasedChapter.Name = ""
				err = manga.InsertIntoDB()
				if util.ErrorContains(err, "chapter name is empty") {
					manga.LastReleasedChapter.Name = mangaTest.LastReleasedChapter.Name
					manga.LastReleasedChapter.Type = 0
					err = manga.InsertIntoDB()
					if util.ErrorContains(err, "chapter type should be 1 (last release) or 2 (last read)") {
						manga.LastReleasedChapter.Type = mangaTest.LastReleasedChapter.Type
						err = manga.InsertIntoDB()
						if err != nil {
							t.Fatal(err)
						}
					} else {
						t.Fatal(err)
					}
				} else {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal(fmt.Errorf("no errors while adding the invalid manga to DB"))
		}
	})
	t.Run("Should get a manga's ID and then get the manga from DB by ID", func(t *testing.T) {
		_, err := getMangaIDByURL(manga.URL + "salt")
		if err != nil {
			if util.ErrorContains(err, errordefs.ErrMangaNotFoundDB.Error()) {
				mangaID, err := getMangaIDByURL(manga.URL)
				if err != nil {
					t.Fatal(err)
				}
				if mangaID != manga.ID {
					t.Fatal(fmt.Errorf("manga ID from URL is different from the one in DB"))
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while getting the invalid manga from DB")
		}

		_, err = GetMangaDBByID(manga.ID - 10000)
		if err != nil {
			if util.ErrorContains(err, errordefs.ErrMangaHasNoIDOrURL.Error()) {
				_, err := getMangaIDByURL(manga.URL)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal(fmt.Errorf("no errors while getting the invalid manga from DB"))
		}
		_, err = GetMangaDBByID(manga.ID + 10000)
		if err != nil {
			if util.ErrorContains(err, errordefs.ErrMangaNotFoundDB.Error()) {
				_, err = getMangaIDByURL(manga.URL)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while getting the invalid manga from DB")
		}
	})
	t.Run("Should get a manga from DB By URL", func(t *testing.T) {
		_, err := GetMangaDBByURL(manga.URL + "salt")
		if err != nil {
			if util.ErrorContains(err, errordefs.ErrMangaNotFoundDB.Error()) {
				_, err = getMangaIDByURL(manga.URL)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while getting the invalid manga from DB")
		}
	})
	t.Run("Should update a manga's status in DB", func(t *testing.T) {
		err := manga.UpdateStatusInDB(6)
		if err != nil {
			if util.ErrorContains(err, "status should be >= 1 && <= 5") {
				err = manga.UpdateStatusInDB(5)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while updating the manga with an invalid status in DB")
		}
	})
	t.Run("Should update a manga's name in DB", func(t *testing.T) {
		err := manga.UpdateNameInDB("new manga name")
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should update a manga's URL in DB", func(t *testing.T) {
		err := manga.UpdateURLInDB("https://new-manga-url.com")
		if err != nil {
			t.Fatal(err)
		}
		err = manga.UpdateURLInDB(mangaTest.URL)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should update a manga's cover image in DB", func(t *testing.T) {
		err := manga.UpdateCoverImgInDB([]byte{}, false, "https://cnd.random.best-manga.jpg")
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Get library stats", func(t *testing.T) {
		stats, err := GetLibraryStats()
		if err != nil {
			t.Fatal(err)
		}

		if len(stats) == 0 {
			t.Fatal("no stats returned")
		}
	})
	t.Run("Should delete the last released chapter of manga from DB", func(t *testing.T) {
		err := manga.DeleteLastReleasedChapterFromDB()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should update a manga's last released chapter in DB", func(t *testing.T) {
		chapter := *chaptersTest["last_released_chapter"]
		chapter.Type = 0

		err := manga.UpsertChapterIntoDB(&chapter)
		if err != nil {
			if util.ErrorContains(err, "chapter type should be 1 (last release) or 2 (last read)") {
				chapter.Type = chaptersTest["last_released_chapter"].Type
				err = manga.UpsertChapterIntoDB(&chapter)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while updating the manga with an invalid chapter in DB")
		}
	})
	t.Run("Should update a manga's last read chapter in DB", func(t *testing.T) {
		chapter := *chaptersTest["last_read_chapter"]
		chapter.Type = 0

		err := manga.UpsertChapterIntoDB(&chapter)
		if err != nil {
			if util.ErrorContains(err, "chapter type should be 1 (last release) or 2 (last read)") {
				chapter.Type = chaptersTest["last_read_chapter"].Type
				err = manga.UpsertChapterIntoDB(&chapter)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while updating the manga with an invalid chapter in DB")
		}
	})
	t.Run("Should delete the last read chapter of manga from DB", func(t *testing.T) {
		err := manga.DeleteLastReadChapterFromDB()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should delete a manga into DB", func(t *testing.T) {
		err = manga.DeleteFromDB()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestMangaWithoutChaptersDBLifeCycle(t *testing.T) {
	var err error
	manga := getMangaCopy(mangaTest)
	manga.LastReleasedChapter = nil
	manga.LastReadChapter = nil

	// Testing manga and chapter validations
	t.Run("Should insert a manga into DB", func(t *testing.T) {
		manga.Status = 0
		err = manga.InsertIntoDB()
		if err != nil {
			if util.ErrorContains(err, "status should be >= 1 && <= 5") {
				manga.Status = mangaTest.Status
				err = manga.InsertIntoDB()
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while adding the invalid manga to DB")
		}
	})
	t.Run("Should get a manga's ID and then get the manga from DB by ID", func(t *testing.T) {
		_, err := getMangaIDByURL(manga.URL + "salt")
		if err != nil {
			if util.ErrorContains(err, errordefs.ErrMangaNotFoundDB.Error()) {
				mangaID, err := getMangaIDByURL(manga.URL)
				if err != nil {
					t.Fatal(err)
				}
				if mangaID != manga.ID {
					t.Fatal("manga ID from URL is different from the one in DB")
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while getting the invalid manga from DB")
		}

		_, err = GetMangaDBByID(manga.ID - 10000)
		if err != nil {
			if util.ErrorContains(err, errordefs.ErrMangaHasNoIDOrURL.Error()) {
				_, err = getMangaIDByURL(manga.URL)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while getting the invalid manga from DB")
		}
		_, err = GetMangaDBByID(manga.ID + 10000)
		if err != nil {
			if util.ErrorContains(err, errordefs.ErrMangaNotFoundDB.Error()) {
				_, err = getMangaIDByURL(manga.URL)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while getting the invalid manga from DB")
		}
	})
	t.Run("Should get a manga from DB By URL", func(t *testing.T) {
		_, err := GetMangaDBByURL(manga.URL + "salt")
		if err != nil {
			if util.ErrorContains(err, errordefs.ErrMangaNotFoundDB.Error()) {
				_, err = getMangaIDByURL(manga.URL)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while getting the invalid manga from DB")
		}
	})
	t.Run("Should update a manga's status in DB", func(t *testing.T) {
		err := manga.UpdateStatusInDB(6)
		if err != nil {
			if util.ErrorContains(err, "status should be >= 1 && <= 5") {
				err = manga.UpdateStatusInDB(5)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while updating the manga with an invalid status in DB")
		}
	})
	t.Run("Should update a manga's last released chapter in DB", func(t *testing.T) {
		chapter := *chaptersTest["last_released_chapter"]
		chapter.Type = 0

		err := manga.UpsertChapterIntoDB(&chapter)
		if err != nil {
			if util.ErrorContains(err, "chapter type should be 1 (last release) or 2 (last read)") {
				chapter.Type = chaptersTest["last_released_chapter"].Type
				err = manga.UpsertChapterIntoDB(&chapter)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while updating the manga with an invalid chapter in DB")
		}
	})
	t.Run("Should update a manga's last read chapter in DB", func(t *testing.T) {
		chapter := *chaptersTest["last_read_chapter"]
		chapter.Type = 0

		err := manga.UpsertChapterIntoDB(&chapter)
		if err != nil {
			if util.ErrorContains(err, "chapter type should be 1 (last release) or 2 (last read)") {
				chapter.Type = chaptersTest["last_read_chapter"].Type
				err = manga.UpsertChapterIntoDB(&chapter)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while updating the manga with an invalid chapter in DB")
		}
	})
	t.Run("Should delete a manga into DB", func(t *testing.T) {
		err = manga.DeleteFromDB()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func getMangaCopy(source *Manga) *Manga {
	manga := *source
	if source.LastReleasedChapter != nil {
		lastReleasedChapter := *source.LastReleasedChapter
		manga.LastReleasedChapter = &lastReleasedChapter
	}
	if source.LastReadChapter != nil {
		lastReadChapter := *source.LastReadChapter
		manga.LastReadChapter = &lastReadChapter
	}

	return &manga
}
