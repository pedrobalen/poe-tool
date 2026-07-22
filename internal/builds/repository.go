package builds

import "context"

// Summary is a lightweight view of a build for lists and pickers, avoiding the
// cost of loading every stage.
type Summary struct {
	ID         string
	Name       string
	Class      string
	Ascendancy string
	SourceType SourceType
	StageCount int
	IsActive   bool
}

// BuildRepository persists and retrieves builds. Implementations live in the
// storage layer; the interface lives here so the domain owns its contract.
type BuildRepository interface {
	Save(ctx context.Context, build Build) error
	FindByID(ctx context.Context, id string) (Build, error)
	FindActive(ctx context.Context) (Build, error)
	List(ctx context.Context) ([]Summary, error)
	SetActive(ctx context.Context, id string) error
	SetCurrentStage(ctx context.Context, buildID string, stageOrder int) error
	Delete(ctx context.Context, id string) error
	// FindIDBySourceHash returns the id of a build already stored with hash,
	// enabling import and update to skip redundant reprocessing.
	FindIDBySourceHash(ctx context.Context, hash string) (string, bool, error)
}

// RequiredLevelProvider supplies a gem's minimum required level from versioned
// local game data. It is optional: when absent, gems carry no required level
// rather than an inferred one.
type RequiredLevelProvider interface {
	RequiredLevel(gemName, treeVersion string) (int, bool)
}
