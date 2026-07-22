//go:build !windows

package platform

import "gioui.org/io/event"

// New returns a no-op overlay integration on non-Windows platforms. The plan
// targets Windows first; other platforms build but provide no OS integration.
func New(_ Config) Overlay {
	return noopOverlay{}
}

type noopOverlay struct{}

func (noopOverlay) Start() error               { return nil }
func (noopOverlay) TryBind(_ event.Event) bool { return false }
func (noopOverlay) Toggle()                    {}
func (noopOverlay) Show()                      {}
func (noopOverlay) Hide()                      {}
func (noopOverlay) SetOpacity(_ float64)       {}
func (noopOverlay) BeginDrag()                 {}
func (noopOverlay) EndDrag()                   {}
func (noopOverlay) Visible() bool              { return true }
func (noopOverlay) Stop() error                { return nil }
