// Package repositories implements the domain persistence contracts on top of
// SQLite. Repositories accept *sql.DB directly so they do not depend on the
// storage package, keeping the dependency graph acyclic.
package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/pedrobalen/poe-build-overlay/internal/builds"
)

const activeBuildKey = "active_build_id"

// BuildRepo persists builds and their stages, skill groups, and gems.
type BuildRepo struct {
	db *sql.DB
}

// NewBuildRepo returns a BuildRepo backed by db.
func NewBuildRepo(db *sql.DB) *BuildRepo {
	return &BuildRepo{db: db}
}

var _ builds.BuildRepository = (*BuildRepo)(nil)

// Save writes a build and all of its children atomically, replacing any prior
// stages so an update never leaves stale rows.
func (r *BuildRepo) Save(ctx context.Context, b builds.Build) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := upsertBuild(ctx, tx, b); err != nil {
		return err
	}
	if err := replaceStages(ctx, tx, b); err != nil {
		return err
	}
	if err := upsertProgress(ctx, tx, b); err != nil {
		return err
	}

	return tx.Commit()
}

func upsertProgress(ctx context.Context, tx *sql.Tx, b builds.Build) error {
	const q = `INSERT INTO build_progress (build_id, current_stage, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(build_id) DO UPDATE SET current_stage = excluded.current_stage, updated_at = excluded.updated_at`
	if _, err := tx.ExecContext(ctx, q, b.ID, b.CurrentStage, formatTime(b.UpdatedAt)); err != nil {
		return fmt.Errorf("repositories: upserting progress: %w", err)
	}

	return nil
}

func upsertBuild(ctx context.Context, tx *sql.Tx, b builds.Build) error {
	const q = `INSERT INTO builds
		(id, name, class, ascendancy, tree_version, source_type, source_url, source_hash, current_stage, imported_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, class=excluded.class, ascendancy=excluded.ascendancy,
			tree_version=excluded.tree_version, source_type=excluded.source_type,
			source_url=excluded.source_url, source_hash=excluded.source_hash,
			current_stage=excluded.current_stage, updated_at=excluded.updated_at`

	_, err := tx.ExecContext(ctx, q,
		b.ID,
		b.Name,
		b.Class,
		b.Ascendancy,
		b.TreeVersion,
		string(b.SourceType),
		b.SourceURL,
		b.SourceHash,
		b.CurrentStage,
		formatTime(b.ImportedAt),
		formatTime(b.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("repositories: upserting build: %w", err)
	}

	const src = `INSERT INTO build_sources (build_id, source_type, source_url, source_hash)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(build_id) DO UPDATE SET
			source_type=excluded.source_type, source_url=excluded.source_url, source_hash=excluded.source_hash`
	if _, err := tx.ExecContext(ctx, src, b.ID, string(b.SourceType), b.SourceURL, b.SourceHash); err != nil {
		return fmt.Errorf("repositories: upserting build source: %w", err)
	}

	return nil
}

func replaceStages(ctx context.Context, tx *sql.Tx, b builds.Build) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM build_stages WHERE build_id = ?`, b.ID); err != nil {
		return fmt.Errorf("repositories: clearing stages: %w", err)
	}

	for _, stage := range b.Stages {
		if err := insertStage(ctx, tx, b.ID, stage); err != nil {
			return err
		}
	}

	return nil
}

func insertStage(ctx context.Context, tx *sql.Tx, buildID string, stage builds.BuildStage) error {
	const q = `INSERT INTO build_stages
		(id, build_id, name, stage_order, character_level, association, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, q,
		stage.ID,
		buildID,
		stage.Name,
		stage.Order,
		nullableInt(stage.CharacterLevel),
		string(stage.Association),
		stage.Notes,
	); err != nil {
		return fmt.Errorf("repositories: inserting stage: %w", err)
	}

	if err := insertNodes(ctx, tx, stage); err != nil {
		return err
	}
	if err := insertMasteries(ctx, tx, stage); err != nil {
		return err
	}

	for i, group := range stage.SkillGroups {
		if err := insertGroup(ctx, tx, stage.ID, i, group); err != nil {
			return err
		}
	}

	return nil
}

func insertMasteries(ctx context.Context, tx *sql.Tx, stage builds.BuildStage) error {
	const q = `INSERT OR IGNORE INTO build_stage_masteries (stage_id, node_id, effect_id) VALUES (?, ?, ?)`
	for node, effect := range stage.MasterySelections {
		if _, err := tx.ExecContext(ctx, q, stage.ID, node, effect); err != nil {
			return fmt.Errorf("repositories: inserting stage mastery: %w", err)
		}
	}

	return nil
}

func insertNodes(ctx context.Context, tx *sql.Tx, stage builds.BuildStage) error {
	const q = `INSERT OR IGNORE INTO build_stage_nodes (stage_id, node_id, role) VALUES (?, ?, ?)`

	roles := []struct {
		role  string
		nodes []int
	}{
		{"current", stage.PassiveNodes},
		{"new", stage.NewNodes},
		{"removed", stage.RemovedNodes},
	}

	for _, group := range roles {
		for _, node := range group.nodes {
			if _, err := tx.ExecContext(ctx, q, stage.ID, node, group.role); err != nil {
				return fmt.Errorf("repositories: inserting stage node: %w", err)
			}
		}
	}

	return nil
}

func insertGroup(ctx context.Context, tx *sql.Tx, stageID string, order int, g builds.SkillGroup) error {
	const q = `INSERT INTO build_skill_groups
		(id, stage_id, label, slot, enabled, is_main, group_order)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, q,
		g.ID,
		stageID,
		g.Label,
		g.Slot,
		boolToInt(g.Enabled),
		boolToInt(g.IsMain),
		order,
	); err != nil {
		return fmt.Errorf("repositories: inserting skill group: %w", err)
	}

	for i, gem := range g.Gems {
		if err := insertGem(ctx, tx, g.ID, i, gem); err != nil {
			return err
		}
	}

	return nil
}

func insertGem(ctx context.Context, tx *sql.Tx, groupID string, order int, gem builds.Gem) error {
	const q = `INSERT INTO build_gems
		(group_id, gem_order, name, level, required_level, quality, enabled, is_support)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, q,
		groupID,
		order,
		gem.Name,
		gem.Level,
		nullableInt(gem.RequiredLevel),
		gem.Quality,
		boolToInt(gem.Enabled),
		boolToInt(gem.IsSupport),
	); err != nil {
		return fmt.Errorf("repositories: inserting gem: %w", err)
	}

	return nil
}

// FindByID loads a fully hydrated build.
func (r *BuildRepo) FindByID(ctx context.Context, id string) (builds.Build, error) {
	build, err := r.scanBuild(ctx, `SELECT id, name, class, ascendancy, tree_version, source_type, source_url, source_hash, current_stage, imported_at, updated_at FROM builds WHERE id = ?`, id)
	if err != nil {
		return builds.Build{}, err
	}

	if err := r.loadStages(ctx, &build); err != nil {
		return builds.Build{}, err
	}

	return build, nil
}

// FindActive loads the build referenced by the active_build_id setting.
func (r *BuildRepo) FindActive(ctx context.Context) (builds.Build, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key = ?`, activeBuildKey).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && id == "") {
		return builds.Build{}, builds.ErrNoActiveBuild
	}
	if err != nil {
		return builds.Build{}, fmt.Errorf("repositories: reading active build id: %w", err)
	}

	return r.FindByID(ctx, id)
}

// List returns lightweight summaries of every stored build.
func (r *BuildRepo) List(ctx context.Context) ([]builds.Summary, error) {
	activeID, _ := r.activeID(ctx)

	const q = `SELECT b.id, b.name, b.class, b.ascendancy, b.source_type,
		(SELECT COUNT(*) FROM build_stages s WHERE s.build_id = b.id)
		FROM builds b ORDER BY b.updated_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("repositories: listing builds: %w", err)
	}
	defer rows.Close()

	summaries := []builds.Summary{}
	for rows.Next() {
		var s builds.Summary
		var sourceType string
		if err := rows.Scan(&s.ID, &s.Name, &s.Class, &s.Ascendancy, &sourceType, &s.StageCount); err != nil {
			return nil, err
		}
		s.SourceType = builds.SourceType(sourceType)
		s.IsActive = s.ID == activeID
		summaries = append(summaries, s)
	}

	return summaries, rows.Err()
}

// SetActive records which build is active in app_settings.
func (r *BuildRepo) SetActive(ctx context.Context, id string) error {
	const q = `INSERT INTO app_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	if _, err := r.db.ExecContext(ctx, q, activeBuildKey, id); err != nil {
		return fmt.Errorf("repositories: setting active build: %w", err)
	}

	return nil
}

// SetCurrentStage persists the selected stage on the build and its progress row.
func (r *BuildRepo) SetCurrentStage(ctx context.Context, buildID string, stageOrder int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `UPDATE builds SET current_stage = ? WHERE id = ?`, stageOrder, buildID); err != nil {
		return fmt.Errorf("repositories: updating current stage: %w", err)
	}

	const prog = `INSERT INTO build_progress (build_id, current_stage, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(build_id) DO UPDATE SET current_stage = excluded.current_stage, updated_at = excluded.updated_at`
	if _, err := tx.ExecContext(ctx, prog, buildID, stageOrder, formatTime(time.Now())); err != nil {
		return fmt.Errorf("repositories: updating progress: %w", err)
	}

	return tx.Commit()
}

// Delete removes a build and clears the active pointer when it referenced it.
func (r *BuildRepo) Delete(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM builds WHERE id = ?`, id); err != nil {
		return fmt.Errorf("repositories: deleting build: %w", err)
	}

	if activeID, _ := r.activeID(ctx); activeID == id {
		if _, err := r.db.ExecContext(ctx, `DELETE FROM app_settings WHERE key = ?`, activeBuildKey); err != nil {
			return fmt.Errorf("repositories: clearing active build: %w", err)
		}
	}

	return nil
}

// FindIDBySourceHash reports whether a build with the given content hash exists.
func (r *BuildRepo) FindIDBySourceHash(ctx context.Context, hash string) (string, bool, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM builds WHERE source_hash = ?`, hash).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("repositories: looking up source hash: %w", err)
	}

	return id, true, nil
}

func (r *BuildRepo) activeID(ctx context.Context) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key = ?`, activeBuildKey).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	return id, err
}
