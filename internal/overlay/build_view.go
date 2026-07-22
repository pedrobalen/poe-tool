package overlay

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	pt "github.com/pedrobalen/poe-build-overlay/internal/passive_tree"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/tree"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/widgets"
)

// NavKind identifies a navigation intent produced by the build view.
type NavKind int

const (
	// NavNone means no navigation was requested this frame.
	NavNone NavKind = iota
	// NavPrev requests the previous stage.
	NavPrev
	// NavNext requests the next stage.
	NavNext
	// NavFit requests recentering the tree on the current stage's new nodes.
	NavFit
)

// NavAction is the build view's per-frame output.
type NavAction struct {
	Kind NavKind
}

// BuildView renders a build's current stage: stage nav, the passive tree, and a
// side panel of skills/gems.
type BuildView struct {
	prev     widget.Clickable
	next     widget.Clickable
	fit      widget.Clickable
	gemsList widget.List
	tree     tree.Widget
	lastID   string // detects stage changes to refit the tree
}

// Layout draws the build view for the active stage and returns any navigation
// intent. treeData may be nil and treeErr non-nil when structural data for the
// build's version is unavailable; the view degrades gracefully in that case.
func (v *BuildView) Layout(
	gtx layout.Context,
	th *theme.Theme,
	b *builds.Build,
	treeData *pt.TreeData,
	treeErr error,
) NavAction {
	stage := b.SelectedStage()
	if stage == nil {
		return NavAction{}
	}

	v.refitOnStageChange(stage.ID)

	action := v.readActions(gtx, b)

	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(v.stageBar(th, b, stage)),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, v.treePanel(th, treeData, treeErr, stage)),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Rigid(v.gemsPanel(th, stage)),
			)
		}),
	)

	return action
}

func (v *BuildView) readActions(gtx layout.Context, b *builds.Build) NavAction {
	switch {
	case v.prev.Clicked(gtx) && b.HasPrev():
		return NavAction{Kind: NavPrev}
	case v.next.Clicked(gtx) && b.HasNext():
		return NavAction{Kind: NavNext}
	case v.fit.Clicked(gtx):
		v.tree.Fit()

		return NavAction{Kind: NavFit}
	default:
		return NavAction{}
	}
}

func (v *BuildView) refitOnStageChange(stageID string) {
	if stageID != v.lastID {
		v.tree.Fit()
		v.lastID = stageID
	}
}

func (v *BuildView) stageBar(th *theme.Theme, b *builds.Build, stage *builds.BuildStage) layout.Widget {
	position := fmt.Sprintf("%s  (%d/%d)", stage.Name, b.CurrentStage+1, len(b.Stages))

	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(navButton(th, &v.prev, "←", b.HasPrev())),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th.Theme, unit.Sp(15), position)
				lbl.Color = th.Fg
				lbl.Alignment = 1 // text.Middle

				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Rigid(navButton(th, &v.next, "→", b.HasNext())),
		)
	}
}

func (v *BuildView) treePanel(
	th *theme.Theme,
	data *pt.TreeData,
	treeErr error,
	stage *builds.BuildStage,
) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return widgets.FillBackground(gtx, th.Surface, func(gtx layout.Context) layout.Dimensions {
			if data == nil {
				return layout.Center.Layout(gtx, widgets.Body(th, treeUnavailableText(treeErr), th.Muted))
			}
			highlight := highlightFor(stage)

			return v.tree.Layout(gtx, th, data, highlight, stage.NewNodes)
		})
	}
}

// gemsPanel renders the skills/gems side panel: a fixed-width, scrollable column
// so its length never resizes the tree. It distinguishes active skill gems from
// support gems and lists the gem changes versus the previous stage.
func (v *BuildView) gemsPanel(th *theme.Theme, stage *builds.BuildStage) layout.Widget {
	rows := buildGemRows(th, stage)
	v.gemsList.Axis = layout.Vertical

	return func(gtx layout.Context) layout.Dimensions {
		width := gtx.Dp(unit.Dp(240))
		gtx.Constraints.Min.X = width
		gtx.Constraints.Max.X = width

		return widgets.FillBackground(gtx, th.Surface, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.List(th.Theme, &v.gemsList).Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
					return rows[i](gtx)
				})
			})
		})
	}
}
