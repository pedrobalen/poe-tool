package overlay

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/tree"
)

// navButton renders a previous/next control, dimmed and inert when disabled.
func navButton(th *theme.Theme, c *widget.Clickable, label string, enabled bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		btn := material.Button(th.Theme, c, label)
		if !enabled {
			btn.Background = th.Surface
			btn.Color = th.Muted
			gtx = gtx.Disabled()
		}

		return btn.Layout(gtx)
	}
}

// highlightFor builds the tree node-state lookups from a stage's precomputed
// diffs, so drawing never recomputes them.
func highlightFor(stage *builds.BuildStage) tree.StageHighlight {
	h := tree.StageHighlight{
		Current:   make(map[int]struct{}, len(stage.PassiveNodes)),
		Previous:  make(map[int]struct{}, len(stage.PassiveNodes)),
		New:       make(map[int]struct{}, len(stage.NewNodes)),
		Removed:   make(map[int]struct{}, len(stage.RemovedNodes)),
		Masteries: stage.MasterySelections,
	}
	for _, n := range stage.PassiveNodes {
		h.Current[n] = struct{}{}
	}
	for _, n := range stage.NewNodes {
		h.New[n] = struct{}{}
	}
	for _, n := range stage.RemovedNodes {
		h.Removed[n] = struct{}{}
	}

	// Previous = nodes allocated in the prior stage = carried-over nodes
	// (current minus new) plus the nodes removed this stage.
	for n := range h.Current {
		if _, isNew := h.New[n]; !isNew {
			h.Previous[n] = struct{}{}
		}
	}
	for n := range h.Removed {
		h.Previous[n] = struct{}{}
	}

	return h
}

func treeUnavailableText(err error) string {
	if err == nil {
		return "Passive tree data is not available for this build's version."
	}

	return "Passive tree unavailable: " + err.Error()
}

// buildGemRows lists the current stage's socket (link) groups and their gems
// exactly as saved in the build, with active skills before supports. Groups are
// separated by a divider; the build's main skill is highlighted. Gems carry no
// progression markers: stage-to-stage comparison lives on the passive tree only.
func buildGemRows(th *theme.Theme, stage *builds.BuildStage) []layout.Widget {
	rows := []layout.Widget{}
	first := true

	for _, group := range stage.SkillGroups {
		if len(group.Gems) == 0 {
			continue
		}
		if !first {
			rows = append(rows, dividerRow(th))
		}
		first = false

		for _, gem := range orderedGems(group.Gems) {
			primary := group.IsMain && !gem.IsSupport
			rows = append(rows, gemRow(th, gem.Name, gem.IsSupport, primary))
		}
	}

	if len(rows) == 0 {
		rows = append(rows, gemRow(th, "No skills.", false, false))
	}

	return rows
}

// orderedGems returns a skill group's gems with active skills before supports,
// preserving author order within each group.
func orderedGems(gems []builds.Gem) []builds.Gem {
	ordered := make([]builds.Gem, 0, len(gems))
	for _, g := range gems {
		if !g.IsSupport {
			ordered = append(ordered, g)
		}
	}
	for _, g := range gems {
		if g.IsSupport {
			ordered = append(ordered, g)
		}
	}

	return ordered
}

// gemRow renders one gem: active skills are bold with a diamond marker; supports
// are indented and dimmed with a link marker. The build's primary skill is drawn
// in the accent color.
func gemRow(th *theme.Theme, name string, isSupport, primary bool) layout.Widget {
	marker := "◆ "
	inset := layout.Inset{Top: unit.Dp(2)}
	col := th.Fg
	switch {
	case isSupport:
		marker = "↳ "
		inset.Left = unit.Dp(14)
		col = th.Muted
	case primary:
		col = th.Line
	}

	return func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body1(th.Theme, marker+name)
		lbl.Color = col
		lbl.MaxLines = 1
		if !isSupport {
			lbl.Font.Weight = 600
		}

		return inset.Layout(gtx, lbl.Layout)
	}
}

// dividerRow draws a thin horizontal separator between socket groups.
func dividerRow(th *theme.Theme) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(1)))
			line := th.Muted
			line.A = 0x40
			paint.FillShape(gtx.Ops, line, clip.Rect{Max: size}.Op())

			return layout.Dimensions{Size: size}
		})
	}
}
