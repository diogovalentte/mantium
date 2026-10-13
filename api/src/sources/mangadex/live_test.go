//go:build live

package mangadex

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/livetest"
)

func TestLiveSource(t *testing.T) {
	s := &Source{}
	for _, target := range []string{"https://mangadex.org/title/75ee72ab-c6bf-4b87-badd-de839156934c", "https://mangadex.org/title/d1a9fdeb-f713-407f-960c-8326b586e6fd/vagabond", "https://mangadex.org/title/736a2bf0-f875-4b52-a7b4-e8c40505b68a/mob-psycho-100"} {
		t.Run(target, func(t *testing.T) { livetest.CheckLiveSource(t, s, target) })
	}
}
