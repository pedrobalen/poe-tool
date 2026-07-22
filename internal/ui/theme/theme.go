// Package theme centralizes the overlay's visual language: a dark, low-contrast
// palette suited to sitting on top of the game, plus the semantic colors used to
// distinguish passive-tree node states.
package theme

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// Theme extends the material theme with overlay-specific semantic colors.
type Theme struct {
	*material.Theme

	// Surface is the overlay panel background.
	Surface color.NRGBA
	// Muted is used for previous-stage nodes and secondary text.
	Muted color.NRGBA
	// New highlights nodes and gems added in the current stage.
	New color.NRGBA
	// Future is a secondary highlight for upcoming nodes.
	Future color.NRGBA
	// Removed marks nodes and gems dropped in the current stage.
	Removed color.NRGBA
	// Line is the color of active tree connections.
	Line color.NRGBA
}

// New builds the overlay theme with a loaded font shaper. material.NewTheme
// ships an empty shaper, so fonts must be attached explicitly or nothing draws.
func New() *Theme {
	base := material.NewTheme()
	base.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	base.TextSize = unit.Sp(15)
	base.Palette = material.Palette{
		Fg:         rgb(0xE6E6E6),
		Bg:         rgb(0x14161B),
		ContrastBg: rgb(0xD4A24C),
		ContrastFg: rgb(0x14161B),
	}

	return &Theme{
		Theme:   base,
		Surface: rgba(0x1C1F27, 0xF2),
		Muted:   rgb(0x6E7480),
		New:     rgb(0x5FD08A),
		Future:  rgb(0x4C86D4),
		Removed: rgb(0xD05F5F),
		Line:    rgb(0xD4A24C),
	}
}

func rgb(c uint32) color.NRGBA {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 0xFF}
}

func rgba(c uint32, a uint8) color.NRGBA {
	n := rgb(c)
	n.A = a

	return n
}
