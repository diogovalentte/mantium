//go:build live

package rawkuma

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/livetest"
)

func TestLiveSource(t *testing.T) {
	s := &Source{}
	for _, target := range []string{"https://rawkuma.net/manga/futari-switch/", "https://rawkuma.net/manga/shingeki-no-kyojin", "https://rawkuma.net/manga/darling-in-the-franxx"} {
		t.Run(target, func(t *testing.T) { livetest.CheckLiveSource(t, s, target) })
	}
}
