// Package pobfiles discovers builds saved locally by the Path of Building
// desktop app. PoB stores each build as a plain (un-encoded) XML file under a
// Builds directory, so they can be read and parsed directly.
package pobfiles

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrDirNotFound indicates no Path of Building Builds directory was located.
var ErrDirNotFound = errors.New("pobfiles: Path of Building Builds directory not found")

// Build is a locally saved Path of Building build file.
type Build struct {
	Name string // display name derived from the path relative to the Builds dir
	Path string
}

// candidateDirs returns the directories where PoB commonly stores builds,
// accounting for the OneDrive-redirected Documents folder.
func candidateDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	const rel = "Path of Building/Builds"

	return []string{
		filepath.Join(home, "Documents", rel),
		filepath.Join(home, "OneDrive", "Documents", rel),
	}
}

// FindDir returns the first existing PoB Builds directory.
func FindDir() (string, bool) {
	for _, dir := range candidateDirs() {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, true
		}
	}

	return "", false
}

// List returns the build files found under the PoB Builds directory, sorted by
// name. It returns ErrDirNotFound when no directory exists.
func List() ([]Build, error) {
	dir, ok := FindDir()
	if !ok {
		return nil, ErrDirNotFound
	}

	return listDir(dir)
}

func listDir(dir string) ([]Build, error) {
	builds := []Build{}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries rather than aborting the walk
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".xml") {
			return nil
		}

		builds = append(builds, Build{
			Name: displayName(dir, path),
			Path: path,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(builds, func(i, j int) bool { return builds[i].Name < builds[j].Name })

	return builds, nil
}

// displayName turns a build file path into a readable name relative to the
// Builds root, e.g. "Duelist/Static Strike Slayer".
func displayName(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = filepath.Base(path)
	}
	rel = strings.TrimSuffix(rel, filepath.Ext(rel))

	return filepath.ToSlash(rel)
}
