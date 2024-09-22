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
	"github.com/diogovalentte/mantium/api/src/sources/models"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL, _ string) (*manga.Manga, error) {
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

	coverImgURL := title.GetImagePortrait()
	if coverImgURL == "" {
		coverImgURL = title.GetImageLandscape()
		if coverImgURL == "" {
			coverImgURL = titleView.GetTitleImageUrl()
			if coverImgURL == "" {
				coverImgURL = titleView.GetBackgroundImageUrl()
			}
		}
	}

	chapters := getChaptersFromAPIList(titleView.GetChapters())
	if len(chapters) > 0 {
		mangaReturn.LastReleasedChapter = chapters[0]
		mangaReturn.LastReleasedChapter.Type = 1
	}

	if coverImgURL != "" {
		coverImg, resized, err := util.GetImageFromURL(coverImgURL, 3, 1*time.Second)
		if err == nil {
			mangaReturn.CoverImgURL = coverImgURL
			mangaReturn.CoverImgResized = resized
			mangaReturn.CoverImg = coverImg
		}
	}

	return mangaReturn, nil
}

func (s *Source) Search(term string, limit int) ([]*models.MangaSearchResult, error) {
	s.checkClient()

	errorContext := "error while searching manga"

	_, response, err := s.client.Request(fmt.Sprintf("%s/title_list/allV2", baseAPIURL))
	if err != nil {
		if util.ErrorContains(err, "non-200 status code -> (404)") {
			return nil, errordefs.ErrMangaNotFound
		}
		return nil, util.AddErrorContext(errorContext, err)
	}

	titlesGroup := response.GetSuccess().GetAllTitlesViewV2().GetAllTitlesGroup()
	mangaSearchResults := make([]*models.MangaSearchResult, 0, len(titlesGroup))
	count := 0
	for _, titleGroup := range titlesGroup {
		if count >= limit {
			break
		}
		title := titleGroup.GetTitles()[0]
		titleName := title.GetTitleName()
		if strings.Contains(strings.ToLower(titleName), strings.ToLower(term)) {
			coverImgURL := title.GetImagePortrait()
			if coverImgURL == "" {
				coverImgURL = models.DefaultCoverImgURL
			}
			mangaSearchResults = append(mangaSearchResults, &models.MangaSearchResult{
				Source:      sourceName,
				URL:         fmt.Sprintf("%s/titles/%d", baseSiteURL, title.GetTitleId()),
				Name:        titleName,
				CoverURL:    coverImgURL,
				LastChapter: "N/A",
			})
			count++
		}
	}

	return mangaSearchResults, nil
}

func getChaptersFromAPIList(titleViewChapters []*TitleDetailView_Chapters) []*manga.Chapter {
	chaptersReturn := make([]*manga.Chapter, 0, 6) // Most manga have only the first and last 3 chapters available

	for i := len(titleViewChapters) - 1; i >= 0; i-- {
		titleViewChapter := titleViewChapters[i]
		for j := len(titleViewChapter.LastChapterList) - 1; j >= 0; j-- {
			protoChapter := titleViewChapter.LastChapterList[j]
			chaptersReturn = append(chaptersReturn, getChapterFromAPIChapter(protoChapter))
		}
		for j := len(titleViewChapter.FirstChapterList) - 1; j >= 0; j-- {
			protoChapter := titleViewChapter.FirstChapterList[j]
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
