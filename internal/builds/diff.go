package builds

import "sort"

// DiffNodes returns the nodes added and removed going from prev to curr.
//
//	added   = curr - prev
//	removed = prev - curr
//
// Both results are sorted for stable display and storage.
func DiffNodes(prev, curr []int) (added, removed []int) {
	prevSet := make(map[int]struct{}, len(prev))
	for _, n := range prev {
		prevSet[n] = struct{}{}
	}

	currSet := make(map[int]struct{}, len(curr))
	for _, n := range curr {
		currSet[n] = struct{}{}
	}

	added = make([]int, 0)
	for _, n := range curr {
		if _, ok := prevSet[n]; !ok {
			added = append(added, n)
		}
	}

	removed = make([]int, 0)
	for _, n := range prev {
		if _, ok := currSet[n]; !ok {
			removed = append(removed, n)
		}
	}

	sort.Ints(added)
	sort.Ints(removed)

	return added, removed
}

// DiffGems compares the gem sets of two stages by name and returns the changes.
// It reports only additions and removals; the app deliberately never judges
// whether a change is an improvement (that is the build author's intent).
func DiffGems(prev, curr []SkillGroup) []GemChange {
	prevGems := gemSupportMap(prev)
	currGems := gemSupportMap(curr)

	changes := make([]GemChange, 0)

	for name, support := range currGems {
		if _, ok := prevGems[name]; !ok {
			changes = append(changes, GemChange{Kind: GemAdded, Name: name, IsSupport: support})
		}
	}
	for name, support := range prevGems {
		if _, ok := currGems[name]; !ok {
			changes = append(changes, GemChange{Kind: GemRemoved, Name: name, IsSupport: support})
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Kind != changes[j].Kind {
			return changes[i].Kind == GemAdded
		}
		if changes[i].IsSupport != changes[j].IsSupport {
			return !changes[i].IsSupport // active skills before supports
		}

		return changes[i].Name < changes[j].Name
	})

	return changes
}

// gemSupportMap maps each gem name to whether it is a support gem.
func gemSupportMap(groups []SkillGroup) map[string]bool {
	gems := map[string]bool{}
	for _, g := range groups {
		for _, gem := range g.Gems {
			gems[gem.Name] = gem.IsSupport
		}
	}

	return gems
}

func gemNameSet(groups []SkillGroup) map[string]struct{} {
	names := map[string]struct{}{}
	for _, g := range groups {
		for _, gem := range g.Gems {
			names[gem.Name] = struct{}{}
		}
	}

	return names
}
