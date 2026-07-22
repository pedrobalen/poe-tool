package app

import (
	"errors"

	"github.com/pedrobalen/poe-build-overlay/internal/importers"
)

// friendlyError maps an import failure to a concise, user-facing message,
// preserving the underlying detail for the less-specific cases.
func friendlyError(err error) string {
	switch {
	case errors.Is(err, importers.ErrUnsupportedInput):
		return "Unrecognized input. Paste a pobb.in / Pastebin link or a Path of Building code."
	default:
		return "Could not import the build. " + err.Error()
	}
}
