package overlay

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/widgets"
)

// ImportView is the compact prompt for importing a build: a single input field
// and two buttons (paste a link/code, or pick a local Path of Building file).
type ImportView struct {
	editor  widget.Editor
	button  widget.Clickable
	cancel  widget.Clickable
	fromPob widget.Clickable
}

// ImportRequest is returned when the user acts on the import prompt.
type ImportRequest struct {
	Requested bool
	Input     string
	Cancelled bool
	FromPoB   bool
}

// Layout draws a small, centered import form. busy disables input while an
// import runs; errMsg shows the last failure; canCancel shows a way back when a
// build already exists.
func (v *ImportView) Layout(gtx layout.Context, th *theme.Theme, errMsg string, busy, canCancel bool) ImportRequest {
	v.editor.SingleLine = true

	req := v.readActions(gtx, busy, canCancel)

	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		width := gtx.Dp(unit.Dp(400))
		if width > gtx.Constraints.Max.X {
			width = gtx.Constraints.Max.X
		}
		gtx.Constraints.Min.X = width
		gtx.Constraints.Max.X = width

		return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if busy {
				return widgets.Body(th, "Importing…", th.Muted)(gtx)
			}

			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(widgets.SectionTitle(th, "IMPORT A BUILD")),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(v.field(th)),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(v.buttons(th, canCancel)),
				layout.Rigid(errorRow(th, errMsg)),
			)
		})
	})

	return req
}

func (v *ImportView) readActions(gtx layout.Context, busy, canCancel bool) ImportRequest {
	req := ImportRequest{}
	if canCancel && v.cancel.Clicked(gtx) {
		req.Cancelled = true
	}
	if !busy && v.fromPob.Clicked(gtx) {
		req.FromPoB = true
	}
	if !busy && v.button.Clicked(gtx) {
		if input := trimmed(v.editor.Text()); input != "" {
			req = ImportRequest{Requested: true, Input: input}
		}
	}

	return req
}

// field is the single-line input, drawn as a rounded surface so it reads as a
// text box.
func (v *ImportView) field(th *theme.Theme) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return cardBackground(gtx, th.Surface, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				ed := material.Editor(th.Theme, &v.editor, "pobb.in / Pastebin link or PoB code")
				ed.Color = th.Fg
				ed.HintColor = th.Muted

				return ed.Layout(gtx)
			})
		})
	}
}

func (v *ImportView) buttons(th *theme.Theme, canCancel bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(material.Button(th.Theme, &v.button, "Import").Layout),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Rigid(secondaryButton(th, &v.fromPob, "From PoB")),
			layout.Flexed(1, layout.Spacer{}.Layout),
			layout.Rigid(cancelIfPossible(th, &v.cancel, canCancel)),
		)
	}
}

func secondaryButton(th *theme.Theme, btn *widget.Clickable, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		b := material.Button(th.Theme, btn, label)
		b.Background = th.Surface
		b.Color = th.Fg

		return b.Layout(gtx)
	}
}

func cancelIfPossible(th *theme.Theme, btn *widget.Clickable, canCancel bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if !canCancel {
			return layout.Dimensions{}
		}
		b := material.Button(th.Theme, btn, "Cancel")
		b.Background = th.Surface
		b.Color = th.Muted

		return b.Layout(gtx)
	}
}

func errorRow(th *theme.Theme, errMsg string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if errMsg == "" {
			return layout.Dimensions{}
		}

		return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, widgets.Body(th, errMsg, th.Removed))
	}
}
