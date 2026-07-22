// Package passive_tree holds the structural passive tree data (nodes,
// positions, connections) and the viewport/renderer used to draw it. Structural
// data is versioned by game tree version and loaded lazily, never during normal
// overlay opening.
package passive_tree

// NodeKind classifies a passive node for styling.
type NodeKind string

const (
	// KindNormal is a minor passive.
	KindNormal NodeKind = "normal"
	// KindNotable is a notable passive.
	KindNotable NodeKind = "notable"
	// KindKeystone is a keystone passive.
	KindKeystone NodeKind = "keystone"
	// KindMastery is a mastery node.
	KindMastery NodeKind = "mastery"
)

// Node is a single passive tree node with its layout position.
type Node struct {
	ID        int
	X         float32
	Y         float32
	Kind      NodeKind
	Name      string
	GroupID   int
	IsMastery bool
	// Effects maps a mastery node's effect ids to their stat text. Empty for
	// non-mastery nodes.
	Effects map[int]string
}

// Connection is an undirected edge between two nodes.
type Connection struct {
	From int
	To   int
}

// Bounds is the axis-aligned extent of a set of nodes.
type Bounds struct {
	MinX float32
	MinY float32
	MaxX float32
	MaxY float32
}

// TreeData is the full structural tree for one game version.
type TreeData struct {
	Version     string
	Nodes       map[int]Node
	Connections []Connection
	Bounds      Bounds
}

// BoundsOf computes the extent covering the given node ids that exist in the
// tree. Missing ids are skipped. The ok result is false when no id matched.
func (t *TreeData) BoundsOf(ids []int) (Bounds, bool) {
	var (
		b     Bounds
		found bool
	)

	for _, id := range ids {
		node, ok := t.Nodes[id]
		if !ok {
			continue
		}
		if !found {
			b = Bounds{MinX: node.X, MinY: node.Y, MaxX: node.X, MaxY: node.Y}
			found = true

			continue
		}
		b.MinX = minf(b.MinX, node.X)
		b.MinY = minf(b.MinY, node.Y)
		b.MaxX = maxf(b.MaxX, node.X)
		b.MaxY = maxf(b.MaxY, node.Y)
	}

	return b, found
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}

	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}

	return b
}
