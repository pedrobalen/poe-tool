// Package platform abstracts the OS integration the overlay needs: a global
// hotkey, a system tray icon, and native window styling (borderless, topmost,
// tool window). The concrete implementation is selected per GOOS; the app
// depends only on the Overlay interface.
package platform

import "gioui.org/io/event"

// Config configures the platform integration.
type Config struct {
	// AppName is shown on the tray icon tooltip.
	AppName string
	// OnToggle is invoked when the global hotkey (Ctrl+B) fires or the tray is
	// activated. It runs on an OS thread other than the UI loop, so it must be
	// safe to call window methods that only post messages (Show/Hide/Invalidate).
	OnToggle func()
	// OnQuit is invoked when the user chooses Exit from the tray.
	OnQuit func()
}

// Overlay is the OS integration handle.
type Overlay interface {
	// Start registers the hotkey and creates the tray icon.
	Start() error
	// TryBind inspects a Gio event for the native window handle. It returns true
	// once the handle is captured and overlay styling has been applied.
	TryBind(ev event.Event) bool
	// Toggle shows the overlay when hidden and hides it when shown.
	Toggle()
	// Show makes the overlay visible and brings it to front.
	Show()
	// Hide hides the overlay window.
	Hide()
	// SetOpacity sets the whole-window opacity in the [0,1] range (1 = opaque).
	SetOpacity(alpha float64)
	// BeginDrag starts moving the window with the cursor until EndDrag.
	BeginDrag()
	// EndDrag stops an in-progress window drag.
	EndDrag()
	// Visible reports the current visibility.
	Visible() bool
	// Stop releases the hotkey and removes the tray icon.
	Stop() error
}
