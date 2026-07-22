// Package builds holds the persisted build domain and the services that import,
// diff, and navigate a build's progression stages. It depends on package pob for
// the parse/normalize pipeline but owns the shapes that reach storage and UI.
package builds

import (
	"time"

	"github.com/pedrobalen/poe-build-overlay/internal/pob"
)

// SourceType identifies where a build's export code originated.
type SourceType string

const (
	// SourcePobbin is a https://pobb.in/<id> link.
	SourcePobbin SourceType = "pobbin"
	// SourcePastebin is a pastebin link exported from PoB.
	SourcePastebin SourceType = "pastebin"
	// SourceDirect is a raw PoB code pasted directly.
	SourceDirect SourceType = "direct"
	// SourcePoBFile is a build read from a local Path of Building build file.
	SourcePoBFile SourceType = "pob"
)

// Build is the top-level persisted entity for an imported build.
type Build struct {
	ID           string
	Name         string
	Class        string
	Ascendancy   string
	TreeVersion  string
	SourceType   SourceType
	SourceURL    string
	SourceHash   string
	CurrentStage int // 0-based index into Stages
	ImportedAt   time.Time
	UpdatedAt    time.Time
	Stages       []BuildStage
}

// BuildStage is one progression step with its precomputed diff against the
// previous stage. Diffs are calculated once at import time so that opening the
// overlay never recomputes them.
type BuildStage struct {
	ID             string
	BuildID        string
	Name           string
	Order          int
	CharacterLevel *int
	PassiveNodes   []int
	NewNodes       []int
	RemovedNodes   []int
	SkillGroups    []SkillGroup
	Association    pob.StageAssociation
	// MasterySelections maps a mastery node id to the chosen effect id.
	MasterySelections map[int]int
	Notes             string
}

// SkillGroup is a linked group of gems within a stage.
type SkillGroup struct {
	ID      string
	StageID string
	Label   string
	Slot    string
	Enabled bool
	IsMain  bool
	Gems    []Gem
}

// Gem is a single gem within a skill group.
type Gem struct {
	Name          string
	Level         int
	RequiredLevel *int // populated from versioned local gem data, when available
	Quality       int
	Enabled       bool
	IsSupport     bool
}
