package builds

import (
	"reflect"
	"testing"
)

func TestDiffNodes(t *testing.T) {
	prev := []int{1, 2, 3}
	curr := []int{2, 3, 4, 5}

	added, removed := DiffNodes(prev, curr)
	if !reflect.DeepEqual(added, []int{4, 5}) {
		t.Fatalf("added = %v, want [4 5]", added)
	}
	if !reflect.DeepEqual(removed, []int{1}) {
		t.Fatalf("removed = %v, want [1]", removed)
	}
}

func TestDiffNodesFirstStage(t *testing.T) {
	added, removed := DiffNodes(nil, []int{7, 8})
	if !reflect.DeepEqual(added, []int{7, 8}) {
		t.Fatalf("added = %v, want [7 8]", added)
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %v, want empty", removed)
	}
}

func TestComputeStageDiffs(t *testing.T) {
	build := Build{Stages: []BuildStage{
		{PassiveNodes: []int{1, 2}},
		{PassiveNodes: []int{1, 2, 3}},
	}}

	computeStageDiffs(&build)

	if !reflect.DeepEqual(build.Stages[0].NewNodes, []int{1, 2}) {
		t.Fatalf("stage 0 new nodes = %v, want [1 2]", build.Stages[0].NewNodes)
	}
	if !reflect.DeepEqual(build.Stages[1].NewNodes, []int{3}) {
		t.Fatalf("stage 1 new nodes = %v, want [3]", build.Stages[1].NewNodes)
	}
}
