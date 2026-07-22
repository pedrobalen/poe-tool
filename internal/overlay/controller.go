// Package overlay owns the overlay window's lifecycle and its two views: the
// import prompt and the build progression display. Visibility is delegated to
// the platform integration; this package concerns itself with what is drawn.
package overlay

import "github.com/pedrobalen/poe-build-overlay/internal/platform"

// Controller implements the plan's OverlayController contract by delegating to
// the platform integration, keeping visibility policy in one place.
type Controller struct {
	p platform.Overlay
}

// NewController wraps a platform overlay.
func NewController(p platform.Overlay) *Controller {
	return &Controller{p: p}
}

// ToggleBuildOverlay shows the overlay when hidden and hides it when shown.
func (c *Controller) ToggleBuildOverlay() { c.p.Toggle() }

// Hide hides the overlay.
func (c *Controller) Hide() { c.p.Hide() }

// IsVisible reports whether the overlay is currently visible.
func (c *Controller) IsVisible() bool { return c.p.Visible() }
