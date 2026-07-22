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

func TestDiffGems(t *testing.T) {
	prev := []SkillGroup{{Gems: []Gem{{Name: "Ground Slam"}, {Name: "Melee Physical Damage Support"}}}}
	curr := []SkillGroup{{Gems: []Gem{{Name: "Static Strike"}, {Name: "Melee Physical Damage Support"}}}}

	changes := DiffGems(prev, curr)
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d: %+v", len(changes), changes)
	}
	// Additions sort before removals.
	if changes[0].Kind != GemAdded || changes[0].Name != "Static Strike" {
		t.Fatalf("unexpected first change: %+v", changes[0])
	}
	if changes[1].Kind != GemRemoved || changes[1].Name != "Ground Slam" {
		t.Fatalf("unexpected second change: %+v", changes[1])
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

func TestNextGemChangeStage(t *testing.T) {
	build := Build{Stages: []BuildStage{
		{SkillGroups: []SkillGroup{{Gems: []Gem{{Name: "A"}}}}},
		{SkillGroups: []SkillGroup{{Gems: []Gem{{Name: "A"}}}}},
		{SkillGroups: []SkillGroup{{Gems: []Gem{{Name: "A"}, {Name: "B"}}}}},
	}}

	next, ok := build.NextGemChangeStage(0)
	if !ok || next != 2 {
		t.Fatalf("NextGemChangeStage(0) = %d,%v; want 2,true", next, ok)
	}
	if _, ok := build.NextGemChangeStage(2); ok {
		t.Fatal("expected no further gem change after last stage")
	}
}
