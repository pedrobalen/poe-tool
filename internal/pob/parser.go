package pob

import (
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Parse interprets Path of Building export XML into a ParsedBuild.
//
// The root element name is intentionally not constrained: PoB and community
// forks emit slightly different roots ("PathOfBuilding", ...), and omitting
// XMLName lets encoding/xml match whatever the outermost element is.
func Parse(data []byte) (ParsedBuild, error) {
	if len(data) == 0 {
		return ParsedBuild{}, errors.New("pob: empty XML")
	}

	var doc xmlDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return ParsedBuild{}, fmt.Errorf("pob: parsing XML: %w", err)
	}

	build := ParsedBuild{
		Level:           doc.Build.Level,
		ClassName:       strings.TrimSpace(doc.Build.ClassName),
		Ascendancy:      strings.TrimSpace(doc.Build.AscendClassName),
		MainSocketGroup: doc.Build.MainSocketGroup,
		ActiveSpec:      doc.Tree.ActiveSpec,
		ActiveSkillSet:  doc.Skills.ActiveSkillSet,
		Specs:           specsFromXML(doc.Tree.Specs),
		SkillSets:       skillSetsFromXML(doc.Skills),
	}

	if len(build.Specs) > 0 {
		build.TreeVersion = build.Specs[0].TreeVersion
	}
	if len(build.Specs) == 0 && len(build.SkillSets) == 0 {
		return ParsedBuild{}, errors.New("pob: build has neither a passive tree nor a skill set")
	}

	return build, nil
}

func specsFromXML(specs []xmlSpec) []ParsedSpec {
	out := make([]ParsedSpec, 0, len(specs))
	for _, s := range specs {
		out = append(out, ParsedSpec{
			Title:             strings.TrimSpace(s.Title),
			TreeVersion:       strings.TrimSpace(s.TreeVersion),
			ClassID:           s.ClassID,
			AscendID:          s.AscendClassID,
			Nodes:             parseNodeList(s.Nodes),
			URL:               strings.TrimSpace(s.URL),
			MasterySelections: parseMasteryEffects(s.MasteryEffects),
		})
	}

	return out
}

// masteryEffectRe matches the "{node,effect}" pairs Path of Building stores in a
// spec's masteryEffects attribute.
var masteryEffectRe = regexp.MustCompile(`\{(\d+),(\d+)\}`)

// parseMasteryEffects turns "{nodeId,effectId},{...}" into a node->effect map.
func parseMasteryEffects(raw string) map[int]int {
	selections := map[int]int{}
	for _, m := range masteryEffectRe.FindAllStringSubmatch(raw, -1) {
		node, err1 := strconv.Atoi(m[1])
		effect, err2 := strconv.Atoi(m[2])
		if err1 == nil && err2 == nil {
			selections[node] = effect
		}
	}

	return selections
}

// skillSetsFromXML handles both the modern layout (Skills > SkillSet > Skill)
// and the legacy layout (Skills > Skill) by wrapping bare skills in a single
// default set so downstream code always sees named sets.
func skillSetsFromXML(skills xmlSkills) []ParsedSkillSet {
	if len(skills.SkillSets) > 0 {
		out := make([]ParsedSkillSet, 0, len(skills.SkillSets))
		for _, set := range skills.SkillSets {
			out = append(out, ParsedSkillSet{
				ID:     set.ID,
				Title:  strings.TrimSpace(set.Title),
				Groups: groupsFromXML(set.Skills),
			})
		}

		return out
	}

	if len(skills.Skills) == 0 {
		return []ParsedSkillSet{}
	}

	return []ParsedSkillSet{{
		ID:     1,
		Title:  "",
		Groups: groupsFromXML(skills.Skills),
	}}
}

func groupsFromXML(skills []xmlSkill) []ParsedSocketGroup {
	out := make([]ParsedSocketGroup, 0, len(skills))
	for _, s := range skills {
		out = append(out, ParsedSocketGroup{
			Label:   strings.TrimSpace(s.Label),
			Slot:    strings.TrimSpace(s.Slot),
			Enabled: parseBoolAttr(s.Enabled, true),
			IsMain:  isMainSkill(s.MainActiveSkill),
			Gems:    gemsFromXML(s.Gems),
		})
	}

	return out
}

func gemsFromXML(gems []xmlGem) []ParsedGem {
	out := make([]ParsedGem, 0, len(gems))
	for _, g := range gems {
		name := strings.TrimSpace(g.NameSpec)
		out = append(out, ParsedGem{
			Name:      name,
			Level:     g.Level,
			Quality:   g.Quality,
			Enabled:   parseBoolAttr(g.Enabled, true),
			IsSupport: isSupportGem(name, g.SkillID),
		})
	}

	return out
}

// parseNodeList turns "123,456,789" into []int, skipping empty and malformed
// entries rather than failing the whole import for one bad token.
func parseNodeList(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int{}
	}

	parts := strings.Split(raw, ",")
	nodes := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.Atoi(p); err == nil {
			nodes = append(nodes, n)
		}
	}

	return nodes
}

// parseBoolAttr interprets a PoB boolean attribute, falling back to def when
// the attribute is absent or unrecognized.
func parseBoolAttr(raw string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1":
		return true
	case "false", "0":
		return false
	default:
		return def
	}
}

// isMainSkill reports whether a skill group's mainActiveSkill attribute marks
// it as active. PoB uses "nil" or "0" for inactive groups.
func isMainSkill(raw string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || raw == "nil" || raw == "0" {
		return false
	}

	return true
}

// isSupportGem classifies a gem as a support. The skillId is authoritative when
// present (PoB prefixes support skill ids with "Support"); otherwise it falls
// back to the "Support" name suffix.
func isSupportGem(name, skillID string) bool {
	if strings.HasPrefix(skillID, "Support") {
		return true
	}

	return strings.HasSuffix(name, "Support")
}

// xmlDocument omits XMLName so any root element name unmarshals successfully.
type xmlDocument struct {
	Build  xmlBuild  `xml:"Build"`
	Tree   xmlTree   `xml:"Tree"`
	Skills xmlSkills `xml:"Skills"`
}

type xmlBuild struct {
	Level           int    `xml:"level,attr"`
	ClassName       string `xml:"className,attr"`
	AscendClassName string `xml:"ascendClassName,attr"`
	MainSocketGroup int    `xml:"mainSocketGroup,attr"`
}

type xmlTree struct {
	ActiveSpec int       `xml:"activeSpec,attr"`
	Specs      []xmlSpec `xml:"Spec"`
}

type xmlSpec struct {
	Title          string `xml:"title,attr"`
	TreeVersion    string `xml:"treeVersion,attr"`
	ClassID        int    `xml:"classId,attr"`
	AscendClassID  int    `xml:"ascendClassId,attr"`
	Nodes          string `xml:"nodes,attr"`
	MasteryEffects string `xml:"masteryEffects,attr"`
	URL            string `xml:"URL"`
}

type xmlSkills struct {
	ActiveSkillSet int           `xml:"activeSkillSet,attr"`
	SkillSets      []xmlSkillSet `xml:"SkillSet"`
	Skills         []xmlSkill    `xml:"Skill"` // legacy layout without skill sets
}

type xmlSkillSet struct {
	ID     int        `xml:"id,attr"`
	Title  string     `xml:"title,attr"`
	Skills []xmlSkill `xml:"Skill"`
}

type xmlSkill struct {
	Label           string   `xml:"label,attr"`
	Slot            string   `xml:"slot,attr"`
	Enabled         string   `xml:"enabled,attr"`
	MainActiveSkill string   `xml:"mainActiveSkill,attr"`
	Gems            []xmlGem `xml:"Gem"`
}

type xmlGem struct {
	NameSpec string `xml:"nameSpec,attr"`
	SkillID  string `xml:"skillId,attr"`
	Level    int    `xml:"level,attr"`
	Quality  int    `xml:"quality,attr"`
	Enabled  string `xml:"enabled,attr"`
}
