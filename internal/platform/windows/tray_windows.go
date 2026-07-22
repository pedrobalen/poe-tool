//go:build windows

package windows

import "unsafe"

const trayIconID = 1

// addTrayIcon installs the notification-area icon whose clicks are delivered to
// the hidden window as wmTrayIcon messages.
func (c *Controller) addTrayIcon(hwnd uintptr) {
	nid := notifyIconData{
		HWnd:            hwnd,
		ID:              trayIconID,
		Flags:           nifMessage | nifIcon | nifTip,
		CallbackMessage: wmTrayIcon,
		Icon:            loadIcon(),
	}
	nid.Size = uint32(unsafe.Sizeof(nid))
	copyTip(nid.Tip[:], c.cfg.AppName)

	procShellNotifyIcon.Call(uintptr(nimAdd), uintptr(unsafe.Pointer(&nid)))
}

// removeTrayIcon deletes the notification-area icon.
func removeTrayIcon(hwnd uintptr) {
	nid := notifyIconData{HWnd: hwnd, ID: trayIconID}
	nid.Size = uint32(unsafe.Sizeof(nid))
	procShellNotifyIcon.Call(uintptr(nimDelete), uintptr(unsafe.Pointer(&nid)))
}

// handleTrayMessage reacts to a tray mouse event. The mouse message is packed in
// the low word of lParam.
func (c *Controller) handleTrayMessage(lParam uintptr) {
	switch uint16(lParam & 0xFFFF) {
	case wmLButtonUp:
		c.invokeToggle()
	case wmRButtonUp:
		c.showContextMenu()
	}
}

// showContextMenu displays the tray right-click menu and acts on the selection.
func (c *Controller) showContextMenu() {
	hwnd := c.messageHandle()
	if hwnd == 0 {
		return
	}

	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	procAppendMenu.Call(menu, uintptr(mfString), uintptr(menuShowHide), uintptr(unsafe.Pointer(utf16Ptr("Show / Hide"))))
	procAppendMenu.Call(menu, uintptr(mfString), uintptr(menuExit), uintptr(unsafe.Pointer(utf16Ptr("Exit"))))

	// SetForegroundWindow is required so the menu closes when the user clicks
	// elsewhere (a documented Win32 quirk for tray menus).
	procSetForegroundWindow.Call(hwnd)

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	cmd, _, _ := procTrackPopupMenu.Call(
		menu,
		uintptr(tpmRightButton|tpmReturnCmd),
		uintptr(pt.X), uintptr(pt.Y),
		0, hwnd, 0,
	)
	c.handleMenuCommand(uint16(cmd))
}

func (c *Controller) handleMenuCommand(id uint16) {
	switch id {
	case menuShowHide:
		c.invokeToggle()
	case menuExit:
		if c.cfg.OnQuit != nil {
			c.cfg.OnQuit()
		}
		_ = c.Stop()
	}
}
