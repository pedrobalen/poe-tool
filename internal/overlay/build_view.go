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
	Kind          NavKind
	ToggleCompare bool // the compare checkbox changed this frame
	CompareOn     bool // the compare checkbox's new value
}

// BuildView renders a build's current stage: stage nav, the passive tree, and a
// side panel of skills/gems.
type BuildView struct {
	prev       widget.Clickable
	next       widget.Clickable
	fit        widget.Clickable
	compareBox widget.Bool
	gemsList   widget.List
	tree       tree.Widget
	lastID     string // detects stage changes to refit the tree
}

// Layout draws the build view for the active stage and returns any navigation
// intent. treeData may be nil and treeErr non-nil when structural data for the
// build's version is unavailable; the view degrades gracefully in that case.
// compare controls whether the tree highlights the diff against the previous
// stage (green/red) or just shows the current allocation.
func (v *BuildView) Layout(
	gtx layout.Context,
	th *theme.Theme,
	b *builds.Build,
	treeData *pt.TreeData,
	treeErr error,
	compare bool,
) NavAction {
	stage := b.SelectedStage()
	if stage == nil {
		return NavAction{}
	}

	v.refitOnStageChange(stage.ID)

	action := v.readActions(gtx, b)
	v.readCompare(gtx, compare, &action)

	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(v.stageBar(th, b, stage)),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, v.treePanel(th, treeData, treeErr, stage, compare)),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Rigid(v.gemsPanel(th, stage)),
			)
		}),
	)

	return action
}

// readCompare syncs the checkbox to the current state and reports a toggle.
func (v *BuildView) readCompare(gtx layout.Context, compare bool, action *NavAction) {
	v.compareBox.Value = compare
	if v.compareBox.Update(gtx) {
		action.ToggleCompare = true
		action.CompareOn = v.compareBox.Value
	}
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
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				cb := material.CheckBox(th.Theme, &v.compareBox, "Compare")
				cb.Color = th.Muted
				cb.IconColor = th.New
				cb.TextSize = unit.Sp(13)

				return cb.Layout(gtx)
			}),
		)
	}
}

func (v *BuildView) treePanel(
	th *theme.Theme,
	data *pt.TreeData,
	treeErr error,
	stage *builds.BuildStage,
	compare bool,
) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return widgets.FillBackground(gtx, th.Surface, func(gtx layout.Context) layout.Dimensions {
			if data == nil {
				return layout.Center.Layout(gtx, widgets.Body(th, treeUnavailableText(treeErr), th.Muted))
			}
			highlight := highlightFor(stage, compare)
			focus := stage.NewNodes
			if !compare {
				focus = stage.PassiveNodes
			}

			return v.tree.Layout(gtx, th, data, highlight, focus)
		})
	}
}

// gemsPanel renders the skills/gems side panel: a fixed-width, scrollable column
// so its length never resizes the tree. Each socket (link) group is drawn as its
// own card, so it is unambiguous that a support inside a card applies to the
// active skill(s) in that same linked group.
func (v *BuildView) gemsPanel(th *theme.Theme, stage *builds.BuildStage) layout.Widget {
	groups := activeGroups(stage.SkillGroups)
	v.gemsList.Axis = layout.Vertical

	return func(gtx layout.Context) layout.Dimensions {
		width := gtx.Dp(unit.Dp(240))
		gtx.Constraints.Min.X = width
		gtx.Constraints.Max.X = width

		if len(groups) == 0 {
			return widgets.FillBackground(gtx, th.Surface, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, widgets.Body(th, "No skills.", th.Muted))
			})
		}

		return material.List(th.Theme, &v.gemsList).Layout(gtx, len(groups), func(gtx layout.Context, i int) layout.Dimensions {
			return groupCard(th, groups[i])(gtx)
		})
	}
}
