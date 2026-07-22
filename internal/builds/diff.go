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
