// Package mangahub implements the mangahub.io source
// I hate scraping the mangahub site.
// Many problems, I just like the fact
// that it uses Disqus for comments instead of Facebook
package mangahub

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/manga-dashboard-api/src/manga"
	"github.com/diogovalentte/manga-dashboard-api/src/util"
)

// Source is the struct for a mangahub.io source
type Source struct {
	c *colly.Collector
}

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL string) (*manga.Manga, error) {
	s.resetCollector()

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "mangahub.io"
	mangaReturn.URL = mangaURL

	lastChapter := &manga.Chapter{
		Type: 1,
	}
	mangaReturn.LastUploadChapter = lastChapter

	var sharedErr error

	// manga name
	s.c.OnHTML("h1._3xnDj", func(e *colly.HTMLElement) {
		// The h1 tag with the manga's name
		// has a small tag inside it with the
		// manga description that we don't want.
		// It can also have an <a> tag with the
		// manga's name and the word "Hot".
		name := e.Text
		smallTagValue := e.DOM.Find("small").Text()
		aTagValue := e.DOM.Find("a").Text()
		name = strings.Replace(name, smallTagValue, "", -1)
		name = util.RemoveLastOccurrence(name, aTagValue)

		mangaReturn.Name = name
	})

	// manga cover
	s.c.OnHTML("img.manga-thumb", func(e *colly.HTMLElement) {
		mangaReturn.CoverImgURL = e.Attr("src")
	})

	// last chapter
	isFirstUL := true
	s.c.OnHTML("ul.MWqeC:first-of-type > li:first-child a", func(e *colly.HTMLElement) {
		if !isFirstUL {
			return
		}
		isFirstUL = false
		lastChapter.URL = e.Attr("href")

		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		chapterNumber, err := strconv.ParseFloat(strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1)), 32)
		if err != nil {
			sharedErr = err
			return
		}
		lastChapter.Number = manga.Number(chapterNumber)

		chapterName := e.DOM.Find("span._2IG5P").Text()
		lastChapter.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		lastChapter.UpdatedAt = uploadedTime
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, fmt.Errorf("manga not found, is the URL correct?")
		}
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}

	// get cover image
	coverImg, err := s.getCoverImg(mangaReturn.CoverImgURL)
	mangaReturn.CoverImg = coverImg

	return mangaReturn, nil
}

// GetChaptersMetadata scrapes the manga page and return the chapters
func (s *Source) GetChaptersMetadata(mangaURL string) ([]*manga.Chapter, error) {
	s.resetCollector()
	chapters := []*manga.Chapter{}

	var sharedErr error
	s.c.OnHTML("li._287KE a._3pfyN", func(e *colly.HTMLElement) {
		chapter := &manga.Chapter{}

		chapter.URL = e.Attr("href")

		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		chapterNumber, err := strconv.ParseFloat(strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1)), 32)
		if err != nil {
			sharedErr = err
			return
		}
		chapter.Number = manga.Number(chapterNumber)

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapter.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		chapter.UpdatedAt = uploadedTime

		chapters = append(chapters, chapter)
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}

	return chapters, nil
}

// GetChapterMetadata scrapes the manga page and return the chapter
func (s *Source) GetChapterMetadata(mangaURL string, chapterNumber manga.Number) (*manga.Chapter, error) {
	s.resetCollector()
	chapter := &manga.Chapter{
		Number: chapterNumber,
	}
	var sharedErr error

	chapterFound := false
	s.c.OnHTML("ul.MWqeC:first-of-type > li a._3pfyN", func(e *colly.HTMLElement) {
		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		scrapedChapterNumber, err := strconv.ParseFloat(strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1)), 32)
		if err != nil {
			sharedErr = err
			return
		}
		if manga.Number(scrapedChapterNumber) != chapterNumber {
			return
		}
		chapterFound = true

		chapter.URL = e.Attr("href")

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapter.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		chapter.UpdatedAt = uploadedTime
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}
	if !chapterFound {
		return nil, fmt.Errorf("chapter not found, is the URL or chapter number correct?")
	}

	return chapter, nil
}

// GetLastChapterMetadata scrapes the manga page and return the latest chapter
func (s *Source) GetLastChapterMetadata(mangaURL string) (*manga.Chapter, error) {
	s.resetCollector()
	chapter := &manga.Chapter{}
	var sharedErr error

	isFirstUL := true
	s.c.OnHTML("ul.MWqeC:first-of-type > li:first-child a._3pfyN", func(e *colly.HTMLElement) {
		if !isFirstUL {
			return
		}
		isFirstUL = false
		chapter.URL = e.Attr("href")

		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		chapterNumber, err := strconv.ParseFloat(strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1)), 32)
		if err != nil {
			sharedErr = err
			return
		}
		chapter.Number = manga.Number(chapterNumber)

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapter.Name = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}
		chapter.UpdatedAt = uploadedTime
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		return nil, err
	}
	if sharedErr != nil {
		return nil, sharedErr
	}

	return chapter, nil
}

var userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"

func newCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains("mangahub.io"),
		colly.UserAgent(userAgent),
	)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MaxVersion: tls.VersionTLS12,
		},
	}
	c.WithTransport(transport)

	return c
}

func (s *Source) resetCollector() {
	if s.c != nil {
		s.c.Wait()
	}

	s.c = newCollector()
}

func (s *Source) getCoverImg(url string) ([]byte, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second, // xD
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MaxVersion: tls.VersionTLS12,
			},
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf(`error creating request to download cover image at url "%s": %s`, url, err)
		return nil, err

	}

	req.Header.Set("User-Agent", "Custom User Agent")

	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf(`error performing request to download manga cover image at url "%s": %s`, url, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf(`unexpected status code while download manga cover image at turl "%s": %d`, url, resp.StatusCode)
		return nil, err
	}

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf(`error reading image data at url "%s": %s`, url, err)
		return nil, err
	}

	return imageBytes, nil
}

func getMangaUploadedTime(timeString string) (time.Time, error) {
	layout := "01-02-2006"
	parsedTime, err := time.Parse(layout, timeString)
	if err != nil {
		patternsToCheck := map[string]func(string) (time.Time, error){
			"just now": func(timeString string) (time.Time, error) {
				return time.Now(), nil
			},
			"1 hour ago": func(timeString string) (time.Time, error) {
				subOneHour := time.Duration(1) * time.Hour
				releaseDate := time.Now().Add(subOneHour * -1)
				return releaseDate, nil
			},
			"hours ago": func(timeString string) (time.Time, error) {
				hours, err := strconv.Atoi(strings.TrimSpace(strings.Replace(timeString, "hours ago", "", -1)))
				if err != nil {
					return time.Time{}, err
				}
				subHours := time.Duration(hours) * time.Hour
				releaseDate := time.Now().Add(subHours * -1)
				return time.Date(releaseDate.Year(), releaseDate.Month(), releaseDate.Day(), 0, 0, 0, 0, time.UTC), nil
			},
			"Yesterday": func(timeString string) (time.Time, error) {
				yesterday := time.Now().Add(time.Hour * -24)
				return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC), nil
			},
			"days ago": func(timeString string) (time.Time, error) {
				days, err := strconv.Atoi(strings.TrimSpace(strings.Replace(timeString, "days ago", "", -1)))
				if err != nil {
					return time.Time{}, err
				}
				subDays := time.Duration(days) * time.Hour * 24
				releaseDate := time.Now().Add(subDays * -1)
				return time.Date(releaseDate.Year(), releaseDate.Month(), releaseDate.Day(), 0, 0, 0, 0, time.UTC), nil
			},
			"1 week ago": func(timeString string) (time.Time, error) {
				subOneWeek := time.Duration(1) * time.Hour * 24 * 7
				releaseDate := time.Now().Add(subOneWeek * -1)
				return time.Date(releaseDate.Year(), releaseDate.Month(), releaseDate.Day(), 0, 0, 0, 0, time.UTC), nil
			},
			"weeks ago": func(timeString string) (time.Time, error) {
				weeks, err := strconv.Atoi(strings.TrimSpace(strings.Replace(timeString, "weeks ago", "", -1)))
				if err != nil {
					return time.Time{}, err
				}
				subWeeks := time.Duration(weeks) * time.Hour * 24 * 7
				releaseDate := time.Now().Add(subWeeks * -1)
				return time.Date(releaseDate.Year(), releaseDate.Month(), releaseDate.Day(), 0, 0, 0, 0, time.UTC), nil
			},
		}
		for pattern, action := range patternsToCheck {
			if strings.Contains(timeString, pattern) {
				parsedTime, err = action(timeString)
				if err == nil {
					return parsedTime, nil
				}
			}
		}

		return time.Time{}, err
	}

	return parsedTime, nil
}
