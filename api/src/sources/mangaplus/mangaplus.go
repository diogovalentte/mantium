package mangaplus

// Source is the struct for a mangaplus source
type Source struct {
	client *Client
}

func (Source) GetName() string {
	return "mangaplus"
}

var (
	sourceName      = "mangaplus"
	baseSiteURL     = "https://mangaplus.shueisha.co.jp"
	baseAPIURL      = "https://jumpg-webapi.tokyo-cdn.com/api"
	mangaplusClient = NewMangaPlusClient()
)

func (s *Source) checkClient() {
	if s.client == nil {
		s.client = mangaplusClient
	}
}
