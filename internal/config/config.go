// Package config resolves filesystem locations and holds process-wide static
// configuration. Mutable, persisted settings live in SQLite (see the storage
// layer); this package only decides where the database and logs live.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// AppName is the application's identifier, used for the data directory name.
const AppName = "poe-build-overlay"

// Config holds resolved, read-only paths for the running process.
type Config struct {
	// DataDir is the per-user directory holding the database and logs.
	DataDir string
	// DatabasePath is the SQLite database file path.
	DatabasePath string
	// LogPath is the rotating log file path.
	LogPath string
}

// Load resolves the application's data directory (creating it if needed) and
// derives the database and log paths within it.
func Load() (Config, error) {
	base, err := userDataDir()
	if err != nil {
		return Config{}, err
	}

	dataDir := filepath.Join(base, AppName)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return Config{}, fmt.Errorf("config: creating data dir: %w", err)
	}

	return Config{
		DataDir:      dataDir,
		DatabasePath: filepath.Join(dataDir, "overlay.db"),
		LogPath:      filepath.Join(dataDir, "overlay.log"),
	}, nil
}

// userDataDir returns the platform's per-user application data directory.
func userDataDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: resolving user config dir: %w", err)
	}

	return dir, nil
}
