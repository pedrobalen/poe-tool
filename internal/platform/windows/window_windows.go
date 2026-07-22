//go:build windows

package windows

// applyOverlayStyle turns the Gio window into a topmost tool window:
// WS_EX_TOOLWINDOW keeps it off the taskbar, and WS_EX_TOPMOST pins it above
// windowed/borderless game clients.
//
// WS_EX_NOACTIVATE is deliberately NOT applied: a borderless window that never
// activates cannot receive keyboard focus (so the import field is untypable)
// and cannot be moved or dismissed, which reads as a frozen window.
func applyOverlayStyle(hwnd uintptr) {
	ex, _, _ := procGetWindowLongPtr.Call(hwnd, uintptr(gwlExStyle))
	ex |= wsExTopmost | wsExToolWindow | wsExLayered
	procSetWindowLongPtr.Call(hwnd, uintptr(gwlExStyle), ex)

	// A layered window stays blank until its alpha is set; start fully opaque so
	// the overlay is visible before the user adjusts opacity.
	setLayeredAlpha(hwnd, 255)

	pinTopmost(hwnd)
}

// setLayeredAlpha sets the whole-window alpha (0 transparent, 255 opaque).
func setLayeredAlpha(hwnd uintptr, alpha byte) {
	procSetLayeredWindowAttributes.Call(hwnd, 0, uintptr(alpha), uintptr(lwaAlpha))
}

// pinTopmost re-asserts topmost z-order without moving or resizing.
func pinTopmost(hwnd uintptr) {
	procSetWindowPos.Call(
		hwnd,
		hwndTopmost,
		0, 0, 0, 0,
		uintptr(swpNoMove|swpNoSize|swpNoActivate),
	)
}

// showWindow makes the overlay visible, activates it so its text field can take
// keyboard focus, and keeps it pinned on top.
func showWindow(hwnd uintptr) {
	procShowWindow.Call(hwnd, uintptr(swShow))
	procSetForegroundWindow.Call(hwnd)
	pinTopmost(hwnd)
}

func hideWindow(hwnd uintptr) {
	procShowWindow.Call(hwnd, uintptr(swHide))
}
