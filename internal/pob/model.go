// Package pob decodes, parses, and normalizes Path of Building export data.
//
// The pipeline is: Decode (Base64 URL-safe + zlib) -> Parse (XML -> ParsedBuild)
// -> Normalize (ParsedBuild -> NormalizedBuild). Each stage is independently
// testable and free of I/O so the heavy work can run off the UI thread during
// import only.
package pob

// ParsedBuild mirrors the relevant subset of a Path of Building export. It is
// the raw, un-normalized view of the XML and preserves the author's ordering.
type ParsedBuild struct {
	Level           int
	ClassName       string
	Ascendancy      string
	MainSocketGroup int
	TreeVersion     string
	ActiveSpec      int // 1-based index into Specs; 0 when unspecified
	Specs           []ParsedSpec
	ActiveSkillSet  int // 1-based index into SkillSets; 0 when unspecified
	SkillSets       []ParsedSkillSet
}

// ParsedSpec is a single saved passive tree ("Ato 3", "Endgame", ...).
type ParsedSpec struct {
	Title       string
	TreeVersion string
	ClassID     int
	AscendID    int
	Nodes       []int
	URL         string
	// MasterySelections maps a mastery node id to the chosen effect id.
	MasterySelections map[int]int
}

// ParsedSkillSet is a named collection of socket groups.
type ParsedSkillSet struct {
	ID     int
	Title  string
	Groups []ParsedSocketGroup
}

// ParsedSocketGroup is one linked group of gems in a gear slot.
type ParsedSocketGroup struct {
	Label   string
	Slot    string
	Enabled bool
	IsMain  bool
	Gems    []ParsedGem
}

// ParsedGem is a single gem within a socket group.
type ParsedGem struct {
	Name      string
	Level     int
	Quality   int
	Enabled   bool
	IsSupport bool
}

// StageAssociation records how a stage's skill set was matched to its tree.
// The UI surfaces this so an ambiguous match is never presented as certain.
type StageAssociation string

const (
	// AssocByName means the tree and skill set shared an equivalent title.
	AssocByName StageAssociation = "name"
	// AssocByPosition means they were matched by corresponding index.
	AssocByPosition StageAssociation = "position"
	// AssocDefault means the build's active skill set was used as a fallback.
	AssocDefault StageAssociation = "default"
	// AssocNone means no skill set could be associated with the stage.
	AssocNone StageAssociation = "none"
)

// NormalizedBuild is the ordered, UI-ready view produced by Normalize. Stage
// order is exactly the order of the author's saved trees; progression is never
// invented.
type NormalizedBuild struct {
	Name        string
	Class       string
	Ascendancy  string
	TreeVersion string
	ActiveStage int // 0-based index into Stages
	Stages      []NormalizedStage
}

// NormalizedStage is one progression step with its tree and associated skills.
type NormalizedStage struct {
	Name              string
	Order             int
	Nodes             []int
	SkillGroups       []NormalizedSkillGroup
	Association       StageAssociation
	MasterySelections map[int]int // mastery node id -> chosen effect id
}

// NormalizedSkillGroup is a socket group ready for display.
type NormalizedSkillGroup struct {
	Label   string
	Slot    string
	Enabled bool
	IsMain  bool
	Gems    []NormalizedGem
}

// NormalizedGem is a gem ready for display. RequiredLevel is intentionally
// absent here: per the plan it must come from versioned local gem data, never
// from inference over the stage name.
type NormalizedGem struct {
	Name      string
	Level     int
	Quality   int
	Enabled   bool
	IsSupport bool
}
