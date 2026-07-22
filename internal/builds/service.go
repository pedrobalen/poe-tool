package builds

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pedrobalen/poe-build-overlay/internal/id"
	"github.com/pedrobalen/poe-build-overlay/internal/importers"
	"github.com/pedrobalen/poe-build-overlay/internal/pob"
)

// ErrNoActiveBuild is returned when no build is marked active.
var ErrNoActiveBuild = errors.New("builds: no active build")

// Service orchestrates the import pipeline (fetch -> decode -> parse ->
// normalize -> diff -> persist) and build management. All heavy work happens
// here, off the UI thread, so opening the overlay only reads stored data.
type Service struct {
	registry *importers.Registry
	repo     BuildRepository
	levels   RequiredLevelProvider // optional; nil means no required-level data
	now      func() time.Time
}

// NewService wires the service. levels may be nil.
func NewService(
	registry *importers.Registry,
	repo BuildRepository,
	levels RequiredLevelProvider,
) *Service {
	return &Service{
		registry: registry,
		repo:     repo,
		levels:   levels,
		now:      time.Now,
	}
}

// Import resolves the input (link or code), processes it into a build, persists
// it, and marks it active. When an identical source was already imported, the
// existing build is returned unchanged.
func (s *Service) Import(ctx context.Context, input string) (Build, error) {
	res, err := s.registry.Import(ctx, input)
	if err != nil {
		return Build{}, err
	}

	hash := hashCode(res.Code)
	if build, done, err := s.reuseExisting(ctx, hash); done {
		return build, err
	}

	xml, err := pob.Decode(res.Code)
	if err != nil {
		return Build{}, err
	}

	build, err := s.processXML(xml, SourceType(res.Source), res.URL, hash)
	if err != nil {
		return Build{}, err
	}

	return s.saveActive(ctx, build)
}

// ImportFile imports a build directly from a Path of Building build file, whose
// contents are already plain XML (no Base64/zlib decoding needed).
func (s *Service) ImportFile(ctx context.Context, path string) (Build, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Build{}, fmt.Errorf("builds: reading PoB file: %w", err)
	}

	hash := hashCode(string(data))
	if build, done, err := s.reuseExisting(ctx, hash); done {
		return build, err
	}

	build, err := s.processXML(data, SourcePoBFile, path, hash)
	if err != nil {
		return Build{}, err
	}

	return s.saveActive(ctx, build)
}

// reuseExisting reactivates and returns an already-stored build with the same
// content hash. done is true when the caller should stop (the build was reused
// or an error occurred).
func (s *Service) reuseExisting(ctx context.Context, hash string) (Build, bool, error) {
	id, ok, err := s.repo.FindIDBySourceHash(ctx, hash)
	if err != nil {
		return Build{}, true, fmt.Errorf("builds: checking existing hash: %w", err)
	}
	if !ok {
		return Build{}, false, nil
	}

	if err := s.repo.SetActive(ctx, id); err != nil {
		return Build{}, true, fmt.Errorf("builds: activating existing build: %w", err)
	}
	build, err := s.repo.FindByID(ctx, id)

	return build, true, err
}

func (s *Service) saveActive(ctx context.Context, build Build) (Build, error) {
	if err := s.repo.Save(ctx, build); err != nil {
		return Build{}, fmt.Errorf("builds: saving build: %w", err)
	}
	if err := s.repo.SetActive(ctx, build.ID); err != nil {
		return Build{}, fmt.Errorf("builds: activating build: %w", err)
	}

	return build, nil
}

// Update re-fetches a build from its original URL. When the source is unchanged
// (same hash) it returns the stored build and changed=false, avoiding redundant
// reprocessing. Otherwise it reprocesses, preserving the current stage when it
// remains in range.
func (s *Service) Update(ctx context.Context, buildID string) (Build, bool, error) {
	existing, err := s.repo.FindByID(ctx, buildID)
	if err != nil {
		return Build{}, false, err
	}
	if existing.SourceURL == "" {
		return Build{}, false, errors.New("builds: build has no source URL to update from")
	}

	res, err := s.registry.Import(ctx, existing.SourceURL)
	if err != nil {
		return Build{}, false, err
	}

	hash := hashCode(res.Code)
	if hash == existing.SourceHash {
		return existing, false, nil
	}

	xml, err := pob.Decode(res.Code)
	if err != nil {
		return Build{}, false, err
	}

	updated, err := s.processXML(xml, SourceType(res.Source), res.URL, hash)
	if err != nil {
		return Build{}, false, err
	}
	updated.ID = existing.ID
	updated.ImportedAt = existing.ImportedAt
	updated.CurrentStage = updated.ClampStage(existing.CurrentStage)
	rekeyStages(&updated)

	if err := s.repo.Save(ctx, updated); err != nil {
		return Build{}, false, fmt.Errorf("builds: saving updated build: %w", err)
	}

	return updated, true, nil
}

// SetCurrentStage persists the selected stage for a build.
func (s *Service) SetCurrentStage(ctx context.Context, buildID string, stageOrder int) error {
	return s.repo.SetCurrentStage(ctx, buildID, stageOrder)
}

// Active returns the currently active build.
func (s *Service) Active(ctx context.Context) (Build, error) {
	return s.repo.FindActive(ctx)
}

// processXML turns already-decoded PoB XML into a fully assembled build with
// precomputed stage diffs. It performs no I/O.
func (s *Service) processXML(xml []byte, source SourceType, url, hash string) (Build, error) {
	parsed, err := pob.Parse(xml)
	if err != nil {
		return Build{}, err
	}

	norm, err := pob.Normalize(parsed)
	if err != nil {
		return Build{}, err
	}

	now := s.now()
	build := Build{
		ID:           id.New(),
		Name:         norm.Name,
		Class:        norm.Class,
		Ascendancy:   norm.Ascendancy,
		TreeVersion:  norm.TreeVersion,
		SourceType:   source,
		SourceURL:    url,
		SourceHash:   hash,
		CurrentStage: norm.ActiveStage,
		ImportedAt:   now,
		UpdatedAt:    now,
		Stages:       s.assembleStages(norm),
	}
	build.CurrentStage = build.ClampStage(build.CurrentStage)
	for i := range build.Stages {
		build.Stages[i].BuildID = build.ID
	}
	computeStageDiffs(&build)

	return build, nil
}

func (s *Service) assembleStages(norm pob.NormalizedBuild) []BuildStage {
	stages := make([]BuildStage, 0, len(norm.Stages))
	for _, ns := range norm.Stages {
		stageID := id.New()
		stages = append(stages, BuildStage{
			ID:                stageID,
			Name:              ns.Name,
			Order:             ns.Order,
			PassiveNodes:      ns.Nodes,
			SkillGroups:       s.assembleGroups(stageID, ns.SkillGroups, norm.TreeVersion),
			Association:       ns.Association,
			MasterySelections: ns.MasterySelections,
		})
	}

	return stages
}

func (s *Service) assembleGroups(
	stageID string,
	groups []pob.NormalizedSkillGroup,
	treeVersion string,
) []SkillGroup {
	out := make([]SkillGroup, 0, len(groups))
	for _, g := range groups {
		out = append(out, SkillGroup{
			ID:      id.New(),
			StageID: stageID,
			Label:   g.Label,
			Slot:    g.Slot,
			Enabled: g.Enabled,
			IsMain:  g.IsMain,
			Gems:    s.assembleGems(g.Gems, treeVersion),
		})
	}

	return out
}

func (s *Service) assembleGems(gems []pob.NormalizedGem, treeVersion string) []Gem {
	out := make([]Gem, 0, len(gems))
	for _, g := range gems {
		gem := Gem{
			Name:      g.Name,
			Level:     g.Level,
			Quality:   g.Quality,
			Enabled:   g.Enabled,
			IsSupport: g.IsSupport,
		}
		if s.levels != nil {
			if req, ok := s.levels.RequiredLevel(g.Name, treeVersion); ok {
				gem.RequiredLevel = &req
			}
		}
		out = append(out, gem)
	}

	return out
}

// computeStageDiffs fills NewNodes/RemovedNodes for every stage relative to its
// predecessor. Stage zero has no predecessor, so all its nodes are new.
func computeStageDiffs(build *Build) {
	var prevNodes []int
	for i := range build.Stages {
		added, removed := DiffNodes(prevNodes, build.Stages[i].PassiveNodes)
		build.Stages[i].NewNodes = added
		build.Stages[i].RemovedNodes = removed
		prevNodes = build.Stages[i].PassiveNodes
	}
}

// rekeyStages reassigns stage and group ids after an update so foreign keys stay
// consistent while the build id is preserved.
func rekeyStages(build *Build) {
	for i := range build.Stages {
		stageID := id.New()
		build.Stages[i].ID = stageID
		build.Stages[i].BuildID = build.ID
		for j := range build.Stages[i].SkillGroups {
			build.Stages[i].SkillGroups[j].ID = id.New()
			build.Stages[i].SkillGroups[j].StageID = stageID
		}
	}
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))

	return hex.EncodeToString(sum[:])
}
