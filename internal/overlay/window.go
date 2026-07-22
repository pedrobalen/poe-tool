package overlay

import (
	"gioui.org/app"
	"gioui.org/unit"

	"github.com/pedrobalen/poe-build-overlay/internal/storage/repositories"
)

// WindowTitle is the overlay window title (hidden by the tool-window styling but
// used by the OS internally).
const WindowTitle = "PoE Build Progression Overlay"

// Options builds the Gio window options for the overlay from persisted geometry.
// The window is undecorated; native topmost/tool-window styling is applied later
// once the handle is known.
func Options(state repositories.WindowState) []app.Option {
	width := state.Width
	if width <= 0 {
		width = repositories.DefaultWindowState.Width
	}
	height := state.Height
	if height <= 0 {
		height = repositories.DefaultWindowState.Height
	}

	return []app.Option{
		app.Title(WindowTitle),
		app.Size(unit.Dp(width), unit.Dp(height)),
		app.MinSize(unit.Dp(480), unit.Dp(360)),
		app.Decorated(false),
	}
}
