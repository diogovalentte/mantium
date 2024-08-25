// Package comick provides the implementation of the manga.Source interface for the Comick source.
// The text "comick.xyz" is used in some parts to indicate the source without specifing the URL TLD, as it changes constantly.
// API doc: https://api.comick.xyz/docs/static/index.html
// The site and API URL can change without any warning!!! Because of that, the site and API URLs need to be updated manually!
package comick

var (
	baseSiteURL    = "https://comick.io"
	baseAPIURL     = "https://api.comick.fun"
	baseUploadsURL = "https://meo.comick.pictures"
	comickClient   = NewComickClient()
)

// Source is the implementation of the manga.Source interface for the Comick source
type Source struct {
	client *Client
}

func (s *Source) checkClient() {
	if s.client == nil {
		s.client = comickClient
	}
}
