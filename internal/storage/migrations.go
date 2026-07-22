package storage

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// migration is one ordered schema change identified by its numeric filename
// prefix (e.g. "0001_init.sql" -> version 1).
type migration struct {
	version int
	name    string
	sql     string
}

// Migrate applies every pending migration inside a transaction, recording each
// in schema_migrations so it runs at most once.
func (db *DB) Migrate(ctx context.Context) error {
	if err := db.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	applied, err := db.appliedVersions(ctx)
	if err != nil {
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		if err := db.applyMigration(ctx, m); err != nil {
			return fmt.Errorf("storage: applying migration %s: %w", m.name, err)
		}
	}

	return nil
}

func (db *DB) ensureMigrationsTable(ctx context.Context) error {
	const q = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name    TEXT NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`
	if _, err := db.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("storage: creating schema_migrations: %w", err)
	}

	return nil
}

func (db *DB) appliedVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("storage: reading applied migrations: %w", err)
	}
	defer rows.Close()

	applied := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}

	return applied, rows.Err()
}

func (db *DB) applyMigration(ctx context.Context, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return err
	}

	const record = `INSERT INTO schema_migrations (version, name) VALUES (?, ?)`
	if _, err := tx.ExecContext(ctx, record, m.version, m.name); err != nil {
		return err
	}

	return tx.Commit()
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("storage: reading embedded migrations: %w", err)
	}

	migrations := make([]migration, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}

		version, err := versionFromName(e.Name())
		if err != nil {
			return nil, err
		}

		content, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, err
		}

		migrations = append(migrations, migration{
			version: version,
			name:    e.Name(),
			sql:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

func versionFromName(name string) (int, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("storage: migration %q lacks a numeric prefix", name)
	}

	var version int
	if _, err := fmt.Sscanf(prefix, "%d", &version); err != nil {
		return 0, fmt.Errorf("storage: migration %q has an invalid version: %w", name, err)
	}

	return version, nil
}
