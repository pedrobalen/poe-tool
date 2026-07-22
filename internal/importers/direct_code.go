package importers

import (
	"context"
	"errors"
	"strings"
)

// minDirectCodeLen is a lower bound below which an input is too short to be a
// plausible PoB export code. It only gates detection; decoding validates fully.
const minDirectCodeLen = 32

// directImporter is the catch-all: it treats the input as a raw PoB code
// pasted directly. It supports any non-URL input that looks like Base64.
type directImporter struct{}

func newDirectImporter() *directImporter { return &directImporter{} }

func (d *directImporter) Name() string { return "direct" }

func (d *directImporter) Supports(input string) bool {
	input = strings.TrimSpace(input)
	if len(input) < minDirectCodeLen {
		return false
	}
	if looksLikeURL(input) {
		return false
	}

	return isBase64ish(input)
}

func (d *directImporter) Import(_ context.Context, input string) (Result, error) {
	code := strings.TrimSpace(input)
	if code == "" {
		return Result{}, errors.New("empty build code")
	}

	return Result{Source: d.Name(), Code: code, URL: ""}, nil
}

// isBase64ish reports whether every non-whitespace rune belongs to the union of
// the standard and URL-safe Base64 alphabets. It is a cheap gate, not a
// validator: pob.Decode is the source of truth.
func isBase64ish(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '+' || r == '/' || r == '-' || r == '_' || r == '=':
		case r == '\n' || r == '\r' || r == '\t' || r == ' ':
		default:
			return false
		}
	}

	return true
}
