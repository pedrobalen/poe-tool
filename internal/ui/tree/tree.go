// Package tree renders the passive tree with pan/zoom, drawing only what falls
// inside the viewport and highlighting node states for the current stage.
package tree

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	pt "github.com/pedrobalen/poe-build-overlay/internal/passive_tree"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
)

// NodeState classifies how a node should be drawn for the current stage.
type NodeState int

const (
	// StateInactive is a node not allocated in the current stage.
	StateInactive NodeState = iota
	// StatePrevious is a node carried over from the previous stage.
	StatePrevious
	// StateNew is a node allocated in the current stage.
	StateNew
	// StateRemoved is a node dropped in the current stage.
	StateRemoved
	// StateMasteryChanged is a mastery kept allocated but whose selected effect
	// changed this stage (swapped, not respecced).
	StateMasteryChanged
)

// StageHighlight holds the node-state lookups for one stage. The maps are built
// once from precomputed diffs, never recomputed while drawing.
//
//	Current  = nodes allocated in this stage (includes New)
//	Previous = nodes allocated in the prior stage (includes Removed)
//	New      = nodes added this stage
//	Removed  = nodes dropped this stage
type StageHighlight struct {
	Current  map[int]struct{}
	Previous map[int]struct{}
	New      map[int]struct{}
	Removed  map[int]struct{}
	// MasteryChanged holds masteries kept allocated but with a changed effect.
	MasteryChanged map[int]struct{}
	// Masteries maps a mastery node id to the effect id the build selected.
	Masteries map[int]int
}

func (h StageHighlight) stateOf(id int) NodeState {
	if _, ok := h.New[id]; ok {
		return StateNew
	}
	if _, ok := h.Removed[id]; ok {
		return StateRemoved
	}
	if _, ok := h.MasteryChanged[id]; ok {
		return StateMasteryChanged
	}
	if _, ok := h.Current[id]; ok {
		return StatePrevious
	}

	return StateInactive
}

func has(m map[int]struct{}, id int) bool {
	_, ok := m[id]

	return ok
}

// Widget is a stateful, pannable, zoomable tree view. The camera is exported so
// it can be persisted per build and restored on reopen.
type Widget struct {
	Camera   pt.Camera
	dragging bool
	last     f32.Point
	fitted   bool
	hoverPos f32.Point
	hovering bool
}

// Fit resets the camera to frame the given bounds within the last viewport size
// on the next layout. It is used to center on the new nodes of a stage.
func (w *Widget) Fit() {
	w.fitted = false
}

// Layout draws the tree and processes pan/zoom input within the constraints.
func (w *Widget) Layout(gtx layout.Context, th *theme.Theme, data *pt.TreeData, h StageHighlight, focus []int) layout.Dimensions {
	size := gtx.Constraints.Max
	viewW := float32(size.X)
	viewH := float32(size.Y)

	if !w.fitted && data != nil {
		if b, ok := data.BoundsOf(focus); ok {
			w.Camera = pt.FitTo(padBounds(b, 80), viewW, viewH)
		} else {
			w.Camera = pt.FitTo(data.Bounds, viewW, viewH)
		}
		w.fitted = true
	}

	area := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, w)
	w.processInput(gtx, viewW, viewH)

	if data != nil {
		w.drawConnections(gtx, th, data, h)
		w.drawNodes(gtx, th, data, h, viewW, viewH)
		w.drawTooltip(gtx, th, data, h)
	}
	area.Pop()

	return layout.Dimensions{Size: size}
}

func (w *Widget) processInput(gtx layout.Context, viewW, viewH float32) {
	filter := pointer.Filter{
		Target:  w,
		Kinds:   pointer.Scroll | pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Move | pointer.Leave,
		ScrollY: pointer.ScrollRange{Min: -1000, Max: 1000},
	}

	for {
		ev, ok := gtx.Event(filter)
		if !ok {
			break
		}
		pe, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		w.handlePointer(pe)
	}
}

func (w *Widget) handlePointer(pe pointer.Event) {
	switch pe.Kind {
	case pointer.Scroll:
		factor := float32(1) - pe.Scroll.Y*0.0015
		w.Camera = w.Camera.ZoomAt(pe.Position.X, pe.Position.Y, factor)
	case pointer.Press:
		w.dragging = true
		w.last = pe.Position
	case pointer.Drag:
		if w.dragging {
			w.Camera = w.Camera.Pan(pe.Position.X-w.last.X, pe.Position.Y-w.last.Y)
			w.last = pe.Position
		}
	case pointer.Release, pointer.Cancel:
		w.dragging = false
	case pointer.Move:
		w.hoverPos = pe.Position
		w.hovering = true
	case pointer.Leave:
		w.hovering = false
	}
}

// drawConnections renders edges in two passes so the highlighted progression
// paths always sit above the dim base mesh: first the inactive edges, then the
// coloured ones (carried gold, added green, removed red), mirroring Path of
// Building's compare view.
func (w *Widget) drawConnections(gtx layout.Context, th *theme.Theme, data *pt.TreeData, h StageHighlight) {
	w.drawConnectionPass(gtx, th, data, h, false)
	w.drawConnectionPass(gtx, th, data, h, true)
}

func (w *Widget) drawConnectionPass(
	gtx layout.Context,
	th *theme.Theme,
	data *pt.TreeData,
	h StageHighlight,
	highlighted bool,
) {
	for _, c := range data.Connections {
		from, okFrom := data.Nodes[c.From]
		to, okTo := data.Nodes[c.To]
		if !okFrom || !okTo {
			continue
		}

		col, width, isHighlight := edgeStyle(th, h, c.From, c.To)
		if isHighlight != highlighted {
			continue
		}

		var path clip.Path
		path.Begin(gtx.Ops)
		fx, fy := w.Camera.WorldToScreen(from.X, from.Y)
		tx, ty := w.Camera.WorldToScreen(to.X, to.Y)
		path.MoveTo(f32.Pt(fx, fy))
		path.LineTo(f32.Pt(tx, ty))
		shape := clip.Stroke{Path: path.End(), Width: width}.Op()
		paint.FillShape(gtx.Ops, col, shape)
	}
}

// edgeStyle classifies a connection for the current stage and returns its colour,
// stroke width, and whether it belongs to the highlighted (coloured) pass.
func edgeStyle(th *theme.Theme, h StageHighlight, a, b int) (color.NRGBA, float32, bool) {
	if has(h.Current, a) && has(h.Current, b) {
		if has(h.New, a) || has(h.New, b) {
			return th.New, 3, true
		}

		return th.Line, 2.5, true
	}

	if has(h.Previous, a) && has(h.Previous, b) && (has(h.Removed, a) || has(h.Removed, b)) {
		return th.Removed, 3, true
	}

	muted := th.Muted
	muted.A = 0x55

	return muted, 1, false
}

func (w *Widget) drawNodes(
	gtx layout.Context,
	th *theme.Theme,
	data *pt.TreeData,
	h StageHighlight,
	viewW, viewH float32,
) {
	const margin = 60
	for _, node := range data.VisibleNodes(w.Camera, viewW, viewH, margin) {
		sx, sy := w.Camera.WorldToScreen(node.X, node.Y)
		state := h.stateOf(node.ID)
		radius := nodeRadius(node, w.Camera.Zoom)
		if state == StateNew || state == StateRemoved || state == StateMasteryChanged {
			radius += 2
		}
		col := colorFor(th, state)

		center := image.Pt(int(sx), int(sy))
		rect := image.Rectangle{
			Min: center.Sub(image.Pt(radius, radius)),
			Max: center.Add(image.Pt(radius, radius)),
		}
		shape := clip.Ellipse(rect).Op(gtx.Ops)
		paint.FillShape(gtx.Ops, col, shape)
	}
}

// drawTooltip shows the node under the cursor while hovering (and not panning).
// For a mastery node it appends the effect the build selected, mirroring Path of
// Building's mastery tooltip.
func (w *Widget) drawTooltip(gtx layout.Context, th *theme.Theme, data *pt.TreeData, h StageHighlight) {
	if !w.hovering || w.dragging {
		return
	}
	node, ok := w.nodeAt(data)
	if !ok || node.Name == "" {
		return
	}

	w.paintTooltip(gtx, th, tooltipText(node, h))
}

// tooltipText builds the hover text for a node: its name, plus the selected
// mastery effect when applicable.
func tooltipText(node pt.Node, h StageHighlight) string {
	if !node.IsMastery {
		return node.Name
	}

	effectID, ok := h.Masteries[node.ID]
	if !ok {
		return node.Name + "\n(no effect selected)"
	}
	text, ok := node.Effects[effectID]
	if !ok || text == "" {
		return node.Name
	}

	return node.Name + "\n" + text
}

// nodeAt returns the node closest to the cursor within its clickable radius.
func (w *Widget) nodeAt(data *pt.TreeData) (pt.Node, bool) {
	var (
		found pt.Node
		best  float32
		ok    bool
	)

	for _, node := range data.Nodes {
		sx, sy := w.Camera.WorldToScreen(node.X, node.Y)
		dx := sx - w.hoverPos.X
		dy := sy - w.hoverPos.Y
		dist := dx*dx + dy*dy
		reach := float32(nodeRadius(node, w.Camera.Zoom)) + 6
		if dist > reach*reach {
			continue
		}
		if !ok || dist < best {
			found, best, ok = node, dist, true
		}
	}

	return found, ok
}

func (w *Widget) paintTooltip(gtx layout.Context, th *theme.Theme, text string) {
	gtx2 := gtx
	gtx2.Constraints.Min = image.Point{}
	gtx2.Constraints.Max = image.Pt(gtx.Dp(unit.Dp(240)), gtx.Dp(unit.Dp(120)))

	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(6)).Layout(gtx2, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th.Theme, text)
		lbl.Color = th.Fg
		lbl.MaxLines = 6

		return lbl.Layout(gtx)
	})
	call := macro.Stop()

	pos := image.Pt(int(w.hoverPos.X)+14, int(w.hoverPos.Y)+14)
	pos = clampTooltip(pos, dims.Size, gtx.Constraints.Max)

	off := op.Offset(pos).Push(gtx.Ops)
	bg := clip.RRect{Rect: image.Rectangle{Max: dims.Size}, SE: 4, SW: 4, NW: 4, NE: 4}.Push(gtx.Ops)
	paint.Fill(gtx.Ops, color.NRGBA{R: 0x0A, G: 0x0C, B: 0x12, A: 0xF2})
	bg.Pop()
	call.Add(gtx.Ops)
	off.Pop()
}

// clampTooltip keeps the tooltip box fully inside the viewport.
func clampTooltip(pos, size, bounds image.Point) image.Point {
	if pos.X+size.X > bounds.X {
		pos.X = bounds.X - size.X
	}
	if pos.Y+size.Y > bounds.Y {
		pos.Y = bounds.Y - size.Y
	}
	if pos.X < 0 {
		pos.X = 0
	}
	if pos.Y < 0 {
		pos.Y = 0
	}

	return pos
}

func colorFor(th *theme.Theme, state NodeState) color.NRGBA {
	switch state {
	case StateNew:
		return th.New
	case StateRemoved:
		return th.Removed
	case StateMasteryChanged:
		return th.Future
	case StatePrevious:
		return th.Line
	default:
		muted := th.Muted
		muted.A = 0x99

		return muted
	}
}

func nodeRadius(node pt.Node, zoom float32) int {
	base := float32(4)
	switch node.Kind {
	case pt.KindKeystone:
		base = 9
	case pt.KindNotable:
		base = 7
	case pt.KindMastery:
		base = 6
	}

	r := int(base * zoom)
	if r < 2 {
		r = 2
	}
	if r > 24 {
		r = 24
	}

	return r
}

func padBounds(b pt.Bounds, pad float32) pt.Bounds {
	return pt.Bounds{
		MinX: b.MinX - pad,
		MinY: b.MinY - pad,
		MaxX: b.MaxX + pad,
		MaxY: b.MaxY + pad,
	}
}
