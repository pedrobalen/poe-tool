package overlay

import (
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
)

const (
	// minOpacity keeps the overlay faintly visible at the slider's low end so it
	// can never vanish entirely.
	minOpacity  = 0.30
	opacitySpan = 1.0 - minOpacity
)

// Chrome is the top control bar: a drag handle, a lock toggle, an opacity
// slider, and buttons to import a new build or browse saved builds.
type Chrome struct {
	lock    widget.Clickable
	imp     widget.Clickable
	browse  widget.Clickable
	close   widget.Clickable
	opacity widget.Float
	lastOp  float32
	dragTag int // stable identity for the drag-handle pointer input

	iconLock   *widget.Icon
	iconUnlock *widget.Icon
	iconAdd    *widget.Icon
	iconList   *widget.Icon
	iconClose  *widget.Icon
}

// NewChrome builds the control bar with its icons.
func NewChrome() *Chrome {
	c := &Chrome{
		iconLock:   mustIcon(icons.ActionLock),
		iconUnlock: mustIcon(icons.ActionLockOpen),
		iconAdd:    mustIcon(icons.ContentAdd),
		iconList:   mustIcon(icons.ActionViewList),
		iconClose:  mustIcon(icons.NavigationClose),
	}
	c.opacity.Value = 1
	c.lastOp = 1

	return c
}

// SetOpacityAlpha positions the slider to reflect a stored alpha value.
func (c *Chrome) SetOpacityAlpha(alpha float64) {
	v := (alpha - minOpacity) / opacitySpan
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	c.opacity.Value = float32(v)
	c.lastOp = float32(v)
}

// ChromeAction reports what the user did on the control bar this frame.
type ChromeAction struct {
	ToggleLock     bool
	OpacityChanged bool
	OpacitySettled bool // true when the slider is not actively being dragged
	Opacity        float64
	GotoImport     bool
	GotoBrowse     bool
	Quit           bool // user asked to close the app
	DragStart      bool // user pressed the drag handle to begin moving the window
	DragEnd        bool // user released the drag handle
}

// Layout draws the control bar and returns its size and any action. locked
// reflects whether window dragging is currently disabled.
func (c *Chrome) Layout(gtx layout.Context, th *theme.Theme, title string, locked bool) (layout.Dimensions, ChromeAction) {
	action := ChromeAction{
		ToggleLock: c.lock.Clicked(gtx),
		GotoImport: c.imp.Clicked(gtx),
		GotoBrowse: c.browse.Clicked(gtx),
		Quit:       c.close.Clicked(gtx),
	}

	if !locked {
		c.readDrag(gtx, &action)
	}

	dims := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, c.dragHandle(th, title, !locked)),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(c.opacitySlider(th)),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(smallIconButton(th, &c.lock, c.lockIcon(locked), "Lock position")),
		layout.Rigid(smallIconButton(th, &c.imp, c.iconAdd, "Import build")),
		layout.Rigid(smallIconButton(th, &c.browse, c.iconList, "Saved builds")),
		layout.Rigid(c.closeButton(th)),
	)

	c.applyOpacityChange(&action)

	return dims, action
}

// readDrag translates pointer presses/releases on the drag handle into
// window-move start/stop signals for the platform layer.
func (c *Chrome) readDrag(gtx layout.Context, action *ChromeAction) {
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: &c.dragTag,
			Kinds:  pointer.Press | pointer.Release | pointer.Cancel,
		})
		if !ok {
			break
		}
		pe, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch pe.Kind {
		case pointer.Press:
			action.DragStart = true
		case pointer.Release, pointer.Cancel:
			action.DragEnd = true
		}
	}
}

// dragHandle renders the build title. When unlocked, its area is a pointer input
// region whose presses drive the window drag.
func (c *Chrome) dragHandle(th *theme.Theme, title string, draggable bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(th.Theme, unit.Sp(16), title)
		lbl.Color = th.Fg
		lbl.Font.Weight = 700
		lbl.MaxLines = 1

		dims := lbl.Layout(gtx)
		if draggable {
			area := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
			event.Op(gtx.Ops, &c.dragTag)
			area.Pop()
		}

		return dims
	}
}

func (c *Chrome) opacitySlider(th *theme.Theme) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		width := gtx.Dp(unit.Dp(110))
		gtx.Constraints.Min.X = width
		gtx.Constraints.Max.X = width
		sl := material.Slider(th.Theme, &c.opacity)
		sl.Color = th.Line

		return sl.Layout(gtx)
	}
}

func (c *Chrome) applyOpacityChange(action *ChromeAction) {
	if c.opacity.Value == c.lastOp {
		return
	}
	c.lastOp = c.opacity.Value
	action.OpacityChanged = true
	action.OpacitySettled = !c.opacity.Dragging()
	action.Opacity = minOpacity + opacitySpan*float64(c.opacity.Value)
}

// closeButton is the quit-app control, tinted red to distinguish it from the
// hide hotkey.
func (c *Chrome) closeButton(th *theme.Theme) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		b := material.IconButton(th.Theme, &c.close, c.iconClose, "Close app")
		b.Background = th.Surface
		b.Color = th.Removed
		b.Size = unit.Dp(18)
		b.Inset = layout.UniformInset(unit.Dp(8))

		return b.Layout(gtx)
	}
}

func (c *Chrome) lockIcon(locked bool) *widget.Icon {
	if locked {
		return c.iconLock
	}

	return c.iconUnlock
}

func mustIcon(data []byte) *widget.Icon {
	icon, err := widget.NewIcon(data)
	if err != nil {
		panic("overlay: loading icon: " + err.Error())
	}

	return icon
}
