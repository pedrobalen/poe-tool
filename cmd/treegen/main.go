// Command treegen converts an official Grinding Gear Games passive skill tree
// export (data.json from github.com/grindinggear/skilltree-export) into the
// compact "<version>.json" format the overlay loads, computing each node's X/Y
// position from its group, orbit, and orbit index.
//
// Usage:
//
//	go run ./cmd/treegen -data path/to/data.json -version 3_29
//
// With no -out, the file is written to the overlay's data directory
// (%AppData%/poe-build-overlay/tree on Windows).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	dataPath := flag.String("data", "", "path to the GGG export data.json (required)")
	version := flag.String("version", "", "tree version name for the output file, e.g. 3_29 (required)")
	outDir := flag.String("out", "", "output directory (default: overlay data dir)")
	flag.Parse()

	if *dataPath == "" || *version == "" {
		flag.Usage()
		os.Exit(2)
	}

	dir, err := resolveOutDir(*outDir)
	if err != nil {
		log.Fatalf("resolving output dir: %v", err)
	}

	src, err := loadExport(*dataPath)
	if err != nil {
		log.Fatalf("loading export: %v", err)
	}

	tree := convert(src)

	outPath := filepath.Join(dir, *version+".json")
	if err := writeTree(outPath, tree); err != nil {
		log.Fatalf("writing tree: %v", err)
	}

	fmt.Printf("wrote %d nodes and %d connections to %s\n", len(tree.Nodes), len(tree.Connections), outPath)
}

func resolveOutDir(outDir string) (string, error) {
	if outDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		outDir = filepath.Join(base, "poe-build-overlay", "tree")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}

	return outDir, nil
}

func loadExport(path string) (*export, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var e export
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, err
	}

	return &e, nil
}

// convert projects every positioned node into the overlay's tree format and
// collects the deduplicated set of connections between included nodes.
func convert(e *export) outTree {
	nodes := make([]outNode, 0, len(e.Nodes))
	included := make(map[int]struct{}, len(e.Nodes))

	for key, n := range e.Nodes {
		id, err := strconv.Atoi(key)
		if err != nil || n.Group == nil || n.IsProxy || n.AscendancyName != "" {
			// Ascendancy clusters are positioned far from where they attach to
			// the main tree in the raw export (Path of Building repositions them
			// dynamically near the ascendancy start). Rendering their straight
			// connections would streak lines across the whole tree, so the
			// overlay shows the main passive tree only.
			continue
		}
		group, ok := e.Groups[strconv.Itoa(*n.Group)]
		if !ok {
			continue
		}

		x, y := position(group, n, e.Constants)
		nodes = append(nodes, outNode{
			ID:      id,
			X:       float32(x),
			Y:       float32(y),
			Kind:    kindOf(n),
			Name:    n.Name,
			Group:   *n.Group,
			Mastery: n.IsMastery,
			Effects: masteryEffects(n),
		})
		included[id] = struct{}{}
	}

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	return outTree{Nodes: nodes, Connections: connections(e, included)}
}

// connections gathers edges from each node's out/in lists, keeping only edges
// whose endpoints were both included and deduplicating undirected pairs.
func connections(e *export, included map[int]struct{}) []outConn {
	seen := make(map[[2]int]struct{})
	conns := make([]outConn, 0)

	add := func(a, b int) {
		if a == b {
			return
		}
		if _, ok := included[a]; !ok {
			return
		}
		if _, ok := included[b]; !ok {
			return
		}
		key := [2]int{a, b}
		if a > b {
			key = [2]int{b, a}
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		conns = append(conns, outConn{From: key[0], To: key[1]})
	}

	for key, n := range e.Nodes {
		id, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		for _, other := range n.Out {
			if o, err := strconv.Atoi(other); err == nil {
				add(id, o)
			}
		}
		for _, other := range n.In {
			if o, err := strconv.Atoi(other); err == nil {
				add(id, o)
			}
		}
	}

	sort.Slice(conns, func(i, j int) bool {
		if conns[i].From != conns[j].From {
			return conns[i].From < conns[j].From
		}

		return conns[i].To < conns[j].To
	})

	return conns
}

// position computes a node's world coordinates. Nodes are placed on concentric
// orbits around their group centre; orbit 0 sits at the centre. The 16-node
// orbits use GGG's non-uniform angle table (see the export README).
func position(group exportGroup, n exportNode, c exportConstants) (float64, float64) {
	if n.Orbit < 0 || n.Orbit >= len(c.OrbitRadii) {
		return group.X, group.Y
	}
	radius := c.OrbitRadii[n.Orbit]
	angle := orbitAngle(n.Orbit, n.OrbitIndex, c.SkillsPerOrbit)

	return group.X + radius*math.Sin(angle), group.Y - radius*math.Cos(angle)
}

// angles16 is GGG's angle table (degrees) for the 16-node orbits (2 and 3),
// where positions are not evenly spaced.
var angles16 = []float64{0, 30, 45, 60, 90, 120, 135, 150, 180, 210, 225, 240, 270, 300, 315, 330}

func orbitAngle(orbit, orbitIndex int, skillsPerOrbit []int) float64 {
	if orbit >= len(skillsPerOrbit) {
		return 0
	}
	count := skillsPerOrbit[orbit]
	if count == 16 && orbitIndex >= 0 && orbitIndex < len(angles16) {
		return angles16[orbitIndex] * math.Pi / 180
	}
	if count == 0 {
		return 0
	}

	return float64(orbitIndex) * 2 * math.Pi / float64(count)
}

// masteryEffects maps a mastery node's effect ids to their combined stat text.
// It returns nil for non-mastery nodes so the field is omitted from the output.
func masteryEffects(n exportNode) map[int]string {
	if len(n.MasteryEffects) == 0 {
		return nil
	}

	effects := make(map[int]string, len(n.MasteryEffects))
	for _, e := range n.MasteryEffects {
		effects[e.Effect] = strings.Join(e.Stats, "\n")
	}

	return effects
}

func kindOf(n exportNode) string {
	switch {
	case n.IsKeystone:
		return "keystone"
	case n.IsMastery:
		return "mastery"
	case n.IsNotable:
		return "notable"
	default:
		return "normal"
	}
}

func writeTree(path string, tree outTree) error {
	raw, err := json.Marshal(tree)
	if err != nil {
		return err
	}

	return os.WriteFile(path, raw, 0o644)
}

// export mirrors the subset of the GGG data.json this tool reads.
type export struct {
	Groups    map[string]exportGroup `json:"groups"`
	Nodes     map[string]exportNode  `json:"nodes"`
	Constants exportConstants        `json:"constants"`
}

type exportGroup struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type exportNode struct {
	Name           string   `json:"name"`
	AscendancyName string   `json:"ascendancyName"`
	Group          *int     `json:"group"`
	Orbit          int      `json:"orbit"`
	OrbitIndex     int      `json:"orbitIndex"`
	IsNotable      bool     `json:"isNotable"`
	IsKeystone     bool     `json:"isKeystone"`
	IsMastery      bool     `json:"isMastery"`
	IsProxy        bool     `json:"isProxy"`
	Out            []string `json:"out"`
	In             []string `json:"in"`
	MasteryEffects []struct {
		Effect int      `json:"effect"`
		Stats  []string `json:"stats"`
	} `json:"masteryEffects"`
}

type exportConstants struct {
	SkillsPerOrbit []int     `json:"skillsPerOrbit"`
	OrbitRadii     []float64 `json:"orbitRadii"`
}

// outTree / outNode / outConn match internal/passive_tree.jsonTree.
type outTree struct {
	Nodes       []outNode `json:"nodes"`
	Connections []outConn `json:"connections"`
}

type outNode struct {
	ID      int            `json:"id"`
	X       float32        `json:"x"`
	Y       float32        `json:"y"`
	Kind    string         `json:"kind"`
	Name    string         `json:"name"`
	Group   int            `json:"group"`
	Mastery bool           `json:"mastery"`
	Effects map[int]string `json:"effects,omitempty"`
}

type outConn struct {
	From int `json:"from"`
	To   int `json:"to"`
}
