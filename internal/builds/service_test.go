package builds_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	"github.com/pedrobalen/poe-build-overlay/internal/importers"
	"github.com/pedrobalen/poe-build-overlay/internal/storage"
	"github.com/pedrobalen/poe-build-overlay/internal/storage/repositories"
)

const pobFileXML = `<?xml version="1.0" encoding="UTF-8"?>
<PathOfBuilding>
  <Build level="90" className="Duelist" ascendClassName="Slayer" mainSocketGroup="1"/>
  <Tree activeSpec="2">
    <Spec title="Act 3" treeVersion="3_25" nodes="100,101" masteryEffects="{100,5}"/>
    <Spec title="Endgame" treeVersion="3_25" nodes="100,101,200" masteryEffects="{100,5},{200,9}"/>
  </Tree>
  <Skills activeSkillSet="2">
    <SkillSet id="1" title="Act 3">
      <Skill slot="Weapon 1" enabled="true" mainActiveSkill="1">
        <Gem nameSpec="Ground Slam" level="10" quality="0" enabled="true"/>
      </Skill>
    </SkillSet>
    <SkillSet id="2" title="Endgame">
      <Skill slot="Weapon 1" enabled="true" mainActiveSkill="1">
        <Gem nameSpec="Static Strike" level="20" quality="20" enabled="true"/>
        <Gem nameSpec="Melee Physical Damage Support" level="20" quality="20" enabled="true" skillId="SupportMeleePhysicalDamage"/>
      </Skill>
    </SkillSet>
  </Skills>
</PathOfBuilding>`

func newService(t *testing.T) (*builds.Service, *repositories.BuildRepo) {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "svc.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := repositories.NewBuildRepo(db.DB)
	svc := builds.NewService(importers.NewRegistry(http.DefaultClient), repo, nil)

	return svc, repo
}

func TestImportFile(t *testing.T) {
	svc, repo := newService(t)
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "build.xml")
	if err := os.WriteFile(path, []byte(pobFileXML), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	build, err := svc.ImportFile(ctx, path)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}

	if build.SourceType != builds.SourcePoBFile || build.SourceURL != path {
		t.Fatalf("unexpected source: %s %q", build.SourceType, build.SourceURL)
	}
	if len(build.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(build.Stages))
	}

	// Mastery selections must parse ({node,effect}) and persist through a reload.
	loaded, err := repo.FindActive(ctx)
	if err != nil {
		t.Fatalf("FindActive: %v", err)
	}
	endgame := loaded.Stages[1]
	if got := endgame.MasterySelections[100]; got != 5 {
		t.Fatalf("mastery node 100 effect = %d, want 5", got)
	}
	if got := endgame.MasterySelections[200]; got != 9 {
		t.Fatalf("mastery node 200 effect = %d, want 9", got)
	}

	// The support gem must be flagged via its skillId.
	var sawSupport bool
	for _, group := range endgame.SkillGroups {
		for _, gem := range group.Gems {
			if gem.Name == "Melee Physical Damage Support" && gem.IsSupport {
				sawSupport = true
			}
		}
	}
	if !sawSupport {
		t.Fatal("expected the support gem to be flagged")
	}
}

func TestImportFileDeduplicates(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "build.xml")
	_ = os.WriteFile(path, []byte(pobFileXML), 0o644)

	first, err := svc.ImportFile(ctx, path)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	second, err := svc.ImportFile(ctx, path)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("re-importing the same file should reuse the build: %s != %s", first.ID, second.ID)
	}
}
