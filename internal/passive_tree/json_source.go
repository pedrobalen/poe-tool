package passive_tree

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
)

// JSONSource imports structural tree data from versioned JSON assets of the
// form "<version>.json" inside a filesystem (typically an embedded assets FS).
//
// The JSON schema intentionally mirrors a trimmed Path of Building tree export:
// only the fields the overlay renders are kept.
type JSONSource struct {
	fsys fs.FS
	dir  string
}

// NewJSONSource reads tree files from dir within fsys.
func NewJSONSource(fsys fs.FS, dir string) *JSONSource {
	return &JSONSource{fsys: fsys, dir: dir}
}

// Available reports whether a JSON file exists for the version.
func (s *JSONSource) Available(version string) bool {
	f, err := s.fsys.Open(s.pathFor(version))
	if err != nil {
		return false
	}
	_ = f.Close()

	return true
}

// Import parses the version's JSON file into TreeData, computing bounds.
func (s *JSONSource) Import(version string) (*TreeData, error) {
	raw, err := fs.ReadFile(s.fsys, s.pathFor(version))
	if err != nil {
		return nil, fmt.Errorf("passive_tree: reading %s: %w", version, err)
	}

	var doc jsonTree
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("passive_tree: parsing %s: %w", version, err)
	}

	return doc.toTreeData(version), nil
}

func (s *JSONSource) pathFor(version string) string {
	return path.Join(s.dir, version+".json")
}

type jsonTree struct {
	Nodes       []jsonNode       `json:"nodes"`
	Connections []jsonConnection `json:"connections"`
}

type jsonNode struct {
	ID        int            `json:"id"`
	X         float32        `json:"x"`
	Y         float32        `json:"y"`
	Kind      string         `json:"kind"`
	Name      string         `json:"name"`
	GroupID   int            `json:"group"`
	IsMastery bool           `json:"mastery"`
	Effects   map[int]string `json:"effects"`
}

type jsonConnection struct {
	From int `json:"from"`
	To   int `json:"to"`
}

func (d jsonTree) toTreeData(version string) *TreeData {
	data := &TreeData{
		Version:     version,
		Nodes:       make(map[int]Node, len(d.Nodes)),
		Connections: make([]Connection, 0, len(d.Connections)),
	}

	ids := make([]int, 0, len(d.Nodes))
	for _, n := range d.Nodes {
		kind := NodeKind(n.Kind)
		if kind == "" {
			kind = KindNormal
		}
		data.Nodes[n.ID] = Node{
			ID:        n.ID,
			X:         n.X,
			Y:         n.Y,
			Kind:      kind,
			Name:      n.Name,
			GroupID:   n.GroupID,
			IsMastery: n.IsMastery || kind == KindMastery,
			Effects:   n.Effects,
		}
		ids = append(ids, n.ID)
	}

	for _, c := range d.Connections {
		data.Connections = append(data.Connections, Connection{From: c.From, To: c.To})
	}

	if b, ok := data.BoundsOf(ids); ok {
		data.Bounds = b
	}

	return data
}
