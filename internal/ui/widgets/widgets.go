// Package widgets holds small, reusable layout helpers shared across the
// overlay views, keeping view code focused on structure rather than primitives.
package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
)

// FillBackground paints c over the whole constraints area behind w.
func FillBackground(gtx layout.Context, c color.NRGBA, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()

	rect := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
	paint.Fill(gtx.Ops, c)
	rect.Pop()

	call.Add(gtx.Ops)

	return dims
}

// SectionTitle renders a muted uppercase-style section header.
func SectionTitle(th *theme.Theme, text string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th.Theme, unit.Sp(12), text)
		lbl.Color = th.Muted
		lbl.Font.Weight = 600

		return lbl.Layout(gtx)
	}
}

// Body renders standard body text in the given color.
func Body(th *theme.Theme, text string, c color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body1(th.Theme, text)
		lbl.Color = c

		return lbl.Layout(gtx)
	}
}

// Dot draws a small filled circle of the given color and diameter, used as a
// bullet or status marker.
func Dot(gtx layout.Context, c color.NRGBA, diameter int) layout.Dimensions {
	size := image.Pt(diameter, diameter)
	shape := clip.Ellipse{Max: size}.Op(gtx.Ops)
	paint.FillShape(gtx.Ops, c, shape)

	return layout.Dimensions{Size: size}
}
