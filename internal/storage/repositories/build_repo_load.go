package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	"github.com/pedrobalen/poe-build-overlay/internal/pob"
)

// ErrBuildNotFound is returned when a build id has no row.
var ErrBuildNotFound = errors.New("repositories: build not found")

func (r *BuildRepo) scanBuild(ctx context.Context, query string, args ...any) (builds.Build, error) {
	var (
		b          builds.Build
		sourceType string
		importedAt string
		updatedAt  string
	)

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&b.ID,
		&b.Name,
		&b.Class,
		&b.Ascendancy,
		&b.TreeVersion,
		&sourceType,
		&b.SourceURL,
		&b.SourceHash,
		&b.CurrentStage,
		&importedAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return builds.Build{}, ErrBuildNotFound
	}
	if err != nil {
		return builds.Build{}, fmt.Errorf("repositories: scanning build: %w", err)
	}

	b.SourceType = builds.SourceType(sourceType)
	b.ImportedAt = parseTime(importedAt)
	b.UpdatedAt = parseTime(updatedAt)

	return b, nil
}

func (r *BuildRepo) loadStages(ctx context.Context, b *builds.Build) error {
	const q = `SELECT id, name, stage_order, character_level, association, notes
		FROM build_stages WHERE build_id = ? ORDER BY stage_order`
	rows, err := r.db.QueryContext(ctx, q, b.ID)
	if err != nil {
		return fmt.Errorf("repositories: loading stages: %w", err)
	}
	defer rows.Close()

	stages := []builds.BuildStage{}
	for rows.Next() {
		var (
			s           builds.BuildStage
			level       sql.NullInt64
			association string
		)
		if err := rows.Scan(&s.ID, &s.Name, &s.Order, &level, &association, &s.Notes); err != nil {
			return err
		}
		s.BuildID = b.ID
		s.Association = pob.StageAssociation(association)
		if level.Valid {
			v := int(level.Int64)
			s.CharacterLevel = &v
		}
		stages = append(stages, s)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range stages {
		if err := r.loadStageNodes(ctx, &stages[i]); err != nil {
			return err
		}
		if err := r.loadStageMasteries(ctx, &stages[i]); err != nil {
			return err
		}
		if err := r.loadStageGroups(ctx, &stages[i]); err != nil {
			return err
		}
	}

	b.Stages = stages

	return nil
}

func (r *BuildRepo) loadStageMasteries(ctx context.Context, stage *builds.BuildStage) error {
	const q = `SELECT node_id, effect_id FROM build_stage_masteries WHERE stage_id = ?`
	rows, err := r.db.QueryContext(ctx, q, stage.ID)
	if err != nil {
		return fmt.Errorf("repositories: loading stage masteries: %w", err)
	}
	defer rows.Close()

	stage.MasterySelections = map[int]int{}
	for rows.Next() {
		var node, effect int
		if err := rows.Scan(&node, &effect); err != nil {
			return err
		}
		stage.MasterySelections[node] = effect
	}

	return rows.Err()
}

func (r *BuildRepo) loadStageNodes(ctx context.Context, stage *builds.BuildStage) error {
	const q = `SELECT node_id, role FROM build_stage_nodes WHERE stage_id = ?`
	rows, err := r.db.QueryContext(ctx, q, stage.ID)
	if err != nil {
		return fmt.Errorf("repositories: loading stage nodes: %w", err)
	}
	defer rows.Close()

	stage.PassiveNodes = []int{}
	stage.NewNodes = []int{}
	stage.RemovedNodes = []int{}

	for rows.Next() {
		var (
			node int
			role string
		)
		if err := rows.Scan(&node, &role); err != nil {
			return err
		}
		switch role {
		case "current":
			stage.PassiveNodes = append(stage.PassiveNodes, node)
		case "new":
			stage.NewNodes = append(stage.NewNodes, node)
		case "removed":
			stage.RemovedNodes = append(stage.RemovedNodes, node)
		}
	}

	return rows.Err()
}

func (r *BuildRepo) loadStageGroups(ctx context.Context, stage *builds.BuildStage) error {
	const q = `SELECT id, label, slot, enabled, is_main
		FROM build_skill_groups WHERE stage_id = ? ORDER BY group_order`
	rows, err := r.db.QueryContext(ctx, q, stage.ID)
	if err != nil {
		return fmt.Errorf("repositories: loading skill groups: %w", err)
	}
	defer rows.Close()

	groups := []builds.SkillGroup{}
	for rows.Next() {
		var (
			g       builds.SkillGroup
			enabled int
			isMain  int
		)
		if err := rows.Scan(&g.ID, &g.Label, &g.Slot, &enabled, &isMain); err != nil {
			return err
		}
		g.StageID = stage.ID
		g.Enabled = enabled != 0
		g.IsMain = isMain != 0
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range groups {
		if err := r.loadGems(ctx, &groups[i]); err != nil {
			return err
		}
	}

	stage.SkillGroups = groups

	return nil
}

func (r *BuildRepo) loadGems(ctx context.Context, group *builds.SkillGroup) error {
	const q = `SELECT name, level, required_level, quality, enabled, is_support
		FROM build_gems WHERE group_id = ? ORDER BY gem_order`
	rows, err := r.db.QueryContext(ctx, q, group.ID)
	if err != nil {
		return fmt.Errorf("repositories: loading gems: %w", err)
	}
	defer rows.Close()

	gems := []builds.Gem{}
	for rows.Next() {
		var (
			gem      builds.Gem
			required sql.NullInt64
			enabled  int
			support  int
		)
		if err := rows.Scan(&gem.Name, &gem.Level, &required, &gem.Quality, &enabled, &support); err != nil {
			return err
		}
		gem.Enabled = enabled != 0
		gem.IsSupport = support != 0
		if required.Valid {
			v := int(required.Int64)
			gem.RequiredLevel = &v
		}
		gems = append(gems, gem)
	}

	group.Gems = gems

	return rows.Err()
}
