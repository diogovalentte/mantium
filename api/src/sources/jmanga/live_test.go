//go:build live

package jmanga

import (
	"testing"

	"github.com/diogovalentte/mantium/api/internal/livetest"
)

func TestLiveSource(t *testing.T) {
	s := &Source{}
	for _, target := range []string{siteURL.Get() + "/read/%E3%82%A2%E3%83%B3%E3%83%81%E3%83%AD%E3%83%9E%E3%83%B3%E3%82%B9-raw/", siteURL.Get() + "/read/%E3%82%A2%E3%82%A4%E3%83%9E%E3%82%A4%E3%83%9F%E3%83%BC%E3%83%9E%E3%82%A4%E3%83%B3-raw/", siteURL.Get() + "/read/%E6%80%9D%E3%81%88%E3%81%B0%E9%81%A0%E3%81%8F%E3%81%AB%E3%82%AA%E3%83%96%E3%82%B9%E3%82%AF%E3%83%A9-raw/"} {
		t.Run(target, func(t *testing.T) { livetest.CheckLiveSource(t, s, target) })
	}
}
