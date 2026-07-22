package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SettingsRepo is a typed key/value store over app_settings for small scalar
// application settings.
type SettingsRepo struct {
	db *sql.DB
}

// NewSettingsRepo returns a SettingsRepo backed by db.
func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

// Get returns the value for key and whether it was present.
func (r *SettingsRepo) Get(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("repositories: reading setting %q: %w", key, err)
	}

	return value, true, nil
}

// Set writes value for key, inserting or updating as needed.
func (r *SettingsRepo) Set(ctx context.Context, key, value string) error {
	const q = `INSERT INTO app_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	if _, err := r.db.ExecContext(ctx, q, key, value); err != nil {
		return fmt.Errorf("repositories: writing setting %q: %w", key, err)
	}

	return nil
}
