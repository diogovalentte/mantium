package mangaplus

// Source is the struct for a mangaplus source
// It holds no mutable state: the Sources registry shares one instance
// across every request, so any field written per call would be a race.
type Source struct{}

func (Source) GetName() string {
	return "mangaplus"
}

var (
	sourceName      = "mangaplus"
	baseSiteURL     = "https://mangaplus.shueisha.co.jp"
	baseAPIURL      = "https://jumpg-webapi.tokyo-cdn.com/api"
	mangaplusClient = NewMangaPlusClient()
)
