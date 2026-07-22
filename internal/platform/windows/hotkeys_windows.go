//go:build windows

package windows

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

const messageClassName = "PoeBuildOverlayMsgWindow"

// runMessageLoop owns a dedicated OS thread that hosts a hidden message-only
// window. That window receives the global hotkey and tray callbacks and pumps
// messages until it is destroyed. Locking the thread is required: Win32 ties
// message queues and window ownership to the creating thread.
func (c *Controller) runMessageLoop(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hwnd, err := c.createMessageWindow()
	if err != nil {
		ready <- err

		return
	}
	c.setMessageHandle(hwnd)

	if err := registerToggleHotkey(hwnd); err != nil {
		procDestroyWindow.Call(hwnd)
		ready <- err

		return
	}

	c.addTrayIcon(hwnd)
	ready <- nil

	c.pumpMessages()
}

func (c *Controller) createMessageWindow() (uintptr, error) {
	instance := getModuleHandle()
	className := utf16Ptr(messageClassName)

	wc := wndClassEx{
		Style:     0,
		WndProc:   syscall.NewCallback(c.wndProc),
		Instance:  instance,
		Icon:      loadIcon(),
		Cursor:    loadCursor(),
		ClassName: className,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))

	if ret, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); ret == 0 {
		return 0, fmt.Errorf("windows: registering window class: %w", err)
	}

	hwnd, _, err := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr(messageClassName))),
		0,
		uintptr(cwUseDefault), uintptr(cwUseDefault), 0, 0,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("windows: creating message window: %w", err)
	}

	return hwnd, nil
}

func registerToggleHotkey(hwnd uintptr) error {
	ret, _, err := procRegisterHotKey.Call(hwnd, 1, uintptr(modControl|modNoRepeat), uintptr(vkB))
	if ret == 0 {
		return fmt.Errorf("windows: registering Ctrl+B hotkey (already in use?): %w", err)
	}

	return nil
}

// wndProc handles the small set of messages the hidden window cares about.
func (c *Controller) wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmHotkey:
		c.invokeToggle()

		return 0
	case wmTrayIcon:
		c.handleTrayMessage(lParam)

		return 0
	case wmCommand:
		c.handleMenuCommand(loWord(wParam))

		return 0
	case wmDestroy:
		removeTrayIcon(hwnd)
		procUnregisterHotKey.Call(hwnd, 1)
		procPostQuitMessage.Call(0)

		return 0
	default:
		ret, _, _ := procDefWindowProc.Call(hwnd, message, wParam, lParam)

		return ret
	}
}

func (c *Controller) pumpMessages() {
	var m msg
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		// GetMessage returns 0 on WM_QUIT and -1 (as ^uintptr(0)) on error.
		if ret == 0 || ret == ^uintptr(0) {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func (c *Controller) invokeToggle() {
	if c.cfg.OnToggle != nil {
		c.cfg.OnToggle()

		return
	}
	c.Toggle()
}

func loWord(v uintptr) uint16 {
	return uint16(v & 0xFFFF)
}
