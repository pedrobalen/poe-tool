// Package app wires the overlay's components together and runs the Gio event
// loop. It owns the mutable UI state and coordinates the background import work
// so the UI thread only ever reads already-processed data.
package app

import (
	"context"
	"sync"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	"github.com/pedrobalen/poe-build-overlay/internal/overlay"
	pt "github.com/pedrobalen/poe-build-overlay/internal/passive_tree"
	"github.com/pedrobalen/poe-build-overlay/internal/platform"
	"github.com/pedrobalen/poe-build-overlay/internal/storage/repositories"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/theme"
	"github.com/pedrobalen/poe-build-overlay/internal/ui/widgets"
)

// Deps are the collaborators the app needs, injected by main.
type Deps struct {
	AppName      string
	Service      *builds.Service
	BuildRepo    *repositories.BuildRepo
	WindowRepo   *repositories.WindowRepo
	SettingsRepo *repositories.SettingsRepo
	TreeLoader   *pt.Loader
}

// App holds the running application's state and views.
type App struct {
	deps Deps
	ctx  context.Context

	th         *theme.Theme
	importView *overlay.ImportView
	buildView  *overlay.BuildView
	buildsView *overlay.BuildsView
	pobView    *overlay.PobFilesView
	chrome     *overlay.Chrome

	plat platform.Overlay
	ctl  *overlay.Controller

	window invalidator

	mu    sync.Mutex
	state uiState
}

// invalidator is the subset of the Gio window the app calls from any goroutine.
type invalidator interface {
	Invalidate()
}

// New constructs the app and its platform integration. The Gio window is
// supplied later in Run.
func New(ctx context.Context, deps Deps) *App {
	a := &App{
		deps:       deps,
		ctx:        ctx,
		th:         theme.New(),
		importView: &overlay.ImportView{},
		buildView:  &overlay.BuildView{},
		buildsView: overlay.NewBuildsView(),
		pobView:    overlay.NewPobFilesView(),
		chrome:     overlay.NewChrome(),
	}
	a.state.screen = screenLoading
	a.state.opacity = 1
	a.state.compare = true

	a.plat = platform.New(platform.Config{
		AppName:  deps.AppName,
		OnToggle: a.onToggle,
		OnQuit:   a.onQuit,
	})
	a.ctl = overlay.NewController(a.plat)

	return a
}

// onToggle flips overlay visibility and requests a redraw. It runs on the
// hotkey/tray thread, so it only calls goroutine-safe window methods.
func (a *App) onToggle() {
	a.plat.Toggle()
	if a.window != nil {
		a.window.Invalidate()
	}
}

// onQuit is invoked from the tray Exit item; closing the window unwinds Run.
func (a *App) onQuit() {
	a.requestClose()
}

// layout renders the current screen. It runs on the UI goroutine.
func (a *App) layout(gtx layout.Context) {
	widgets.FillBackground(gtx, a.th.Bg, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})

	a.mu.Lock()
	st := a.state
	a.mu.Unlock()

	switch st.screen {
	case screenImport:
		a.layoutImport(gtx, st)
	case screenBuild:
		a.layoutBuild(gtx, st)
	case screenBrowse:
		a.layoutBrowse(gtx, st)
	case screenPobFiles:
		a.layoutPobFiles(gtx, st)
	default:
		a.layoutLoading(gtx)
	}
}

func (a *App) layoutLoading(gtx layout.Context) {
	lbl := material.Body1(a.th.Theme, "Loading…")
	lbl.Color = a.th.Muted
	layout.Center.Layout(gtx, lbl.Layout)
}

func (a *App) layoutImport(gtx layout.Context, st uiState) {
	req := a.importView.Layout(gtx, a.th, st.importErr, st.importing, st.build != nil)
	switch {
	case req.Requested:
		a.startImport(req.Input)
	case req.FromPoB:
		a.gotoPobFiles()
	case req.Cancelled:
		a.setScreen(screenBuild)
	}
	// Escape is not captured on the import screen so it does not fight the text
	// editor for keyboard focus; Ctrl+B or the tray still closes the overlay.
}

func (a *App) layoutBuild(gtx layout.Context, st uiState) {
	if st.build == nil {
		a.layoutLoading(gtx)

		return
	}

	var (
		chromeAction overlay.ChromeAction
		navAction    overlay.NavAction
	)

	layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				var dims layout.Dimensions
				dims, chromeAction = a.chrome.Layout(gtx, a.th, st.build.Name, st.locked)

				return dims
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				navAction = a.buildView.Layout(gtx, a.th, st.build, st.treeData, st.treeErr, st.compare)

				return layout.Dimensions{Size: gtx.Constraints.Max}
			}),
		)
	})

	a.handleChrome(chromeAction)
	a.handleNav(navAction, st.build)
	a.handleEscape(gtx)
}

func (a *App) layoutBrowse(gtx layout.Context, st uiState) {
	action := a.buildsView.Layout(gtx, a.th, st.summaries)
	a.handleBrowse(action)
	a.handleEscape(gtx)
}

func (a *App) layoutPobFiles(gtx layout.Context, st uiState) {
	action := a.pobView.Layout(gtx, a.th, st.pobFiles, st.pobErr)
	switch {
	case action.SelectPath != "":
		a.importPobFile(action.SelectPath)
	case action.Back:
		a.setScreen(screenImport)
	}
	a.handleEscape(gtx)
}
