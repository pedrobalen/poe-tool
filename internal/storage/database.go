// Package storage owns the SQLite database: connection setup, versioned schema
// migrations, and repository implementations for the domain contracts.
//
// The pure-Go modernc.org/sqlite driver is used deliberately so the application
// builds without cgo and a C toolchain.
package storage

import (
	"database/sql"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite"
)

// DB wraps *sql.DB with the pragmas and lifecycle the application relies on.
type DB struct {
	*sql.DB
}

// Open connects to the SQLite database at path, applying pragmas that suit a
// single-process desktop app: WAL for concurrent reads during writes, enforced
// foreign keys, and a busy timeout so brief lock contention retries instead of
// failing.
func Open(path string) (*DB, error) {
	dsn := buildDSN(path)

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: opening database: %w", err)
	}

	// A desktop app has one writer; capping connections avoids WAL writer
	// contention and keeps memory predictable.
	sqlDB.SetMaxOpenConns(1)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()

		return nil, fmt.Errorf("storage: pinging database: %w", err)
	}

	return &DB{DB: sqlDB}, nil
}

func buildDSN(path string) string {
	params := url.Values{}
	params.Add("_pragma", "busy_timeout(5000)")
	params.Add("_pragma", "journal_mode(WAL)")
	params.Add("_pragma", "foreign_keys(1)")
	params.Add("_pragma", "synchronous(NORMAL)")

	return "file:" + path + "?" + params.Encode()
}
