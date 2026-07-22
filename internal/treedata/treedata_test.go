package treedata_test

import (
	"testing"

	pt "github.com/pedrobalen/poe-build-overlay/internal/passive_tree"
	"github.com/pedrobalen/poe-build-overlay/internal/treedata"
)

// TestBundledTreesLoad verifies the embedded tree data parses for every shipped
// version and carries mastery effects.
func TestBundledTreesLoad(t *testing.T) {
	src := pt.NewJSONSource(treedata.FS, treedata.Dir)

	for _, version := range []string{"3_28", "3_29"} {
		if !src.Available(version) {
			t.Fatalf("bundled tree %s is not available", version)
		}

		data, err := src.Import(version)
		if err != nil {
			t.Fatalf("import %s: %v", version, err)
		}
		if len(data.Nodes) < 1000 {
			t.Fatalf("%s has too few nodes: %d", version, len(data.Nodes))
		}
		if len(data.Connections) < 1000 {
			t.Fatalf("%s has too few connections: %d", version, len(data.Connections))
		}

		masteriesWithEffects := 0
		for _, n := range data.Nodes {
			if n.IsMastery && len(n.Effects) > 0 {
				masteriesWithEffects++
			}
		}
		if masteriesWithEffects == 0 {
			t.Fatalf("%s has no mastery effects", version)
		}
	}
}
