package overlay

import (
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

// gemStatus marks whether a gem was added, removed, or carried over this stage.
type gemStatus int

const (
	gemKept gemStatus = iota
	gemAdded
	gemRemoved
)

// buildGemRows renders each socket (link) group with its gems, marking additions
// and removals in place. Changes are shown within the gem's own group so a
// support is never displayed against a skill it is not linked to.
func buildGemRows(th *theme.Theme, b *builds.Build, stage *builds.BuildStage) []layout.Widget {
	prev := b.StageAt(stage.Order - 1)
	rows := []layout.Widget{}

	for i, group := range stage.SkillGroups {
		if len(group.Gems) == 0 {
			continue
		}
		prevGroup := matchPrevGroup(prev, group, i)
		rows = append(rows, groupHeaderRow(th, group))
		rows = append(rows, groupGemRows(th, group, prevGroup, prev != nil)...)
		rows = append(rows, spacerRow(8))
	}

	if len(rows) == 0 {
		rows = append(rows, gemRow(th, "No skills.", false, gemKept))
	}

	return rows
}

// groupGemRows renders a group's current gems (marking new ones) followed by any
// gems dropped from the matching previous group.
func groupGemRows(
	th *theme.Theme,
	group builds.SkillGroup,
	prevGroup *builds.SkillGroup,
	hasPrev bool,
) []layout.Widget {
	prevNames := gemNames(prevGroup)
	rows := []layout.Widget{}

	for _, gem := range orderedGems(group.Gems) {
		status := gemKept
		if hasPrev {
			if _, ok := prevNames[gem.Name]; !ok {
				status = gemAdded
			}
		}
		rows = append(rows, gemRow(th, gem.Name, gem.IsSupport, status))
	}

	if prevGroup != nil {
		currNames := gemNames(&group)
		for _, gem := range orderedGems(prevGroup.Gems) {
			if _, ok := currNames[gem.Name]; !ok {
				rows = append(rows, gemRow(th, gem.Name, gem.IsSupport, gemRemoved))
			}
		}
	}

	return rows
}

// matchPrevGroup finds the previous stage's socket group corresponding to g,
// preferring slot+label, then slot, then position.
func matchPrevGroup(prev *builds.BuildStage, g builds.SkillGroup, index int) *builds.SkillGroup {
	if prev == nil {
		return nil
	}
	for i := range prev.SkillGroups {
		p := &prev.SkillGroups[i]
		if (g.Slot != "" || g.Label != "") && p.Slot == g.Slot && p.Label == g.Label {
			return p
		}
	}
	for i := range prev.SkillGroups {
		if g.Slot != "" && prev.SkillGroups[i].Slot == g.Slot {
			return &prev.SkillGroups[i]
		}
	}
	if index < len(prev.SkillGroups) {
		return &prev.SkillGroups[index]
	}

	return nil
}

func gemNames(group *builds.SkillGroup) map[string]struct{} {
	names := map[string]struct{}{}
	if group == nil {
		return names
	}
	for _, gem := range group.Gems {
		names[gem.Name] = struct{}{}
	}

	return names
}

func groupHeaderRow(th *theme.Theme, group builds.SkillGroup) layout.Widget {
	title := group.Slot
	if title == "" {
		title = group.Label
	}
	if title == "" {
		title = "Skill group"
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

// gemRow renders one gem with its status. Active skills are bold with a diamond
// marker; supports are indented with a link marker. Added gems are green,
// removed gems red, carried-over gems neutral.
func gemRow(th *theme.Theme, name string, isSupport bool, status gemStatus) layout.Widget {
	marker := "◆ "
	inset := layout.Inset{Top: unit.Dp(2)}
	if isSupport {
		marker = "↳ "
		inset.Left = unit.Dp(14)
	}

	sign := ""
	col := th.Fg
	if isSupport {
		col = th.Muted
	}
	switch status {
	case gemAdded:
		sign = "+ "
		col = th.New
	case gemRemoved:
		sign = "− "
		col = th.Removed
	}

	return func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body1(th.Theme, sign+marker+name)
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
