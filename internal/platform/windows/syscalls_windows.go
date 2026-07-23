//go:build windows

// Package windows implements the platform.Overlay contract using Win32: a
// global hotkey, a tray icon, and topmost tool-window styling applied to the
// Gio-owned HWND.
package windows

import "syscall"

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterHotKey      = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey    = user32.NewProc("UnregisterHotKey")
	procGetMessage          = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessage     = user32.NewProc("DispatchMessageW")
	procDefWindowProc       = user32.NewProc("DefWindowProcW")
	procRegisterClassEx     = user32.NewProc("RegisterClassExW")
	procCreateWindowEx      = user32.NewProc("CreateWindowExW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procSetWindowLongPtr    = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtr    = user32.NewProc("GetWindowLongPtrW")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procShowWindow          = user32.NewProc("ShowWindow")
	procLoadIcon            = user32.NewProc("LoadIconW")
	procLoadCursor          = user32.NewProc("LoadCursorW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenu          = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procPostMessage         = user32.NewProc("PostMessageW")

	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")

	procShellNotifyIcon = shell32.NewProc("Shell_NotifyIconW")

	procGetModuleHandle = kernel32.NewProc("GetModuleHandleW")
)

// gwlExStyle is the GWL_EXSTYLE index (-20). It is a typed variable so the
// negative value converts to uintptr at runtime instead of overflowing as an
// untyped constant.
var gwlExStyle int32 = -20

// Window styles and messages.
const (
	wsExTopmost    = 0x00000008
	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000
	wsExLayered    = 0x00080000

	lwaAlpha = 0x00000002

	swHide           = 0
	swShowNoActivate = 4
	swShow           = 5

	hwndTopmost = ^uintptr(0) // (HWND)-1

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010
	swpShowWindow = 0x0040

	modControl  = 0x0002
	modNoRepeat = 0x4000
	vkB         = 0x42
	vkLButton   = 0x01

	wmDestroy   = 0x0002
	wmClose     = 0x0010
	wmHotkey    = 0x0312
	wmApp       = 0x8000
	wmTrayIcon  = wmApp + 1
	wmCommand   = 0x0111
	wmLButtonUp = 0x0202
	wmRButtonUp = 0x0205

	idiApplication = 32512
	idcArrow       = 32512

	// appIconID is the RT_GROUP_ICON resource id embedded by goversioninfo (see
	// versioninfo.json). It is absent when building without the generated .syso
	// (e.g. a bare `go run`), so loadIcon falls back to the system icon.
	appIconID = 1

	menuShowHide = 1
	menuExit     = 2

	mfString = 0x0000

	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100

	nimAdd    = 0x0000
	nimModify = 0x0001
	nimDelete = 0x0002

	nifMessage = 0x0001
	nifIcon    = 0x0002
	nifTip     = 0x0004

	cwUseDefault = ^uint32(0x7fffffff) // 0x80000000, CW_USEDEFAULT
)

// point mirrors the Win32 POINT struct.
type point struct {
	X int32
	Y int32
}

// rect mirrors the Win32 RECT struct.
type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// msg mirrors the Win32 MSG struct.
type msg struct {
	HWND    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

// wndClassEx mirrors the Win32 WNDCLASSEXW struct.
type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

// notifyIconData mirrors the fields of NOTIFYICONDATAW the tray icon uses.
type notifyIconData struct {
	Size            uint32
	HWnd            uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Timeout         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
}

func getModuleHandle() uintptr {
	h, _, _ := procGetModuleHandle.Call(0)

	return h
}

// loadIcon returns the application icon embedded in the executable (resource
// appIconID), falling back to the generic system application icon when that
// resource is absent.
func loadIcon() uintptr {
	if h, _, _ := procLoadIcon.Call(getModuleHandle(), uintptr(appIconID)); h != 0 {
		return h
	}

	h, _, _ := procLoadIcon.Call(0, uintptr(idiApplication))

	return h
}

func loadCursor() uintptr {
	h, _, _ := procLoadCursor.Call(0, uintptr(idcArrow))

	return h
}

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return nil
	}

	return p
}

// copyTip writes s into a fixed-size UTF-16 tooltip buffer, truncating safely.
func copyTip(dst []uint16, s string) {
	encoded := syscall.StringToUTF16(s)
	n := len(encoded)
	if n > len(dst) {
		n = len(dst)
	}
	copy(dst[:n], encoded[:n])
	dst[len(dst)-1] = 0
}
