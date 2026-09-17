package mangaupdates

var (
	baseSiteURL        = "https://www.mangaupdates.com"
	baseAPIURL         = "https://api.mangaupdates.com"
	baseUploadsURL     = "https://cdn.mangaupdates.com"
	mangaUpdatesClient = NewMangaUpdatesClient()
)

// Source is the implementation of the manga.Source interface for the MangaUpdates source
// It holds no mutable state: the Sources registry shares one instance
// across every request, so any field written per call would be a race.
type Source struct{}

func (Source) GetName() string {
	return "mangaupdates"
}
