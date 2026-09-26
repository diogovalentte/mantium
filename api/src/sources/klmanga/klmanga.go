// Package klmanga provides the implementation of the manga.Source interface for the KLManga source
package klmanga

import (
	"fmt"
	"regexp"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/util"
)

// siteURL starts at the last known domain. The source moves every few weeks,
// so the sources package follows its redirects to find the current one.
var siteURL = util.NewSiteURL("https://klmanga.zone")

// Source is the struct for the KLManga source
type Source struct{}

func (Source) GetName() string {
	return "klmanga"
}

// SiteURL returns the base URL of the source.
func (Source) SiteURL() *util.SiteURL {
	return siteURL
}

var userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"

func newCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent(userAgent),
	)
	c.SetRequestTimeout(util.ExternalRequestTimeout)

	return c
}

func extractChapter(s string) (string, error) {
	re := regexp.MustCompile(`第(.*?)話`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 1 {
		return matches[1], nil
	}
	return "", fmt.Errorf("could not extract chapter number from %s", s)
}
