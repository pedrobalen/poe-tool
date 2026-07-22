//go:build windows

package windows

import (
	"sync"
	"sync/atomic"

	"gioui.org/app"
	"gioui.org/io/event"
)

// Config configures the Windows overlay integration.
type Config struct {
	AppName  string
	OnToggle func()
	OnQuit   func()
}

// Controller implements platform.Overlay on Windows.
type Controller struct {
	cfg Config

	mu      sync.Mutex
	hwnd    uintptr // Gio window handle, captured via TryBind
	msgHWND uintptr // hidden message-only window owning the hotkey and tray
	visible bool
	bound   bool

	started  bool
	dragging atomic.Bool
}

// New constructs a Controller.
func New(cfg Config) *Controller {
	return &Controller{cfg: cfg}
}

// Start creates the hidden message window (which registers the hotkey and tray
// icon) on its own OS thread and begins pumping messages.
func (c *Controller) Start() error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()

		return nil
	}
	c.started = true
	c.mu.Unlock()

	ready := make(chan error, 1)
	go c.runMessageLoop(ready)

	return <-ready
}

// TryBind captures the native window handle from a Gio ViewEvent and applies the
// overlay styling once. It returns true on the frame the handle is captured.
func (c *Controller) TryBind(ev event.Event) bool {
	ve, ok := ev.(app.Win32ViewEvent)
	if !ok || ve.HWND == 0 {
		return false
	}

	c.mu.Lock()
	already := c.bound
	c.hwnd = ve.HWND
	c.bound = true
	c.mu.Unlock()

	if already {
		return false
	}

	// Apply styling and the initial hide on a separate goroutine, never on the
	// caller's event loop. Manipulating the Gio-owned HWND (SetWindowPos /
	// SetWindowLong / ShowWindow) makes Gio's window thread emit a ConfigEvent
	// synchronously; if the event-consuming goroutine were blocked inside these
	// Win32 calls, the two threads would deadlock and the window would freeze.
	go func(hwnd uintptr) {
		applyOverlayStyle(hwnd)
		c.Hide()
	}(ve.HWND)

	return true
}

// Toggle flips visibility.
func (c *Controller) Toggle() {
	if c.Visible() {
		c.Hide()

		return
	}
	c.Show()
}

// Show makes the overlay visible without stealing focus and pins it on top.
func (c *Controller) Show() {
	hwnd := c.handle()
	if hwnd == 0 {
		return
	}

	showWindow(hwnd)

	c.mu.Lock()
	c.visible = true
	c.mu.Unlock()
}

// Hide hides the overlay window.
func (c *Controller) Hide() {
	hwnd := c.handle()
	if hwnd == 0 {
		return
	}

	hideWindow(hwnd)

	c.mu.Lock()
	c.visible = false
	c.mu.Unlock()
}

// SetOpacity sets the overlay's whole-window opacity. The call is issued on a
// goroutine so it never blocks the UI loop that drives the opacity slider.
func (c *Controller) SetOpacity(alpha float64) {
	hwnd := c.handle()
	if hwnd == 0 {
		return
	}

	go setLayeredAlpha(hwnd, alphaToByte(alpha))
}

func alphaToByte(alpha float64) byte {
	switch {
	case alpha <= 0:
		return 0
	case alpha >= 1:
		return 255
	default:
		return byte(alpha * 255)
	}
}

// Visible reports the current visibility.
func (c *Controller) Visible() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.visible
}

// Stop asks the message window to close from its own thread. WM_CLOSE triggers
// the default DestroyWindow, whose WM_DESTROY handler removes the tray icon,
// unregisters the hotkey, and posts the quit message that unwinds the loop.
func (c *Controller) Stop() error {
	hwnd := c.messageHandle()
	if hwnd == 0 {
		return nil
	}

	procPostMessage.Call(hwnd, uintptr(wmClose), 0, 0)

	return nil
}

func (c *Controller) handle() uintptr {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.hwnd
}

func (c *Controller) messageHandle() uintptr {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.msgHWND
}

func (c *Controller) setMessageHandle(h uintptr) {
	c.mu.Lock()
	c.msgHWND = h
	c.mu.Unlock()
}
