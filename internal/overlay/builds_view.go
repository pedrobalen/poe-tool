package overlay

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/widgets"
)

// BuildsView lists the locally saved builds and lets the user activate, delete,
// or import a new one.
type BuildsView struct {
	list       widget.List
	back       widget.Clickable
	imp        widget.Clickable
	rows       []buildRow
	iconBack   *widget.Icon
	iconAdd    *widget.Icon
	iconDelete *widget.Icon
}

type buildRow struct {
	load widget.Clickable
	del  widget.Clickable
}

// NewBuildsView constructs the saved-builds view.
func NewBuildsView() *BuildsView {
	v := &BuildsView{
		iconBack:   mustIcon(icons.NavigationArrowBack),
		iconAdd:    mustIcon(icons.ContentAdd),
		iconDelete: mustIcon(icons.ActionDelete),
	}
	v.list.Axis = layout.Vertical

	return v
}

// BuildsAction reports what the user did on the saved-builds view this frame.
type BuildsAction struct {
	SelectID string
	DeleteID string
	Import   bool
	Back     bool
}

// Layout draws the saved-builds view and returns any action.
func (v *BuildsView) Layout(gtx layout.Context, th *theme.Theme, summaries []builds.Summary) BuildsAction {
	v.syncRows(len(summaries))

	action := BuildsAction{
		Import: v.imp.Clicked(gtx),
		Back:   v.back.Clicked(gtx),
	}
	v.collectRowActions(gtx, summaries, &action)

	layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(v.header(th)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Flexed(1, v.body(th, summaries)),
		)
	})

	return action
}

func (v *BuildsView) header(th *theme.Theme) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(smallIconButton(th, &v.back, v.iconBack, "Back")),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th.Theme, unit.Sp(16), "Saved builds")
				lbl.Color = th.Fg
				lbl.Font.Weight = 700

				return lbl.Layout(gtx)
			}),
			layout.Rigid(smallIconButton(th, &v.imp, v.iconAdd, "Import build")),
		)
	}
}

func (v *BuildsView) body(th *theme.Theme, summaries []builds.Summary) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if len(summaries) == 0 {
			return layout.Center.Layout(gtx, widgets.Body(th, "No saved builds yet. Import one to get started.", th.Muted))
		}

		return material.List(th.Theme, &v.list).Layout(gtx, len(summaries), func(gtx layout.Context, i int) layout.Dimensions {
			return v.row(th, summaries[i], &v.rows[i])(gtx)
		})
	}
}

func (v *BuildsView) row(th *theme.Theme, s builds.Summary, row *buildRow) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return row.load.Layout(gtx, v.rowLabel(th, s))
				}),
				layout.Rigid(smallIconButton(th, &row.del, v.iconDelete, "Delete build")),
			)
		})
	}
}

func (v *BuildsView) rowLabel(th *theme.Theme, s builds.Summary) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return widgets.FillBackground(gtx, th.Surface, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(widgets.Body(th, s.Name, th.Fg)),
							layout.Rigid(activeBadge(th, s.IsActive)),
						)
					}),
					layout.Rigid(widgets.Body(th, buildSubtitle(s), th.Muted)),
				)
			})
		})
	}
}

func (v *BuildsView) collectRowActions(gtx layout.Context, summaries []builds.Summary, action *BuildsAction) {
	for i := range summaries {
		if v.rows[i].del.Clicked(gtx) {
			action.DeleteID = summaries[i].ID
		}
		if v.rows[i].load.Clicked(gtx) {
			action.SelectID = summaries[i].ID
		}
	}
}

// syncRows keeps the per-row widget state sized to the current build count.
func (v *BuildsView) syncRows(n int) {
	if len(v.rows) == n {
		return
	}
	if n < len(v.rows) {
		v.rows = v.rows[:n]

		return
	}
	for len(v.rows) < n {
		v.rows = append(v.rows, buildRow{})
	}
}

func activeBadge(th *theme.Theme, active bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		if !active {
			return layout.Dimensions{}
		}

		return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, widgets.Body(th, "• active", th.New))
	}
}

func buildSubtitle(s builds.Summary) string {
	class := s.Ascendancy
	if class == "" {
		class = s.Class
	}
	if class == "" {
		class = "Unknown class"
	}

	return fmt.Sprintf("%s · %d stage(s) · %s", class, s.StageCount, s.SourceType)
}

func smallIconButton(
	th *theme.Theme,
	btn *widget.Clickable,
	icon *widget.Icon,
	desc string,
) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		b := material.IconButton(th.Theme, btn, icon, desc)
		b.Background = th.Surface
		b.Color = th.Fg
		b.Size = unit.Dp(18)
		b.Inset = layout.UniformInset(unit.Dp(8))

		return b.Layout(gtx)
	}
}
