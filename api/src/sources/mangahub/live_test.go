//go:build live

package mangahub

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/livetest"
)

func TestLiveSource(t *testing.T) {
	s := &Source{}
	for _, target := range []string{"https://mangahub.io/manga/death-note_119", "https://mangahub.io/manga/vagabond_119", "https://mangahub.io/manga/mob-psycho-100"} {
		t.Run(target, func(t *testing.T) { livetest.CheckLiveSource(t, s, target) })
	}
}
