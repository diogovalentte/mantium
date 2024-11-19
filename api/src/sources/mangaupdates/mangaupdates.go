package mangaupdates

var (
	baseSiteURL        = "https://www.mangaupdates.com"
	baseAPIURL         = "https://api.mangaupdates.com"
	baseUploadsURL     = "https://cdn.mangaupdates.com"
	mangaUpdatesClient = NewMangaUpdatesClient()
)

// Source is the implementation of the manga.Source interface for the MangaUpdates source
type Source struct {
	client *Client
}

func (Source) GetName() string {
	return "mangaupdates"
}

func (s *Source) checkClient() {
	if s.client == nil {
		s.client = mangaUpdatesClient
	}
}
