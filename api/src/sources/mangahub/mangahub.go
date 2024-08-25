// Package mangahub implements the mangahub.io source.
package mangahub

var (
	baseSiteURL    = "https://mangahub.io"
	baseAPIURL     = "https://api.mghcdn.com/graphql"
	baseUploadsURL = "https://thumb.mghcdn.com"
	userAgent      = "Mozilla/5.0 (X11; Linux x86_64; rv:30.0) Gecko/20100101 Firefox/30.0"
	mangahubClient = NewMangaHubClient()
)

// Source is the struct for a mangahub.io source
type Source struct {
	client *Client
}

func (s *Source) checkClient() {
	if s.client == nil {
		s.client = mangahubClient
	}
}
