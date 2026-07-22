package passive_tree

import "testing"

func sampleTree() *TreeData {
	return &TreeData{
		Version: "3_25",
		Nodes: map[int]Node{
			1: {ID: 1, X: 0, Y: 0},
			2: {ID: 2, X: 100, Y: 0},
			3: {ID: 3, X: 100, Y: 100},
		},
	}
}

func TestBoundsOf(t *testing.T) {
	tree := sampleTree()
	b, ok := tree.BoundsOf([]int{1, 2, 3, 999})
	if !ok {
		t.Fatal("expected bounds to be found")
	}
	if b.MinX != 0 || b.MinY != 0 || b.MaxX != 100 || b.MaxY != 100 {
		t.Fatalf("unexpected bounds: %+v", b)
	}
}

func TestFitToCentersContent(t *testing.T) {
	bounds := Bounds{MinX: 0, MinY: 0, MaxX: 100, MaxY: 100}
	cam := FitTo(bounds, 200, 200)

	// The world center (50,50) must project to the viewport center (100,100).
	sx, sy := cam.WorldToScreen(50, 50)
	if abs(sx-100) > 0.01 || abs(sy-100) > 0.01 {
		t.Fatalf("center projected to (%.2f,%.2f), want (100,100)", sx, sy)
	}
}

func TestZoomAtKeepsAnchorFixed(t *testing.T) {
	cam := Camera{Zoom: 1, OffsetX: 10, OffsetY: 20}
	before := worldUnder(cam, 150, 150)
	zoomed := cam.ZoomAt(150, 150, 1.5)
	after := worldUnder(zoomed, 150, 150)

	if abs(before[0]-after[0]) > 0.01 || abs(before[1]-after[1]) > 0.01 {
		t.Fatalf("anchor world point moved: %v -> %v", before, after)
	}
}

func TestVisibleNodesCulls(t *testing.T) {
	tree := sampleTree()
	// A camera that pushes everything far off-screen should cull all nodes.
	cam := Camera{Zoom: 1, OffsetX: 100000, OffsetY: 100000}
	if got := tree.VisibleNodes(cam, 200, 200, 10); len(got) != 0 {
		t.Fatalf("expected all nodes culled, got %d", len(got))
	}

	fit := FitTo(Bounds{MaxX: 100, MaxY: 100}, 200, 200)
	if got := tree.VisibleNodes(fit, 200, 200, 50); len(got) != 3 {
		t.Fatalf("expected 3 visible nodes, got %d", len(got))
	}
}

// worldUnder inverts the projection to find the world point under a screen pixel.
func worldUnder(c Camera, sx, sy float32) [2]float32 {
	return [2]float32{(sx - c.OffsetX) / c.Zoom, (sy - c.OffsetY) / c.Zoom}
}

func abs(f float32) float32 {
	if f < 0 {
		return -f
	}

	return f
}
