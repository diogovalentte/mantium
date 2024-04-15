// Package mangahub implements the mangahub.io source.
// I hate scraping the mangahub site. It has Many problems.
// I just like the fact that it uses Disqus for comments instead of Facebook
// and has some mangas that are not available in other sources.
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
)

// Source is the struct for a mangahub.io source
type Source struct {
	c *colly.Collector
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
			"less than an hour": func(timeString string) (time.Time, error) {
				subHalfHour := time.Duration(30) * time.Minute
				releaseDate := time.Now().Add(subHalfHour * -1)
				return releaseDate, nil
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
				return releaseDate, nil
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
