// Package mangahub implements the mangahub.io source.
package mangahub

var (
	baseSiteURL    = "https://mangahub.io"
	baseAPIURL     = "https://api.mghcdn.com/graphql"
	baseUploadsURL = "https://thumb.mghcdn.com"
	mangahubClient = NewMangaHubClient()
)

// Source is the struct for a mangahub.io source
// It holds no mutable state: the Sources registry shares one instance
// across every request, so any field written per call would be a race.
type Source struct{}

func (Source) GetName() string {
	return "mangahub"
}
