//go:build windows

package platform

import "github.com/pedrobalen/poe-build-overlay/internal/platform/windows"

// New returns the Windows overlay integration.
func New(cfg Config) Overlay {
	return windows.New(windows.Config{
		AppName:  cfg.AppName,
		OnToggle: cfg.OnToggle,
		OnQuit:   cfg.OnQuit,
	})
}
