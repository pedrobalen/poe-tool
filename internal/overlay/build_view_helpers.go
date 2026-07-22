package overlay

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
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
// diffs, so drawing never recomputes them. When compare is false, only the
// current allocation is populated, so the tree shows the stage's tree without
// the green/red progression coloring (useful for builds whose saved trees are
// unrelated variants rather than A→B steps).
func highlightFor(b *builds.Build, stage *builds.BuildStage, compare bool) tree.StageHighlight {
	h := tree.StageHighlight{
		Current:        make(map[int]struct{}, len(stage.PassiveNodes)),
		Previous:       map[int]struct{}{},
		New:            map[int]struct{}{},
		Removed:        map[int]struct{}{},
		MasteryChanged: map[int]struct{}{},
		Masteries:      stage.MasterySelections,
	}
	for _, n := range stage.PassiveNodes {
		h.Current[n] = struct{}{}
	}

	if !compare {
		return h
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

	markChangedMasteries(b, stage, h.MasteryChanged)

	return h
}

// markChangedMasteries flags mastery nodes that stay allocated from the previous
// stage but whose selected effect changed (a swap, not a respec).
func markChangedMasteries(b *builds.Build, stage *builds.BuildStage, changed map[int]struct{}) {
	prev := b.StageAt(stage.Order - 1)
	if prev == nil {
		return
	}
	for node, effect := range stage.MasterySelections {
		if prevEffect, ok := prev.MasterySelections[node]; ok && prevEffect != effect {
			changed[node] = struct{}{}
		}
	}
}

func treeUnavailableText(err error) string {
	if err == nil {
		return "Passive tree data is not available for this build's version."
	}

	return "Passive tree unavailable: " + err.Error()
}

// activeGroups returns the stage's socket groups that actually contain gems.
func activeGroups(groups []builds.SkillGroup) []builds.SkillGroup {
	out := make([]builds.SkillGroup, 0, len(groups))
	for _, g := range groups {
		if len(g.Gems) > 0 {
			out = append(out, g)
		}
	}

	return out
}

// groupCard renders one socket (link) group as a self-contained card: its active
// skill(s) first, then the support gems. Because every gem in the card belongs
// to the same linked group, the supports it contains apply to the skill(s) shown
// in the same card — never to gems in another card.
func groupCard(th *theme.Theme, group builds.SkillGroup) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return cardBackground(gtx, th.Surface, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, 0, len(group.Gems))
					for _, gem := range orderedGems(group.Gems) {
						primary := group.IsMain && !gem.IsSupport
						children = append(children, layout.Rigid(gemRow(th, gem.Name, gem.IsSupport, primary)))
					}

					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
				})
			})
		})
	}
}

// orderedGems returns a skill group's gems with active skills before supports,
// preserving author order within each category.
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

// cardBackground paints a rounded surface behind w, used to bound a socket group.
func cardBackground(gtx layout.Context, c color.NRGBA, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()

	rr := clip.RRect{Rect: image.Rectangle{Max: dims.Size}, SE: 6, SW: 6, NW: 6, NE: 6}.Push(gtx.Ops)
	paint.Fill(gtx.Ops, c)
	rr.Pop()
	call.Add(gtx.Ops)

	return dims
}
