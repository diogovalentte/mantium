// Package rawkuma provides the implementation of the manga.Source interface for the Rawkuma source
package rawkuma

import (
	"github.com/gocolly/colly/v2"
)

var baseSiteURL = "https://rawkuma.net"

// Source is the struct for the Rawkuma source.
// It holds no mutable state: the Sources registry shares one instance across
// every request, so a collector kept on the struct would be swapped and
// visited by concurrent goroutines, mixing up results between mangas.
type Source struct{}

func (Source) GetName() string {
	return "rawkuma"
}

var userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"

func newCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent(userAgent),
	)

	return c
}
