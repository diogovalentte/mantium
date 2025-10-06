// Package rawkuma provides the implementation of the manga.Source interface for the Rawkuma source
package rawkuma

import (
	"github.com/gocolly/colly/v2"
)

var baseSiteURL = "https://rawkuma.net"

// Source is the struct for the Rawkuma source
type Source struct {
	col    *colly.Collector
	client *Client
}

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

func (s *Source) resetCollector() {
	if s.col != nil {
		s.col.Wait()
	}

	s.col = newCollector()
}

func (s *Source) resetAPIClient() {
	s.client = newAPIClient()
}
