// Package importers resolves a user-supplied link or code into a raw Path of
// Building export code. Detection is automatic: the first importer that
// supports the input wins, with the direct-code importer acting as the
// catch-all.
//
// This package is intentionally free of any dependency on the builds domain so
// that the import pipeline stays a leaf: it returns plain strings that the
// caller maps onto domain types.
package importers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrUnsupportedInput is returned when no importer recognizes the input.
var ErrUnsupportedInput = errors.New("importers: unrecognized link or build code")

// Result is the outcome of a successful import.
type Result struct {
	Source string // importer name: "pobbin", "pastebin", or "direct"
	Code   string // raw PoB export code (Base64 text)
	URL    string // canonical source URL; empty for directly pasted codes
}

// Importer resolves one class of input into a PoB export code.
type Importer interface {
	// Name identifies the importer and doubles as the persisted source type.
	Name() string
	// Supports reports whether this importer can handle the input.
	Supports(input string) bool
	// Import fetches and returns the raw PoB code for the input.
	Import(ctx context.Context, input string) (Result, error)
}

// Registry holds the ordered set of importers and dispatches to the first match.
type Registry struct {
	importers []Importer
}

// NewRegistry builds the default registry. Order matters: URL-based importers
// are tried before the direct-code catch-all.
func NewRegistry(client *http.Client) *Registry {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	return &Registry{
		importers: []Importer{
			newPobbinImporter(client),
			newPastebinImporter(client),
			newDirectImporter(),
		},
	}
}

// Detect returns the importer that supports input, or ErrUnsupportedInput.
func (r *Registry) Detect(input string) (Importer, error) {
	for _, imp := range r.importers {
		if imp.Supports(input) {
			return imp, nil
		}
	}

	return nil, ErrUnsupportedInput
}

// Import detects the source and returns the resolved PoB code.
func (r *Registry) Import(ctx context.Context, input string) (Result, error) {
	imp, err := r.Detect(input)
	if err != nil {
		return Result{}, err
	}

	res, err := imp.Import(ctx, input)
	if err != nil {
		return Result{}, fmt.Errorf("importers: %s: %w", imp.Name(), err)
	}

	return res, nil
}
