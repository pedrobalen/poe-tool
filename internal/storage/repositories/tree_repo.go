package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pt "github.com/pedrobalen/poe-build-overlay/internal/passive_tree"
)

// TreeRepo persists versioned structural passive tree data.
type TreeRepo struct {
	db *sql.DB
}

// NewTreeRepo returns a TreeRepo backed by db.
func NewTreeRepo(db *sql.DB) *TreeRepo {
	return &TreeRepo{db: db}
}

var _ pt.Store = (*TreeRepo)(nil)

// HasVersion reports whether structural data is stored for a tree version.
func (r *TreeRepo) HasVersion(ctx context.Context, version string) (bool, error) {
	var one int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM passive_tree_versions WHERE version = ?`, version).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("repositories: checking tree version: %w", err)
	}

	return true, nil
}

// LoadTree loads all nodes and connections for a version.
func (r *TreeRepo) LoadTree(ctx context.Context, version string) (*pt.TreeData, error) {
	data := &pt.TreeData{Version: version, Nodes: map[int]pt.Node{}, Connections: []pt.Connection{}}

	if err := r.loadBounds(ctx, data); err != nil {
		return nil, err
	}
	if err := r.loadNodes(ctx, data); err != nil {
		return nil, err
	}
	if err := r.loadConnections(ctx, data); err != nil {
		return nil, err
	}
	if err := r.loadMasteryEffects(ctx, data); err != nil {
		return nil, err
	}

	return data, nil
}

func (r *TreeRepo) loadMasteryEffects(ctx context.Context, data *pt.TreeData) error {
	const q = `SELECT node_id, effect_id, text FROM passive_tree_mastery_effects WHERE version = ?`
	rows, err := r.db.QueryContext(ctx, q, data.Version)
	if err != nil {
		return fmt.Errorf("repositories: loading mastery effects: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			nodeID   int
			effectID int
			text     string
		)
		if err := rows.Scan(&nodeID, &effectID, &text); err != nil {
			return err
		}
		node, ok := data.Nodes[nodeID]
		if !ok {
			continue
		}
		if node.Effects == nil {
			node.Effects = map[int]string{}
		}
		node.Effects[effectID] = text
		data.Nodes[nodeID] = node
	}

	return rows.Err()
}

func (r *TreeRepo) loadBounds(ctx context.Context, data *pt.TreeData) error {
	const q = `SELECT min_x, min_y, max_x, max_y FROM passive_tree_versions WHERE version = ?`
	err := r.db.QueryRowContext(ctx, q, data.Version).Scan(
		&data.Bounds.MinX, &data.Bounds.MinY, &data.Bounds.MaxX, &data.Bounds.MaxY,
	)
	if err != nil {
		return fmt.Errorf("repositories: loading tree bounds: %w", err)
	}

	return nil
}

func (r *TreeRepo) loadNodes(ctx context.Context, data *pt.TreeData) error {
	const q = `SELECT node_id, x, y, kind, name, group_id, is_mastery
		FROM passive_tree_nodes WHERE version = ?`
	rows, err := r.db.QueryContext(ctx, q, data.Version)
	if err != nil {
		return fmt.Errorf("repositories: loading tree nodes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			n       pt.Node
			kind    string
			mastery int
		)
		if err := rows.Scan(&n.ID, &n.X, &n.Y, &kind, &n.Name, &n.GroupID, &mastery); err != nil {
			return err
		}
		n.Kind = pt.NodeKind(kind)
		n.IsMastery = mastery != 0
		data.Nodes[n.ID] = n
	}

	return rows.Err()
}

func (r *TreeRepo) loadConnections(ctx context.Context, data *pt.TreeData) error {
	const q = `SELECT from_node, to_node FROM passive_tree_connections WHERE version = ?`
	rows, err := r.db.QueryContext(ctx, q, data.Version)
	if err != nil {
		return fmt.Errorf("repositories: loading tree connections: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c pt.Connection
		if err := rows.Scan(&c.From, &c.To); err != nil {
			return err
		}
		data.Connections = append(data.Connections, c)
	}

	return rows.Err()
}

// SaveTree persists a version's structural data atomically, replacing any prior
// data for that version.
func (r *TreeRepo) SaveTree(ctx context.Context, data *pt.TreeData) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM passive_tree_versions WHERE version = ?`, data.Version); err != nil {
		return fmt.Errorf("repositories: clearing tree version: %w", err)
	}

	const ver = `INSERT INTO passive_tree_versions (version, min_x, min_y, max_x, max_y, imported_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, ver,
		data.Version, data.Bounds.MinX, data.Bounds.MinY, data.Bounds.MaxX, data.Bounds.MaxY,
		formatTime(time.Now()),
	); err != nil {
		return fmt.Errorf("repositories: inserting tree version: %w", err)
	}

	if err := insertTreeNodes(ctx, tx, data); err != nil {
		return err
	}
	if err := insertTreeConnections(ctx, tx, data); err != nil {
		return err
	}
	if err := insertMasteryEffects(ctx, tx, data); err != nil {
		return err
	}

	return tx.Commit()
}

func insertMasteryEffects(ctx context.Context, tx *sql.Tx, data *pt.TreeData) error {
	const q = `INSERT OR IGNORE INTO passive_tree_mastery_effects (version, node_id, effect_id, text)
		VALUES (?, ?, ?, ?)`
	for _, n := range data.Nodes {
		for effectID, text := range n.Effects {
			if _, err := tx.ExecContext(ctx, q, data.Version, n.ID, effectID, text); err != nil {
				return fmt.Errorf("repositories: inserting mastery effect: %w", err)
			}
		}
	}

	return nil
}

func insertTreeNodes(ctx context.Context, tx *sql.Tx, data *pt.TreeData) error {
	const q = `INSERT INTO passive_tree_nodes
		(version, node_id, x, y, kind, name, group_id, is_mastery)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	for _, n := range data.Nodes {
		if _, err := tx.ExecContext(ctx, q,
			data.Version, n.ID, n.X, n.Y, string(n.Kind), n.Name, n.GroupID, boolToInt(n.IsMastery),
		); err != nil {
			return fmt.Errorf("repositories: inserting tree node: %w", err)
		}
	}

	return nil
}

func insertTreeConnections(ctx context.Context, tx *sql.Tx, data *pt.TreeData) error {
	const q = `INSERT OR IGNORE INTO passive_tree_connections (version, from_node, to_node)
		VALUES (?, ?, ?)`
	for _, c := range data.Connections {
		if _, err := tx.ExecContext(ctx, q, data.Version, c.From, c.To); err != nil {
			return fmt.Errorf("repositories: inserting tree connection: %w", err)
		}
	}

	return nil
}
