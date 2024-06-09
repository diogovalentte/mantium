package mangaplus

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL string, ignoreGetLastChapterError bool) (*manga.Manga, error) {
	s.checkClient()
	errorContext := "error while getting manga metadata"

	mangaID, err := getMangaID(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	_, response, err := s.client.Request(fmt.Sprintf("%s/title_detailV3?title_id=%d", baseAPIURL, mangaID))
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, errordefs.ErrMangaNotFound
		}
		return nil, util.AddErrorContext(errorContext, err)
	}

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = sourceName
	mangaReturn.URL = mangaURL

	titleView := response.GetSuccess().GetTitleDetailView()
	title := titleView.GetTitle()

	mangaReturn.Name = title.GetTitleName()
	caser := cases.Title(language.AmericanEnglish)
	mangaReturn.Name = strings.TrimSpace(caser.String(strings.ToLower(mangaReturn.Name)))

	mangaReturn.CoverImgURL = title.GetImagePortrait()
	if mangaReturn.CoverImgURL == "" {
		mangaReturn.CoverImgURL = title.GetImageLandscape()
		if mangaReturn.CoverImgURL == "" {
			mangaReturn.CoverImgURL = titleView.GetTitleImageUrl()
			if mangaReturn.CoverImgURL == "" {
				mangaReturn.CoverImgURL = titleView.GetBackgroundImageUrl()
				if mangaReturn.CoverImgURL == "" {
					return nil, util.AddErrorContext(errorContext, fmt.Errorf("manga cover image URL not found"))
				}
			}
		}
	}

	chapters := getChaptersFromAPIList(titleView.GetChapters())
	if len(chapters) == 0 {
		if !ignoreGetLastChapterError {
			return nil, errordefs.ErrChapterNotFound
		}
	} else {
		mangaReturn.LastReleasedChapter = chapters[len(chapters)-1]
		mangaReturn.LastReleasedChapter.Type = 1
	}

	coverImg, resized, err := util.GetImageFromURL(mangaReturn.CoverImgURL, 3, 1*time.Second)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	mangaReturn.CoverImgResized = resized
	mangaReturn.CoverImg = coverImg

	return mangaReturn, nil
}

func getChaptersFromAPIList(titleViewChapters []*TitleDetailView_Chapters) []*manga.Chapter {
	var chaptersReturn []*manga.Chapter

	for _, titleViewChapter := range titleViewChapters {
		for _, protoChapter := range titleViewChapter.FirstChapterList {
			chaptersReturn = append(chaptersReturn, getChapterFromAPIChapter(protoChapter))
		}
		for _, protoChapter := range titleViewChapter.LastChapterList {
			chaptersReturn = append(chaptersReturn, getChapterFromAPIChapter(protoChapter))
		}
	}

	return chaptersReturn
}

func getChapterFromAPIChapter(protoChapter *Chapter) *manga.Chapter {
	url := fmt.Sprintf("%s/viewer/%d", baseSiteURL, protoChapter.GetChapterId())
	chapter := protoChapter.GetTitleName()
	chapter = cleanChapter(chapter)
	if chapter == "" {
		chapter = protoChapter.GetTitleName()
	}

	chapterName := protoChapter.GetChapterSubTitle()
	if chapterName == "" {
		chapterName = chapter
	}

	updatedAt := time.Unix(int64(protoChapter.GetStartTimeStamp()), 0).In(time.Local)

	return &manga.Chapter{
		URL:       url,
		Chapter:   chapter,
		Name:      chapterName,
		UpdatedAt: updatedAt,
	}
}

// getMangaID returns the ID of a manga given its URL.
// URL like "https://mangaplus.shueisha.co.jp/titles/100171" returns "100171".
func getMangaID(mangaURL string) (int, error) {
	parts := strings.Split(mangaURL, "/titles/")
	if len(parts) < 2 {
		return 0, fmt.Errorf("manga ID not found in the URL, URL should be like 'https://mangaplus.shueisha.co.jp/titles/100171'")
	}

	mangaIDStr := strings.TrimSpace(parts[len(parts)-1])
	mangaID, err := strconv.Atoi(mangaIDStr)
	if err != nil {
		return 0, fmt.Errorf("manga ID not found in the URL, URL should be like 'https://mangaplus.shueisha.co.jp/titles/100171'")
	}

	return mangaID, nil
}
