//go:build windows

package windows

var procGetSystemMetrics = user32.NewProc("GetSystemMetrics")

const (
	smCXScreen = 0
	smCYScreen = 1
)

// PrimaryScreenSize returns the primary monitor's pixel dimensions. It is
// exposed for placing the overlay on first run before persisted geometry exists.
func PrimaryScreenSize() (width, height int) {
	w, _, _ := procGetSystemMetrics.Call(uintptr(smCXScreen))
	h, _, _ := procGetSystemMetrics.Call(uintptr(smCYScreen))

	return int(w), int(h)
}
