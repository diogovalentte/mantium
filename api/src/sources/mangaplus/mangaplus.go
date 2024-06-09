package mangaplus

// Source is the struct for a mangaplus source
type Source struct {
	client *Client
}

var (
	sourceName      = "mangaplus.shueisha.co.jp"
	baseSiteURL     = "https://mangaplus.shueisha.co.jp"
	baseAPIURL      = "https://jumpg-webapi.tokyo-cdn.com/api"
	mangaplusClient = NewMangaPlusClient()
)

func (s *Source) checkClient() {
	if s.client == nil {
		s.client = mangaplusClient
	}
}
