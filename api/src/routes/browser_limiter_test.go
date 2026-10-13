package routes

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/diogovalentte/mantium/api/src/manga"
)

// maxConcurrentScrapes runs scrapes concurrent scrapes through the limiter and returns how many ran at the same time.
func maxConcurrentScrapes(limiter chan struct{}, useBrowser bool, scrapes int) int32 {
	var running, maxRunning atomic.Int32
	var wg sync.WaitGroup
	for range scrapes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			limitBrowserScrape(limiter, useBrowser, func() (*manga.Chapter, error) {
				current := running.Add(1)
				for {
					old := maxRunning.Load()
					if current <= old || maxRunning.CompareAndSwap(old, current) {
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
				running.Add(-1)
				return nil, nil
			})
		}()
	}
	wg.Wait()
	return maxRunning.Load()
}

func TestBrowserLimiter(t *testing.T) {
	for parallelJobs, want := range map[int]int32{1: 1, 2: 1, 4: 2, 5: 2} {
		if got := maxConcurrentScrapes(newBrowserLimiter(parallelJobs), true, 8); got != want {
			t.Errorf("parallel jobs %d: %d browser scrapes ran at the same time, want %d", parallelJobs, got, want)
		}
	}

	if got := maxConcurrentScrapes(newBrowserLimiter(2), false, 8); got != 8 {
		t.Errorf("scrapes without the browser should not be limited, %d ran at the same time, want 8", got)
	}
}
