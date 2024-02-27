// Package comick provides the implementation of the manga.Source interface for the Comick source
// API doc: https://api.comick.xyz/docs/static/index.html
// The site and API URL can change without any warning!!! Because of that, the site and API URLs need to be updated manually!
// The text comick.xyz is used in some parts to indicate the source without needing to change the it in the future.
package comick

import (
	"github.com/gocolly/colly/v2"
)

var (
	baseSiteURL    = "https://comick.io"
	baseAPIURL     = "https://api.comick.io"
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
