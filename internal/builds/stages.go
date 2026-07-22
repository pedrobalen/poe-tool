package builds

// SelectedStage returns the build's currently selected stage, or nil when the
// build has no stages.
func (b *Build) SelectedStage() *BuildStage {
	return b.StageAt(b.CurrentStage)
}

// StageAt returns the stage at index i, or nil when i is out of range.
func (b *Build) StageAt(i int) *BuildStage {
	if i < 0 || i >= len(b.Stages) {
		return nil
	}

	return &b.Stages[i]
}

// HasPrev reports whether a stage precedes the current one.
func (b *Build) HasPrev() bool {
	return b.CurrentStage > 0
}

// HasNext reports whether a stage follows the current one.
func (b *Build) HasNext() bool {
	return b.CurrentStage < len(b.Stages)-1
}

// ClampStage constrains i to a valid stage index. An empty build clamps to 0.
func (b *Build) ClampStage(i int) int {
	if len(b.Stages) == 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= len(b.Stages) {
		return len(b.Stages) - 1
	}

	return i
}
