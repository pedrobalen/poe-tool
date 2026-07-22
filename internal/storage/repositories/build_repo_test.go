package repositories_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	"github.com/pedrobalen/poe-build-overlay/internal/pob"
	"github.com/pedrobalen/poe-build-overlay/internal/storage"
	"github.com/pedrobalen/poe-build-overlay/internal/storage/repositories"
)

func newTestDB(t *testing.T) *storage.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return db
}

func sampleBuild() builds.Build {
	now := time.Now()
	level := 12

	return builds.Build{
		ID:           "build-1",
		Name:         "Slayer Build",
		Class:        "Duelist",
		Ascendancy:   "Slayer",
		TreeVersion:  "3_25",
		SourceType:   builds.SourcePobbin,
		SourceURL:    "https://pobb.in/abc",
		SourceHash:   "hash-abc",
		CurrentStage: 1,
		ImportedAt:   now,
		UpdatedAt:    now,
		Stages: []builds.BuildStage{
			{
				ID:           "stage-0",
				BuildID:      "build-1",
				Name:         "Act 3",
				Order:        0,
				PassiveNodes: []int{100, 101},
				NewNodes:     []int{100, 101},
				RemovedNodes: []int{},
				Association:  pob.AssocByName,
				SkillGroups: []builds.SkillGroup{
					{
						ID: "grp-0", StageID: "stage-0", Label: "Main", Slot: "Weapon 1",
						Enabled: true, IsMain: true,
						Gems: []builds.Gem{
							{Name: "Ground Slam", Level: 10, Quality: 0, Enabled: true},
						},
					},
				},
			},
			{
				ID:             "stage-1",
				BuildID:        "build-1",
				Name:           "Endgame",
				Order:          1,
				CharacterLevel: &level,
				PassiveNodes:   []int{100, 101, 200},
				NewNodes:       []int{200},
				RemovedNodes:   []int{},
				Association:    pob.AssocByName,
				SkillGroups: []builds.SkillGroup{
					{
						ID: "grp-1", StageID: "stage-1", Label: "Main", Slot: "Weapon 1",
						Enabled: true, IsMain: true,
						Gems: []builds.Gem{
							{Name: "Static Strike", Level: 20, Quality: 20, Enabled: true},
							{Name: "Fortify Support", Level: 20, Enabled: true, IsSupport: true},
						},
					},
				},
			},
		},
	}
}

func TestBuildRoundTrip(t *testing.T) {
	db := newTestDB(t)
	repo := repositories.NewBuildRepo(db.DB)
	ctx := context.Background()

	original := sampleBuild()
	if err := repo.Save(ctx, original); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := repo.SetActive(ctx, original.ID); err != nil {
		t.Fatalf("set active: %v", err)
	}

	loaded, err := repo.FindActive(ctx)
	if err != nil {
		t.Fatalf("find active: %v", err)
	}

	if loaded.Name != original.Name || loaded.CurrentStage != 1 {
		t.Fatalf("unexpected build: %+v", loaded)
	}
	if len(loaded.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(loaded.Stages))
	}

	endgame := loaded.Stages[1]
	if endgame.Name != "Endgame" || endgame.CharacterLevel == nil || *endgame.CharacterLevel != 12 {
		t.Fatalf("endgame stage not restored: %+v", endgame)
	}
	if len(endgame.NewNodes) != 1 || endgame.NewNodes[0] != 200 {
		t.Fatalf("endgame new nodes not restored: %v", endgame.NewNodes)
	}
	if len(endgame.SkillGroups[0].Gems) != 2 {
		t.Fatalf("gems not restored: %+v", endgame.SkillGroups)
	}
	if !endgame.SkillGroups[0].Gems[1].IsSupport {
		t.Fatalf("support flag not restored")
	}
}

func TestSetCurrentStageAndActive(t *testing.T) {
	db := newTestDB(t)
	repo := repositories.NewBuildRepo(db.DB)
	ctx := context.Background()

	if err := repo.Save(ctx, sampleBuild()); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := repo.SetActive(ctx, "build-1"); err != nil {
		t.Fatalf("set active: %v", err)
	}
	if err := repo.SetCurrentStage(ctx, "build-1", 0); err != nil {
		t.Fatalf("set current stage: %v", err)
	}

	loaded, err := repo.FindByID(ctx, "build-1")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if loaded.CurrentStage != 0 {
		t.Fatalf("current stage = %d, want 0", loaded.CurrentStage)
	}

	summaries, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(summaries) != 1 || !summaries[0].IsActive || summaries[0].StageCount != 2 {
		t.Fatalf("unexpected summaries: %+v", summaries)
	}
}

func TestFindIDBySourceHash(t *testing.T) {
	db := newTestDB(t)
	repo := repositories.NewBuildRepo(db.DB)
	ctx := context.Background()

	if err := repo.Save(ctx, sampleBuild()); err != nil {
		t.Fatalf("save: %v", err)
	}

	id, ok, err := repo.FindIDBySourceHash(ctx, "hash-abc")
	if err != nil || !ok || id != "build-1" {
		t.Fatalf("FindIDBySourceHash = %q,%v,%v", id, ok, err)
	}

	if _, ok, _ := repo.FindIDBySourceHash(ctx, "missing"); ok {
		t.Fatal("expected miss for unknown hash")
	}
}

func TestDeleteClearsActive(t *testing.T) {
	db := newTestDB(t)
	repo := repositories.NewBuildRepo(db.DB)
	ctx := context.Background()

	_ = repo.Save(ctx, sampleBuild())
	_ = repo.SetActive(ctx, "build-1")

	if err := repo.Delete(ctx, "build-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.FindActive(ctx); err != builds.ErrNoActiveBuild {
		t.Fatalf("expected ErrNoActiveBuild, got %v", err)
	}
}
