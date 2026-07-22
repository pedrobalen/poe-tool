package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// WindowState is the persisted overlay geometry.
type WindowState struct {
	X         int
	Y         int
	Width     int
	Height    int
	Maximized bool
}

// DefaultWindowState is used before the user has positioned the overlay.
var DefaultWindowState = WindowState{X: 0, Y: 0, Width: 900, Height: 640}

// WindowRepo persists the single overlay window geometry row.
type WindowRepo struct {
	db *sql.DB
}

// NewWindowRepo returns a WindowRepo backed by db.
func NewWindowRepo(db *sql.DB) *WindowRepo {
	return &WindowRepo{db: db}
}

// Load returns the stored window state, or DefaultWindowState when unset.
func (r *WindowRepo) Load(ctx context.Context) (WindowState, error) {
	const q = `SELECT x, y, width, height, maximized FROM window_state WHERE id = 1`

	var (
		s         WindowState
		maximized int
	)
	err := r.db.QueryRowContext(ctx, q).Scan(&s.X, &s.Y, &s.Width, &s.Height, &maximized)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultWindowState, nil
	}
	if err != nil {
		return WindowState{}, fmt.Errorf("repositories: loading window state: %w", err)
	}
	s.Maximized = maximized != 0

	return s, nil
}

// Save writes the window state to the singleton row.
func (r *WindowRepo) Save(ctx context.Context, s WindowState) error {
	const q = `INSERT INTO window_state (id, x, y, width, height, maximized)
		VALUES (1, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			x=excluded.x, y=excluded.y, width=excluded.width,
			height=excluded.height, maximized=excluded.maximized`
	if _, err := r.db.ExecContext(ctx, q, s.X, s.Y, s.Width, s.Height, boolToInt(s.Maximized)); err != nil {
		return fmt.Errorf("repositories: saving window state: %w", err)
	}

	return nil
}
