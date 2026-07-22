package passive_tree

// Camera maps world tree coordinates to screen pixels via a uniform zoom and a
// screen-space offset. It is deliberately plain data so it can be persisted and
// restored per build without touching the renderer.
type Camera struct {
	OffsetX float32
	OffsetY float32
	Zoom    float32
}

const (
	minZoom = 0.05
	maxZoom = 3.0
	// fitPadding keeps fitted content off the very edges of the viewport.
	fitPadding = 0.9
)

// WorldToScreen projects a world point to screen space.
func (c Camera) WorldToScreen(x, y float32) (float32, float32) {
	return x*c.Zoom + c.OffsetX, y*c.Zoom + c.OffsetY
}

// ClampZoom keeps a zoom factor within the supported range.
func ClampZoom(zoom float32) float32 {
	switch {
	case zoom < minZoom:
		return minZoom
	case zoom > maxZoom:
		return maxZoom
	default:
		return zoom
	}
}

// FitTo returns a camera that centers bounds within a viewW x viewH viewport.
// A zero-area viewport or bounds yields a neutral, centered camera.
func FitTo(bounds Bounds, viewW, viewH float32) Camera {
	width := bounds.MaxX - bounds.MinX
	height := bounds.MaxY - bounds.MinY

	if viewW <= 0 || viewH <= 0 {
		return Camera{Zoom: 1}
	}

	zoom := float32(1)
	if width > 0 && height > 0 {
		zoom = ClampZoom(minf(viewW/width, viewH/height) * fitPadding)
	}

	centerX := (bounds.MinX + bounds.MaxX) / 2
	centerY := (bounds.MinY + bounds.MaxY) / 2

	return Camera{
		Zoom:    zoom,
		OffsetX: viewW/2 - centerX*zoom,
		OffsetY: viewH/2 - centerY*zoom,
	}
}

// ZoomAt scales the camera around a screen-space anchor (e.g. the cursor) so the
// point under the anchor stays fixed. factor > 1 zooms in.
func (c Camera) ZoomAt(anchorX, anchorY, factor float32) Camera {
	newZoom := ClampZoom(c.Zoom * factor)
	// Solve for the offset that keeps (anchor) mapping to the same world point.
	scale := newZoom / c.Zoom

	return Camera{
		Zoom:    newZoom,
		OffsetX: anchorX - (anchorX-c.OffsetX)*scale,
		OffsetY: anchorY - (anchorY-c.OffsetY)*scale,
	}
}

// Pan translates the camera by a screen-space delta.
func (c Camera) Pan(dx, dy float32) Camera {
	c.OffsetX += dx
	c.OffsetY += dy

	return c
}

// VisibleNodes returns the nodes whose projected position falls within the
// viewport (expanded by margin), so the renderer only draws what is on screen.
func (t *TreeData) VisibleNodes(cam Camera, viewW, viewH, margin float32) []Node {
	visible := make([]Node, 0, len(t.Nodes))
	for _, node := range t.Nodes {
		sx, sy := cam.WorldToScreen(node.X, node.Y)
		if sx < -margin || sy < -margin || sx > viewW+margin || sy > viewH+margin {
			continue
		}
		visible = append(visible, node)
	}

	return visible
}
