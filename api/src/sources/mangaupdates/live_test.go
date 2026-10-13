//go:build live

package mangaupdates

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/livetest"
)

func TestLiveSource(t *testing.T) {
	s := &Source{}
	for _, target := range []string{"https://www.mangaupdates.com/series/1ljv3bs/death-note", "https://www.mangaupdates.com/series/su6blie/vagabond"} {
		t.Run(target, func(t *testing.T) { livetest.CheckLiveSource(t, s, target) })
	}
}
