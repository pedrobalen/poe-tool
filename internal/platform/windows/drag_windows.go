//go:build windows

package windows

import (
	"time"
	"unsafe"
)

// dragInterval paces the drag loop so the window follows the cursor smoothly
// without busy-spinning.
const dragInterval = 8 * time.Millisecond

// BeginDrag starts following the cursor with the overlay window until EndDrag.
//
// The move runs on its own goroutine, computing the window position from the
// live cursor position and a fixed anchor offset. Doing it here (rather than via
// Gio's HTCAPTION move) avoids the Win32 modal-move loop that swallows the mouse
// release and leaves Gio's pointer capture stuck after the first drag. Running
// SetWindowPos off the UI event loop also avoids the cross-thread ConfigEvent
// deadlock.
func (c *Controller) BeginDrag() {
	hwnd := c.handle()
	if hwnd == 0 {
		return
	}
	if !c.dragging.CompareAndSwap(false, true) {
		return
	}

	go c.dragLoop(hwnd)
}

// EndDrag stops an in-progress window drag.
func (c *Controller) EndDrag() {
	c.dragging.Store(false)
}

func (c *Controller) dragLoop(hwnd uintptr) {
	offX, offY, ok := dragAnchor(hwnd)
	if !ok {
		c.dragging.Store(false)

		return
	}

	for c.dragging.Load() {
		cursor, ok := cursorPos()
		if !ok {
			break
		}
		moveWindow(hwnd, cursor.X+offX, cursor.Y+offY)
		time.Sleep(dragInterval)
	}
}

// dragAnchor records the offset between the window's top-left corner and the
// cursor at the start of the drag, so the window keeps its grab point.
func dragAnchor(hwnd uintptr) (offX, offY int32, ok bool) {
	cursor, ok := cursorPos()
	if !ok {
		return 0, 0, false
	}

	var r rect
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if ret == 0 {
		return 0, 0, false
	}

	return r.Left - cursor.X, r.Top - cursor.Y, true
}

func cursorPos() (point, bool) {
	var p point
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))

	return p, ret != 0
}

func moveWindow(hwnd uintptr, x, y int32) {
	procSetWindowPos.Call(
		hwnd,
		0,
		uintptr(uint32(x)),
		uintptr(uint32(y)),
		0, 0,
		uintptr(swpNoSize|swpNoZOrder|swpNoActivate),
	)
}
