//go:build live

package mangahub

import (
	"os"
	"testing"

	"github.com/diogovalentte/mantium/api/internal/livetest"
	"github.com/diogovalentte/mantium/api/src/flaresolverr"
)

// TestLiveSource needs FLARESOLVERR_ADDRESS, as MangaHub refuses plain requests.
func TestLiveSource(t *testing.T) {
	addr := os.Getenv("FLARESOLVERR_ADDRESS")
	if addr == "" {
		t.Skip("FLARESOLVERR_ADDRESS isn't set, and MangaHub refuses requests without FlareSolverr")
	}
	flaresolverr.SetAddress(addr)
	defer flaresolverr.SetAddress("")

	s := &Source{}
	for _, target := range []string{"https://mangahub.io/manga/death-note_119", "https://mangahub.io/manga/vagabond_119", "https://mangahub.io/manga/mob-psycho-100"} {
		t.Run(target, func(t *testing.T) { livetest.CheckLiveSource(t, s, target) })
	}
}
