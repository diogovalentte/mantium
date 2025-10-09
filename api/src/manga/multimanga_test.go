package manga

import (
	"fmt"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/util"
)

var multimangaMangasTest = []*Manga{
	{
		Source:         "mangahub",
		URL:            "https://mangahub.io/manga/yotsubato",
		Name:           "Yotsuba&!",
		Status:         1,
		CoverImgURL:    "https://cnd.random.best-manga.jpg", // mangahub.io
		CoverImg:       []byte{},
		PreferredGroup: "testGroup",
		LastReadChapter: &Chapter{
			URL:       "https://mangahub.io/manga/yotsubato/chapter-14",
			Name:      "Chapter 14",
			Chapter:   "14",
			UpdatedAt: time.Now(),
			Type:      2,
		},
	},
	{
		Source:         "mangadex",
		URL:            "https://mangadex.org/title/yotsubato",
		Name:           "Yotsubato!",
		Status:         1,
		CoverImgURL:    "https://cnd.random.best-manga.jpg", // mangahub.io
		CoverImg:       []byte{},
		PreferredGroup: "",
		LastReleasedChapter: &Chapter{
			URL:       "https://magnadex.org/chapter/f54af4afd4",
			Name:      "Chapter 14",
			Chapter:   "Chapter 14",
			UpdatedAt: time.Date(2015, 10, 4, 13, 1, 52, 0, time.UTC),
			Type:      1,
		},
	},
	{
		Source:         "mangaupdates",
		URL:            "https://mangaupdates.com/manga/fas45a4",
		Name:           "Yotsubato",
		Status:         1,
		CoverImgURL:    "https://cnd.random.best-manga.jpg", // mangahub.io
		CoverImg:       []byte{},
		PreferredGroup: "",
		LastReleasedChapter: &Chapter{
			URL:       "https://mangaupdates.com/release/fff4a4a7",
			Name:      "Chapter 15",
			Chapter:   "15",
			UpdatedAt: time.Date(2015, 10, 2, 14, 11, 45, 0, time.UTC),
			Type:      1,
		},
	},
}

var multiMangaTest = &MultiManga{
	Status:       1,
	CurrentManga: multimangaMangasTest[0],
	Mangas:       multimangaMangasTest,
	LastReadChapter: &Chapter{
		URL:       "https://testingsite/manga/best-manga/chapter-15",
		Name:      "Chapter 1",
		Chapter:   "15",
		UpdatedAt: time.Now(),
		Type:      2,
	},
}

func TestString(t *testing.T) {
	t.Run("TestString", func(t *testing.T) {
		ss := multiMangaTest.String()
		if ss == "" {
			t.Errorf("Error: expected string to not be empty")
		}
	})
}

func TestMultiMangaDBLifeCycle(t *testing.T) {
	var err error
	multiManga := getMultiMangaCopy(multiMangaTest)

	t.Run("Should insert a multimanga into DB", func(t *testing.T) {
		multiManga.Status = 0
		err = multiManga.InsertIntoDB()
		if err != nil {
			if util.ErrorContains(err, "status should be >= 1 && <= 5") {
				multiManga.Status = multiMangaTest.Status
				multiManga.LastReadChapter.Name = ""
				err = multiManga.InsertIntoDB()
				if util.ErrorContains(err, "chapter name is empty") {
					multiManga.LastReadChapter.Name = multiMangaTest.LastReadChapter.Name
					multiManga.LastReadChapter.Type = 0
					err = multiManga.InsertIntoDB()
					if util.ErrorContains(err, "chapter type should be 1 (last release) or 2 (last read)") {
						multiManga.LastReadChapter.Type = multiMangaTest.LastReadChapter.Type
						err = multiManga.InsertIntoDB()
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
			t.Fatal(fmt.Errorf("no errors while adding the invalid multimanga to DB"))
		}
	})
	t.Run("Should update a manga's status in DB", func(t *testing.T) {
		err := multiManga.UpdateStatusInDB(6)
		if err != nil {
			if util.ErrorContains(err, "status should be >= 1 && <= 5") {
				err = multiManga.UpdateStatusInDB(5)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while updating the multimanga with an invalid status in DB")
		}
	})
	t.Run("Should update a manga's cover image in DB using image URL", func(t *testing.T) {
		err := multiManga.UpdateCoverImgInDB([]byte{}, false, "https://cnd.random.best-manga.jpg")
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should update a manga's last read chapter in DB", func(t *testing.T) {
		chapter := *chaptersTest["last_read_chapter"]
		chapter.Type = 0

		err := multiManga.UpsertChapterIntoDB(&chapter)
		if err != nil {
			if util.ErrorContains(err, "chapter type should be 1 (last release) or 2 (last read)") {
				chapter.Type = chaptersTest["last_read_chapter"].Type
				err = multiManga.UpsertChapterIntoDB(&chapter)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while updating the multimanga with an invalid chapter in DB")
		}
	})
	t.Run("Should not get a multimanga from DB", func(t *testing.T) {
		_, err = GetMultiMangaFromDB(0)
		if err != nil {
			if !util.ErrorContains(err, errordefs.ErrMultiMangaNotFoundDB.Error()) {
				t.Fatal(err)
			}
		} else {
			t.Fatal(fmt.Errorf("no errors while getting the invalid multimanga from DB"))
		}
		_, err = GetMultiMangaFromDB(multiManga.ID + 10000)
		if err != nil {
			if !util.ErrorContains(err, errordefs.ErrMultiMangaNotFoundDB.Error()) {
				t.Fatal(err)
			}
		} else {
			t.Fatal("no errors while getting the invalid multimanga from DB")
		}
	})
	t.Run("Should update the current manga of a multimanga 1", func(t *testing.T) {
		err := multiManga.UpdateCurrentMangaInDB()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should remove a manga from a multimanga list", func(t *testing.T) {
		err := multiManga.RemoveManga(multiManga.Mangas[1])
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should update the current manga of a multimanga 2", func(t *testing.T) {
		err := multiManga.UpdateCurrentMangaInDB()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should add a manga to a multimanga list", func(t *testing.T) {
		err := multiManga.AddManga(multiMangaTest.Mangas[1])
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should update the current manga of a multimanga 3", func(t *testing.T) {
		err := multiManga.UpdateCurrentMangaInDB()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should get a multimanga from DB", func(t *testing.T) {
		_, err := GetMultiMangaFromDB(multiManga.ID)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should get all multimangas from DB", func(t *testing.T) {
		multimangas, err := GetMultiMangasDB(true)
		if err != nil {
			t.Fatal(err)
		}

		if len(multimangas) < 1 {
			t.Fatal("DB should have at the least one multimanga, instead it has:", len(multimangas))
		}
	})
	t.Run("Should delete a multimanga from DB", func(t *testing.T) {
		err = multiManga.DeleteFromDB()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestMangaIntoMultiMangaDBLifeCycle(t *testing.T) {
	manga := getMangaCopy(multiMangaTest.Mangas[0])
	var multiManga *MultiManga

	t.Run("Should turn a manga into a multimanga into DB", func(t *testing.T) {
		err := manga.InsertIntoDB()
		if err != nil {
			t.Fatal(err)
		}
		multiManga, err = TurnIntoMultiManga(manga)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Should delete a multimanga from DB", func(t *testing.T) {
		err := multiManga.DeleteFromDB()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func getMultiMangaCopy(source *MultiManga) *MultiManga {
	multiManga := *source
	if source.LastReadChapter != nil {
		lastReadChapter := *source.LastReadChapter
		multiManga.LastReadChapter = &lastReadChapter
	}
	if len(source.Mangas) < 1 {
		panic("source.Mangas is empty")
	}

	mangas := make([]*Manga, len(source.Mangas))
	for i, manga := range source.Mangas {
		mangas[i] = getMangaCopy(manga)
	}
	multiManga.Mangas = mangas
	multiManga.CurrentManga = mangas[0]

	return &multiManga
}
