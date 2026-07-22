package pob

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"testing"
)

const sampleXML = `<?xml version="1.0" encoding="UTF-8"?>
<PathOfBuilding>
  <Build level="90" className="Duelist" ascendClassName="Slayer" mainSocketGroup="1"/>
  <Tree activeSpec="2">
    <Spec title="Act 3" treeVersion="3_25" classId="3" nodes="100,101,102">
      <URL>https://example/act3</URL>
    </Spec>
    <Spec title="Endgame" treeVersion="3_25" classId="3" nodes="100,101,102,200,201">
      <URL>https://example/endgame</URL>
    </Spec>
  </Tree>
  <Skills activeSkillSet="2">
    <SkillSet id="1" title="Act 3">
      <Skill slot="Weapon 1" enabled="true" mainActiveSkill="1">
        <Gem nameSpec="Ground Slam" level="10" quality="0" enabled="true"/>
        <Gem nameSpec="Melee Physical Damage Support" level="8" quality="0" enabled="true"/>
      </Skill>
    </SkillSet>
    <SkillSet id="2" title="Endgame">
      <Skill slot="Weapon 1" enabled="true" mainActiveSkill="1">
        <Gem nameSpec="Static Strike" level="20" quality="20" enabled="true"/>
        <Gem nameSpec="Melee Physical Damage Support" level="20" quality="20" enabled="true"/>
        <Gem nameSpec="Fortify Support" level="20" quality="0" enabled="true"/>
      </Skill>
    </SkillSet>
  </Skills>
</PathOfBuilding>`

// encodePoB reproduces the Path of Building export encoding: zlib then URL-safe
// Base64.
func encodePoB(t *testing.T, xml string) string {
	t.Helper()

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write([]byte(xml)); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}

	return base64.URLEncoding.EncodeToString(buf.Bytes())
}

func TestDecodeRoundTrip(t *testing.T) {
	code := encodePoB(t, sampleXML)

	out, err := Decode(code)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(out, []byte(sampleXML)) {
		t.Fatalf("decoded XML does not match original")
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	if _, err := Decode("!!!not-base64!!!"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if _, err := Decode(""); err == nil {
		t.Fatal("expected error for empty code")
	}
}

func TestParse(t *testing.T) {
	build, err := Parse([]byte(sampleXML))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if build.ClassName != "Duelist" || build.Ascendancy != "Slayer" {
		t.Fatalf("unexpected class/ascendancy: %q/%q", build.ClassName, build.Ascendancy)
	}
	if len(build.Specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(build.Specs))
	}
	if build.Specs[0].Title != "Act 3" || build.TreeVersion != "3_25" {
		t.Fatalf("unexpected first spec: %+v", build.Specs[0])
	}
	if got := build.Specs[1].Nodes; len(got) != 5 {
		t.Fatalf("expected 5 nodes in endgame spec, got %v", got)
	}
	if len(build.SkillSets) != 2 {
		t.Fatalf("expected 2 skill sets, got %d", len(build.SkillSets))
	}
	gem := build.SkillSets[1].Groups[0].Gems[2]
	if gem.Name != "Fortify Support" || !gem.IsSupport {
		t.Fatalf("expected Fortify Support flagged as support, got %+v", gem)
	}
}

func TestParseRejectsEmptyBuild(t *testing.T) {
	if _, err := Parse([]byte(`<PathOfBuilding><Build/></PathOfBuilding>`)); err == nil {
		t.Fatal("expected error when build has no tree or skills")
	}
}

func TestNormalizeAssociatesByName(t *testing.T) {
	parsed, err := Parse([]byte(sampleXML))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	norm, err := Normalize(parsed)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if len(norm.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(norm.Stages))
	}
	if norm.ActiveStage != 1 {
		t.Fatalf("expected active stage 1 (Endgame), got %d", norm.ActiveStage)
	}

	act3 := norm.Stages[0]
	if act3.Name != "Act 3" || act3.Association != AssocByName {
		t.Fatalf("expected Act 3 matched by name, got %q/%s", act3.Name, act3.Association)
	}
	if len(act3.SkillGroups) != 1 || act3.SkillGroups[0].Gems[0].Name != "Ground Slam" {
		t.Fatalf("Act 3 stage did not get its own skill set: %+v", act3.SkillGroups)
	}

	endgame := norm.Stages[1]
	if endgame.SkillGroups[0].Gems[0].Name != "Static Strike" {
		t.Fatalf("Endgame stage did not get its own skill set: %+v", endgame.SkillGroups)
	}
}

func TestNormalizeSingleTreeNoInventedStages(t *testing.T) {
	xml := `<PathOfBuilding>
      <Build level="1" className="Witch"/>
      <Tree activeSpec="1"><Spec title="Final" nodes="1,2,3"/></Tree>
      <Skills><SkillSet id="1" title="Final"><Skill slot="Body"><Gem nameSpec="Firestorm" level="1" quality="0"/></Skill></SkillSet></Skills>
    </PathOfBuilding>`

	parsed, err := Parse([]byte(xml))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	norm, err := Normalize(parsed)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if len(norm.Stages) != 1 {
		t.Fatalf("single-tree build must yield exactly one stage, got %d", len(norm.Stages))
	}
}
