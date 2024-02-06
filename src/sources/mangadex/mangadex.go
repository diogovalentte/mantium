// Package mangadex provides the implementation of the manga.Source interface for the MangaDex source
// Some of the code of this package is based/copy from https://github.com/darylhjd/mangodex/
package mangadex

var (
	baseSiteURL    = "https://mangadex.org"
	baseAPIURL     = "https://api.mangadex.org"
	baseUploadsURL = "https://uploads.mangadex.org"
	mangadexClient = NewMangadexClient()
)

// Source is the implementation of the manga.Source interface for the MangaDex source
type Source struct {
	client *MangadexClient
}

func (s *Source) checkClient() {
	if s.client == nil {
		s.client = mangadexClient
	}
}
