// Package rawkuma provides the implementation of the manga.Source interface for the Rawkuma source
package rawkuma

import (
	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/util"
)

// siteURL starts at the last known domain. The source has moved before, so
// the sources package follows its redirects to find the current one.
var siteURL = util.NewSiteURL("https://rawkuma.net")

// Source is the struct for the Rawkuma source.
// It holds no mutable state: the Sources registry shares one instance across
// every request, so a collector kept on the struct would be swapped and
// visited by concurrent goroutines, mixing up results between mangas.
type Source struct{}

func (Source) GetName() string {
	return "rawkuma"
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
