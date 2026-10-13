//go:build live

package mangaplus

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/livetest"
)

func TestLiveSource(t *testing.T) {
	s := &Source{}
	for _, target := range []string{"https://mangaplus.shueisha.co.jp/titles/100145", "https://mangaplus.shueisha.co.jp/titles/100171", "https://mangaplus.shueisha.co.jp/titles/100282"} {
		t.Run(target, func(t *testing.T) { livetest.CheckLiveSource(t, s, target) })
	}
}
