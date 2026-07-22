package overlay

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/pedrobalen/poe-build-overlay/internal/pobfiles"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/widgets"
)

// PobFilesView lists the builds saved locally by the Path of Building desktop
// app so the user can import one directly.
type PobFilesView struct {
	list     widget.List
	back     widget.Clickable
	rows     []widget.Clickable
	iconBack *widget.Icon
}

// NewPobFilesView constructs the Path of Building builds view.
func NewPobFilesView() *PobFilesView {
	v := &PobFilesView{iconBack: mustIcon(icons.NavigationArrowBack)}
	v.list.Axis = layout.Vertical

	return v
}

// PobFilesAction reports what the user did on the PoB builds view this frame.
type PobFilesAction struct {
	SelectPath string
	Back       bool
}

// Layout draws the PoB builds list and returns any action. errText, when set,
// explains why no builds are shown (e.g. the folder was not found).
func (v *PobFilesView) Layout(gtx layout.Context, th *theme.Theme, files []pobfiles.Build, errText string) PobFilesAction {
	v.syncRows(len(files))

	action := PobFilesAction{Back: v.back.Clicked(gtx)}
	for i := range files {
		if v.rows[i].Clicked(gtx) {
			action.SelectPath = files[i].Path
		}
	}

	layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(v.header(th)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Flexed(1, v.body(th, files, errText)),
		)
	})

	return action
}

func (v *PobFilesView) header(th *theme.Theme) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(smallIconButton(th, &v.back, v.iconBack, "Back")),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th.Theme, unit.Sp(16), "Import from Path of Building")
				lbl.Color = th.Fg
				lbl.Font.Weight = 700

				return lbl.Layout(gtx)
			}),
		)
	}
}

func (v *PobFilesView) body(th *theme.Theme, files []pobfiles.Build, errText string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if errText != "" {
			return layout.Center.Layout(gtx, widgets.Body(th, errText, th.Muted))
		}
		if len(files) == 0 {
			return layout.Center.Layout(gtx, widgets.Body(th, "No Path of Building builds found.", th.Muted))
		}

		return material.List(th.Theme, &v.list).Layout(gtx, len(files), func(gtx layout.Context, i int) layout.Dimensions {
			return v.row(th, files[i].Name, &v.rows[i])(gtx)
		})
	}
}

func (v *PobFilesView) row(th *theme.Theme, name string, btn *widget.Clickable) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return widgets.FillBackground(gtx, th.Surface, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, widgets.Body(th, name, th.Fg))
				})
			})
		})
	}
}

func (v *PobFilesView) syncRows(n int) {
	if len(v.rows) == n {
		return
	}
	if n < len(v.rows) {
		v.rows = v.rows[:n]

		return
	}
	for len(v.rows) < n {
		v.rows = append(v.rows, widget.Clickable{})
	}
}
