package rawkuma

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/sources/models"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL, _ string) (*manga.Manga, error) {
	s.resetCollector()

	errorContext := "error while getting manga metadata"

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "rawkuma"
	mangaReturn.URL = mangaURL

	var sharedErr error

	// manga name
	s.c.OnHTML("h1[itemprop='name']", func(e *colly.HTMLElement) {
		mangaReturn.Name = strings.TrimSpace(e.Text)
	})

	// manga cover
	s.c.OnHTML("article img.wp-post-image", func(e *colly.HTMLElement) {
		coverURL := e.Attr("src")

		coverImg, resized, err := util.GetImageFromURL(coverURL, 3, 1*time.Second)
		if err == nil {
			mangaReturn.CoverImgURL = coverURL
			mangaReturn.CoverImgResized = resized
			mangaReturn.CoverImg = coverImg
		}
	})

	// last released chapter
	s.c.OnResponse(func(r *colly.Response) {
		body := string(r.Body)
		re := regexp.MustCompile(`wp-admin/admin-ajax\.php\?manga_id=(\d+)(?:&|$)`)
		HTMLMangaID := re.FindStringSubmatch(body)
		if len(HTMLMangaID) <= 1 {
			sharedErr = fmt.Errorf("manga ID not found in HTML response")
			return
		}

		var err error
		mangaReturn.InternalID = HTMLMangaID[1]
		mangaReturn.LastReleasedChapter, err = s.GetLastChapterMetadata("", HTMLMangaID[1])
		mangaReturn.LastReleasedChapter.Type = 1
		if err != nil {
			sharedErr = err
		}
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
		}
		return nil, util.AddErrorContext(errorContext, util.AddErrorContext("error while visiting manga URL", err))
	}
	if sharedErr != nil {
		return nil, util.AddErrorContext(errorContext, sharedErr)
	}
	if mangaReturn.Name == "" {
		return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
	}

	return mangaReturn, nil
}

func (s *Source) Search(term string, limit int) ([]*models.MangaSearchResult, error) {
	errorContext := "error while searching manga"
	mangaSearchResults := []*models.MangaSearchResult{}
	pageNumber := 1
	var mangaCount int

	for mangaCount < limit {
		searchURL := fmt.Sprintf("%s/wp-admin/admin-ajax.php?action=advanced_search", baseSiteURL)

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField("query", term)
		w.WriteField("orderby", "popular")
		w.WriteField("order", "desc")
		w.WriteField("page", fmt.Sprintf("%d", pageNumber))
		w.Close()

		req, err := http.NewRequest(http.MethodPost, searchURL, &b)
		if err != nil {
			return nil, util.AddErrorContext(errorContext, util.AddErrorContext("error creating search request", err))
		}
		req.Header.Set("Content-Type", w.FormDataContentType())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, util.AddErrorContext(errorContext, util.AddErrorContext("error performing search request", err))
		}
		switch resp.StatusCode {
		case http.StatusOK:
		case http.StatusNotFound:
			return nil, util.AddErrorContext(errorContext, errordefs.ErrMangaNotFound)
		default:
			return nil, util.AddErrorContext(errorContext, fmt.Errorf("unexpected status code: %d", resp.StatusCode))
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return nil, util.AddErrorContext(errorContext, util.AddErrorContext("error parsing search response body", err))
		}

		doc.Find("div > div > a > img").Each(func(_ int, s *goquery.Selection) {
			if mangaCount >= limit {
				return
			}
			s = s.Parent().Parent()
			mangaSearchResult := &models.MangaSearchResult{}
			mangaSearchResult.Source = "rawkuma"
			mangaSearchResult.URL = s.Find("a").AttrOr("href", "")
			mangaSearchResult.Description = s.Find("div > div > p").Text()
			mangaSearchResult.Name = strings.TrimSpace(s.Find("div > div > div > div > a").Text())
			mangaSearchResult.CoverURL = s.Find("a > img").AttrOr("src", "")
			if mangaSearchResult.CoverURL == "" {
				mangaSearchResult.CoverURL = models.DefaultCoverImgURL
			}

			mangaSearchResult.LastChapter = strings.TrimPrefix(s.Find("div > div > div > div > span:first-child").Text(), "Chapter ")
			if mangaSearchResult.LastChapter != "" {
				mangaSearchResult.LastChapterURL = mangaSearchResult.URL
			}
			mangaSearchResult.Status = s.Find("div > div > div > div > span:last-child").First().Text()

			mangaSearchResults = append(mangaSearchResults, mangaSearchResult)
			mangaCount++
		})

		buttonCount := 0
		doc.Find("button polyline").Each(func(_ int, _ *goquery.Selection) {
			buttonCount++
		})

		if buttonCount < 2 && pageNumber > 1 {
			break
		}
		pageNumber++
	}

	return mangaSearchResults, nil
}
