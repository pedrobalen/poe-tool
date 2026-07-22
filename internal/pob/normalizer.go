package pob

import (
	"errors"
	"fmt"
	"strings"
)

// Normalize converts a ParsedBuild into a NormalizedBuild: one stage per saved
// tree, in the author's original order, each associated with the most plausible
// skill set. It never invents progression — a single-tree build yields a single
// stage.
func Normalize(parsed ParsedBuild) (NormalizedBuild, error) {
	if len(parsed.Specs) == 0 {
		return NormalizedBuild{}, errors.New("pob: build has no passive tree to normalize")
	}

	build := NormalizedBuild{
		Name:        buildName(parsed),
		Class:       parsed.ClassName,
		Ascendancy:  parsed.Ascendancy,
		TreeVersion: parsed.TreeVersion,
		ActiveStage: activeStageIndex(parsed),
		Stages:      make([]NormalizedStage, 0, len(parsed.Specs)),
	}

	defaultSet := defaultSkillSet(parsed)

	for i, spec := range parsed.Specs {
		set, assoc := associateSkillSet(spec, i, parsed.SkillSets, defaultSet)
		build.Stages = append(build.Stages, NormalizedStage{
			Name:              stageName(spec, i),
			Order:             i,
			Nodes:             spec.Nodes,
			SkillGroups:       normalizeGroups(set),
			Association:       assoc,
			MasterySelections: spec.MasterySelections,
		})
	}

	return build, nil
}

// associateSkillSet applies the plan's association strategy in priority order:
// name equivalence, then corresponding position, then the build's default set,
// then none.
func associateSkillSet(
	spec ParsedSpec,
	index int,
	sets []ParsedSkillSet,
	def *ParsedSkillSet,
) (*ParsedSkillSet, StageAssociation) {
	if len(sets) == 0 {
		return nil, AssocNone
	}

	if match := matchByName(spec.Title, sets); match != nil {
		return match, AssocByName
	}

	if index < len(sets) {
		return &sets[index], AssocByPosition
	}

	if def != nil {
		return def, AssocDefault
	}

	return nil, AssocNone
}

func matchByName(title string, sets []ParsedSkillSet) *ParsedSkillSet {
	title = normalizeTitle(title)
	if title == "" {
		return nil
	}

	for i := range sets {
		if normalizeTitle(sets[i].Title) == title {
			return &sets[i]
		}
	}

	return nil
}

func defaultSkillSet(parsed ParsedBuild) *ParsedSkillSet {
	if len(parsed.SkillSets) == 0 {
		return nil
	}

	idx := parsed.ActiveSkillSet - 1
	if idx < 0 || idx >= len(parsed.SkillSets) {
		idx = 0
	}

	return &parsed.SkillSets[idx]
}

func normalizeGroups(set *ParsedSkillSet) []NormalizedSkillGroup {
	if set == nil {
		return []NormalizedSkillGroup{}
	}

	groups := make([]NormalizedSkillGroup, 0, len(set.Groups))
	for _, g := range set.Groups {
		groups = append(groups, NormalizedSkillGroup{
			Label:   g.Label,
			Slot:    g.Slot,
			Enabled: g.Enabled,
			IsMain:  g.IsMain,
			Gems:    normalizeGems(g.Gems),
		})
	}

	return groups
}

func normalizeGems(gems []ParsedGem) []NormalizedGem {
	out := make([]NormalizedGem, 0, len(gems))
	for _, g := range gems {
		out = append(out, NormalizedGem{
			Name:      g.Name,
			Level:     g.Level,
			Quality:   g.Quality,
			Enabled:   g.Enabled,
			IsSupport: g.IsSupport,
		})
	}

	return out
}

func activeStageIndex(parsed ParsedBuild) int {
	idx := parsed.ActiveSpec - 1
	if idx < 0 || idx >= len(parsed.Specs) {
		return 0
	}

	return idx
}

// buildName derives a human label for the build from its class and ascendancy,
// since the PoB export has no dedicated build-name field.
func buildName(parsed ParsedBuild) string {
	class := parsed.Ascendancy
	if class == "" || strings.EqualFold(class, "None") {
		class = parsed.ClassName
	}
	if class == "" {
		return "Imported Build"
	}

	return fmt.Sprintf("%s Build", class)
}

// stageName preserves the author's spec title, falling back to a positional
// label only when the spec is unnamed.
func stageName(spec ParsedSpec, index int) string {
	if title := strings.TrimSpace(spec.Title); title != "" {
		return title
	}

	return fmt.Sprintf("Tree %d", index+1)
}

func normalizeTitle(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
