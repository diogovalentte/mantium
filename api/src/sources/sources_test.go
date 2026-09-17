package sources

import (
	"maps"
	"slices"
	"testing"

	"github.com/diogovalentte/mantium/api/src/config"
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
