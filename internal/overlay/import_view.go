package overlay

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/widgets"
)

// ImportView is the first-run prompt: paste a pobb.in / Pastebin link or a raw
// PoB code and import it.
type ImportView struct {
	editor  widget.Editor
	button  widget.Clickable
	cancel  widget.Clickable
	fromPob widget.Clickable
}

// ImportRequest is returned when the user asks to import the current input.
type ImportRequest struct {
	Requested bool
	Input     string
	Cancelled bool
	FromPoB   bool
}

// Layout draws the import prompt and reports an import request. busy disables
// the control while an import is in flight; errMsg shows the last failure.
// canCancel shows a cancel affordance when a build already exists to return to.
func (v *ImportView) Layout(gtx layout.Context, th *theme.Theme, errMsg string, busy, canCancel bool) ImportRequest {
	v.editor.SingleLine = true

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

	inset := layout.UniformInset(unit.Dp(24))

	inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(widgets.SectionTitle(th, "IMPORT BUILD")),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				ed := material.Editor(th.Theme, &v.editor, "Paste a pobb.in / Pastebin link or PoB code")
				ed.Color = th.Fg
				ed.HintColor = th.Muted

				return ed.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(v.statusOrButton(th, errMsg, busy, canCancel)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(widgets.Body(th, "or", th.Muted)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(v.pobButton(th, busy)),
		)
	})

	return req
}

// pobButton offers importing directly from a locally saved Path of Building
// build file.
func (v *ImportView) pobButton(th *theme.Theme, busy bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if busy {
			return layout.Dimensions{}
		}
		btn := material.Button(th.Theme, &v.fromPob, "Import from Path of Building")
		btn.Background = th.Surface
		btn.Color = th.Fg

		return btn.Layout(gtx)
	}
}

func (v *ImportView) statusOrButton(th *theme.Theme, errMsg string, busy, canCancel bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if busy {
			return widgets.Body(th, "Importing…", th.Muted)(gtx)
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if errMsg == "" {
					return layout.Dimensions{}
				}

				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, widgets.Body(th, errMsg, th.Removed))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(material.Button(th.Theme, &v.button, "Import").Layout),
					layout.Rigid(v.cancelButton(th, canCancel)),
				)
			}),
		)
	}
}

func (v *ImportView) cancelButton(th *theme.Theme, canCancel bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if !canCancel {
			return layout.Dimensions{}
		}

		return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th.Theme, &v.cancel, "Cancel")
			btn.Background = th.Surface
			btn.Color = th.Fg

			return btn.Layout(gtx)
		})
	}
}
