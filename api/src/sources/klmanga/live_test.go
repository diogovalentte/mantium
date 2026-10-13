//go:build live

package klmanga

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/livetest"
)

func TestLiveSource(t *testing.T) {
	s := &Source{}
	for _, target := range []string{siteURL.Get() + "/manga-raw/%e3%82%a2%e3%83%b3%e3%83%81%e3%83%ad%e3%83%9e%e3%83%b3%e3%82%b9-raw-free/", siteURL.Get() + "/manga-raw/%e3%82%a2%e3%82%a4%e3%83%9e%e3%82%a4%e3%83%9f%e3%83%bc%e3%83%9e%e3%82%a4%e3%83%b3-raw-free/", siteURL.Get() + "/manga-raw/%e6%80%9d%e3%81%88%e3%81%b0%e9%81%a0%e3%81%8f%e3%81%ab%e3%82%aa%e3%83%96%e3%82%b9%e3%82%af%e3%83%a9-raw-free/"} {
		t.Run(target, func(t *testing.T) { livetest.CheckLiveSource(t, s, target) })
	}
}
