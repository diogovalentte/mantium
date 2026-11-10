// Package mangahub implements the mangahub.io source.
//
// MangaHub's site and API are behind Cloudflare, and its GraphQL API refuses
// clients that don't do the site's own encryption handshake. So the source
// scrapes the site's pages, getting them through FlareSolverr when
// FLARESOLVERR_ADDRESS is set. Without it, the site answers 403.
package mangahub

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/errordefs"
	"github.com/diogovalentte/mantium/api/src/flaresolverr"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

var baseSiteURL = "https://mangahub.io"

// Source is the struct for a mangahub.io source
// It holds no mutable state: the Sources registry shares one instance
// across every request, so any field written per call would be a race.
type Source struct{}

func (Source) GetName() string {
	return "mangahub"
}

// directTransport is used when FlareSolverr isn't configured. MangaHub used to
// refuse TLS 1.3 clients, so it's capped at TLS 1.2.
var directTransport = &http.Transport{
	Proxy:           http.ProxyFromEnvironment,
	TLSClientConfig: &tls.Config{MaxVersion: tls.VersionTLS12},
}

// newCollector returns a collector that gets the pages through FlareSolverr if it's configured.
func newCollector() *colly.Collector {
	c := colly.NewCollector(colly.UserAgent(util.DefaultUserAgent))
	if flaresolverr.Enabled() {
		c.WithTransport(flaresolverr.Transport(flaresolverr.DefaultTimeout))
		// The collector timeout covers the whole request to FlareSolverr, so it's longer than FlareSolverr's.
		c.SetRequestTimeout(flaresolverr.DefaultTimeout + util.ExternalRequestTimeout)
	} else {
		c.WithTransport(directTransport)
		c.SetRequestTimeout(util.ExternalRequestTimeout)
	}

	return c
}

// errBlocked is returned when MangaHub refuses the request and FlareSolverr isn't configured.
var errBlocked = &errordefs.UserError{Message: "MangaHub refused the request, as its site is behind Cloudflare. Set FLARESOLVERR_ADDRESS in the API to get its pages through FlareSolverr"}

// visitError translates the error of visiting a MangaHub page.
func visitError(err error) error {
	switch {
	case err.Error() == "Not Found":
		return errordefs.ErrMangaNotFound
	case err.Error() == "Forbidden" && !flaresolverr.Enabled():
		return errBlocked
	}
	var userErr *errordefs.UserError
	if errors.As(err, &userErr) {
		return err
	}
	return util.AddErrorContext("error while visiting MangaHub page", err)
}

// mangaPage is what the manga page has.
type mangaPage struct {
	name     string
	coverURL string
	// chapters are in the page's order, newest first.
	chapters []*manga.Chapter
}

// getMangaPage gets the manga page and scrapes it.
func getMangaPage(mangaURL string) (*mangaPage, error) {
	c := newCollector()
	page := &mangaPage{}

	c.OnHTML("h1._3xnDj", func(e *colly.HTMLElement) {
		// The h1 also has the alternative names in a <small> and labels like "Hot" in an <a>
		h1 := e.DOM.Clone()
		h1.Find("small, a").Remove()
		page.name = strings.TrimSpace(h1.Text())
	})

	c.OnHTML("img.manga-thumb", func(e *colly.HTMLElement) {
		if page.coverURL == "" {
			page.coverURL = e.Request.AbsoluteURL(e.Attr("src"))
		}
	})

	// The chapters are split in tabs of 100 chapters, one list per tab, all in the page.
	// Each item can also link to other uploads of the same chapter (a._1AxFv), which are skipped.
	seen := map[string]bool{}
	c.OnHTML("ul.MWqeC > li a._3pfyN", func(e *colly.HTMLElement) {
		number := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(e.ChildText("span._3D1SJ")), "#"))
		if number == "" || seen[number] {
			return
		}
		seen[number] = true

		name := strings.TrimSpace(e.ChildText("span._2IG5P"))
		name = strings.TrimSpace(strings.TrimPrefix(name, "-"))
		if name == "" {
			name = "Chapter " + number
		}

		page.chapters = append(page.chapters, &manga.Chapter{
			URL:       e.Request.AbsoluteURL(e.Attr("href")),
			Chapter:   number,
			Name:      name,
			UpdatedAt: parseReleaseTime(e.ChildText("small.UovLc")),
		})
	})

	if err := c.Visit(mangaURL); err != nil {
		return nil, visitError(err)
	}
	if page.name == "" {
		return nil, errordefs.ErrMangaAttributesNotFound
	}

	return page, nil
}

// now is time.Now, replaced in tests.
var now = time.Now

var relativeTimeRe = regexp.MustCompile(`^(\d+|an?) (minute|hour|day|week|month|year)s? ago$`)

// parseReleaseTime parses the chapter release time, like "01-19-2016", "4 days ago" or "Yesterday".
// It's the zero time if it's missing or in a format it doesn't know, which only affects the date shown.
func parseReleaseTime(text string) time.Time {
	text = strings.TrimSpace(text)
	if text == "" {
		return time.Time{}
	}
	if t, err := time.Parse("01-02-2006", text); err == nil {
		return t
	}

	current := now()
	today := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
	switch strings.ToLower(text) {
	case "just now":
		return current.Truncate(time.Second)
	case "less than an hour", "less than an hour ago":
		return current.Add(-30 * time.Minute).Truncate(time.Second)
	case "yesterday":
		return today.AddDate(0, 0, -1)
	}

	matches := relativeTimeRe.FindStringSubmatch(strings.ToLower(text))
	if matches == nil {
		return time.Time{}
	}
	amount := 1
	if matches[1] != "a" && matches[1] != "an" {
		amount, _ = strconv.Atoi(matches[1])
	}
	switch matches[2] {
	case "minute":
		return current.Add(-time.Duration(amount) * time.Minute).Truncate(time.Second)
	case "hour":
		return current.Add(-time.Duration(amount) * time.Hour).Truncate(time.Second)
	case "day":
		return today.AddDate(0, 0, -amount)
	case "week":
		return today.AddDate(0, 0, -7*amount)
	case "month":
		return today.AddDate(0, -amount, 0)
	default:
		return today.AddDate(-amount, 0, 0)
	}
}

// findChapter returns the chapter with the given number, or else with the given URL.
func findChapter(chapters []*manga.Chapter, number, chapterURL string) (*manga.Chapter, error) {
	if number != "" {
		want, err := strconv.ParseFloat(number, 64)
		for _, c := range chapters {
			if c.Chapter == number {
				return c, nil
			}
			if got, gotErr := strconv.ParseFloat(c.Chapter, 64); err == nil && gotErr == nil && got == want {
				return c, nil
			}
		}
	}
	if chapterURL != "" {
		for _, c := range chapters {
			if c.URL == chapterURL {
				return c, nil
			}
		}
	}

	return nil, errordefs.ErrChapterNotFound
}

// errNoChapters is used when the manga page lists no chapters.
var errNoChapters = fmt.Errorf("manga has no chapters: %w", errordefs.ErrChapterNotFound)
