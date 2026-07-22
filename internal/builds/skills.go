package builds

// MainSkillGroup returns the stage's active (main) skill group, or nil when
// none is flagged. When several are flagged, the first in author order wins.
func (s *BuildStage) MainSkillGroup() *SkillGroup {
	for i := range s.SkillGroups {
		if s.SkillGroups[i].IsMain {
			return &s.SkillGroups[i]
		}
	}

	return nil
}

// NextGemChangeStage returns the order of the next stage after `from` whose gem
// set differs from stage `from`, and true when such a stage exists. It powers
// the "Próxima mudança" hint without recomputing diffs at display time.
func (b *Build) NextGemChangeStage(from int) (int, bool) {
	base := b.StageAt(from)
	if base == nil {
		return 0, false
	}

	baseNames := gemNameSet(base.SkillGroups)
	for i := from + 1; i < len(b.Stages); i++ {
		if !sameGemSet(baseNames, gemNameSet(b.Stages[i].SkillGroups)) {
			return i, true
		}
	}

	return 0, false
}

func sameGemSet(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for name := range a {
		if _, ok := b[name]; !ok {
			return false
		}
	}

	return true
}
