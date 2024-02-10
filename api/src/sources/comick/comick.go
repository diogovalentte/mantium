// Package comick provides the implementation of the manga.Source interface for the Comick source
// API doc: https://api.comick.cc/docs/static/index.html
package comick

import (
	"github.com/gocolly/colly/v2"
)

var (
	baseSiteURL    = "https://comick.cc"
	baseAPIURL     = "https://api.comick.cc"
	baseUploadsURL = "https://meo.comick.pictures"
	mangadexClient = NewComickClient()
)

// Source is the implementation of the manga.Source interface for the Comick source
type Source struct {
	client *Client
	col    *colly.Collector
}

func (s *Source) checkClient() {
	if s.client == nil {
		s.client = mangadexClient
	}
}
