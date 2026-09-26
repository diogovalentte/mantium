package sources

import (
	"maps"
	"slices"
	"testing"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/util"
)

// config.SourcesList used to be a second, hand-maintained copy of the Sources
// keys, with a comment asking whoever added a source to remember to update it.
// It is now filled from the registry at init; this pins that down.
func TestSourcesListMatchesTheRegistry(t *testing.T) {
	expected := slices.Sorted(maps.Keys(Sources))

	if !slices.Equal(config.SourcesList, expected) {
		t.Fatalf("config.SourcesList = %v, want %v", config.SourcesList, expected)
	}
}

// A ForcedSourcesURLs entry is typed by hand before a release, so catch a
// trailing slash, a path or a typo in the source name before it ships.
func TestForcedSourcesURLsAreValid(t *testing.T) {
	for name, forced := range ForcedSourcesURLs {
		if _, ok := Sources[name].(movingSource); !ok {
			t.Errorf("ForcedSourcesURLs has %q, which is not a source whose domain moves", name)
		}
		normalized, err := util.NormalizeBaseURL(forced)
		if err != nil {
			t.Errorf("ForcedSourcesURLs[%q] = %q is not a valid URL: %s", name, forced, err)
			continue
		}
		if normalized != forced {
			t.Errorf("ForcedSourcesURLs[%q] = %q, write it as %q", name, forced, normalized)
		}
	}
}
