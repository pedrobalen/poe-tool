package passive_tree

import (
	"context"
	"errors"
	"fmt"
)

// ErrTreeUnavailable indicates that no structural data is stored or importable
// for a requested tree version. The UI surfaces this rather than drawing a tree
// that does not match the build's version.
var ErrTreeUnavailable = errors.New("passive_tree: structural data unavailable for version")

// Store persists and retrieves versioned structural tree data. Implemented by
// the storage layer.
type Store interface {
	HasVersion(ctx context.Context, version string) (bool, error)
	LoadTree(ctx context.Context, version string) (*TreeData, error)
	SaveTree(ctx context.Context, data *TreeData) error
}

// Source imports structural data for a version from an external form (e.g. a
// bundled JSON asset) when the store does not yet have it.
type Source interface {
	Available(version string) bool
	Import(version string) (*TreeData, error)
}

// MultiSource tries each source in order, using the first that has the version.
// It lets bundled data take precedence over user-supplied files (or vice versa).
type MultiSource struct {
	sources []Source
}

// NewMultiSource chains sources; earlier sources take precedence.
func NewMultiSource(sources ...Source) *MultiSource {
	return &MultiSource{sources: sources}
}

// Available reports whether any chained source has the version.
func (m *MultiSource) Available(version string) bool {
	return m.pick(version) != nil
}

// Import imports the version from the first source that has it.
func (m *MultiSource) Import(version string) (*TreeData, error) {
	src := m.pick(version)
	if src == nil {
		return nil, fmt.Errorf("%w: %s", ErrTreeUnavailable, version)
	}

	return src.Import(version)
}

func (m *MultiSource) pick(version string) Source {
	for _, s := range m.sources {
		if s != nil && s.Available(version) {
			return s
		}
	}

	return nil
}

// Loader resolves structural tree data, preferring the persisted store and
// falling back to a Source (which it then caches into the store). Both the
// store and source are optional; a Loader with neither always reports
// ErrTreeUnavailable.
type Loader struct {
	store  Store
	source Source
}

// NewLoader wires a Loader. store and source may each be nil.
func NewLoader(store Store, source Source) *Loader {
	return &Loader{store: store, source: source}
}

// Load returns the structural data for version, importing and caching it from
// the source when the store lacks it.
func (l *Loader) Load(ctx context.Context, version string) (*TreeData, error) {
	if version == "" {
		return nil, ErrTreeUnavailable
	}

	if l.store != nil {
		has, err := l.store.HasVersion(ctx, version)
		if err != nil {
			return nil, err
		}
		if has {
			return l.store.LoadTree(ctx, version)
		}
	}

	if l.source == nil || !l.source.Available(version) {
		return nil, fmt.Errorf("%w: %s", ErrTreeUnavailable, version)
	}

	data, err := l.source.Import(version)
	if err != nil {
		return nil, err
	}
	if l.store != nil {
		if err := l.store.SaveTree(ctx, data); err != nil {
			return nil, err
		}
	}

	return data, nil
}
