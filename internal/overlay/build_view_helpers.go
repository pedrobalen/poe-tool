package overlay

import (
	"fmt"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/tree"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/widgets"
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
// exactly as saved in the build, with active skills before supports. Gems carry
// no progression markers: stage-to-stage comparison is expressed on the passive
// tree only.
func buildGemRows(th *theme.Theme, stage *builds.BuildStage) []layout.Widget {
	rows := []layout.Widget{}

	for i, group := range stage.SkillGroups {
		if len(group.Gems) == 0 {
			continue
		}
		rows = append(rows, groupHeaderRow(th, group, i))
		for _, gem := range orderedGems(group.Gems) {
			rows = append(rows, gemRow(th, gem.Name, gem.IsSupport))
		}
		rows = append(rows, spacerRow(8))
	}

	if len(rows) == 0 {
		rows = append(rows, gemRow(th, "No skills.", false))
	}

	return rows
}

func groupHeaderRow(th *theme.Theme, group builds.SkillGroup, index int) layout.Widget {
	title := group.Label
	if title == "" {
		title = group.Slot
	}
	if title == "" {
		title = fmt.Sprintf("Group %d", index+1)
	}
	if group.IsMain {
		title += " · main"
	}

	return sectionRow(th, strings.ToUpper(title))
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
// are indented and dimmed with a link marker.
func gemRow(th *theme.Theme, name string, isSupport bool) layout.Widget {
	marker := "◆ "
	inset := layout.Inset{Top: unit.Dp(2)}
	col := th.Fg
	if isSupport {
		marker = "↳ "
		inset.Left = unit.Dp(14)
		col = th.Muted
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

func sectionRow(th *theme.Theme, title string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, widgets.SectionTitle(th, title))
	}
}

func spacerRow(height int) layout.Widget {
	return layout.Spacer{Height: unit.Dp(height)}.Layout
}
