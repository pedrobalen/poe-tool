package app

import (
	"errors"
	"log"

	gioapp "gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	"github.com/pedrobalen/poe-build-overlay/internal/overlay"
	"github.com/pedrobalen/poe-build-overlay/internal/pobfiles"
)

// Run starts the platform integration, loads the initial state, and runs the
// Gio event loop until the window is destroyed.
func (a *App) Run(w *gioapp.Window) error {
	a.window = w

	if err := a.plat.Start(); err != nil {
		log.Printf("platform integration unavailable: %v", err)
	}

	a.loadInitialState()

	var ops op.Ops
	for {
		ev := w.Event()

		// The first ViewEvent carries the native handle; binding applies the
		// overlay styling and hides the window until the user opens it.
		if a.plat.TryBind(ev) {
			continue
		}

		switch e := ev.(type) {
		case gioapp.DestroyEvent:
			_ = a.plat.Stop()

			return e.Err
		case gioapp.FrameEvent:
			gtx := gioapp.NewContext(&ops, e)
			a.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

// loadInitialState restores overlay preferences and the active build (if any)
// from local storage without touching the network, satisfying the fast-open
// requirement.
func (a *App) loadInitialState() {
	prefs := a.loadPreferences()
	a.chrome.SetOpacityAlpha(prefs.opacity)
	a.plat.SetOpacity(prefs.opacity)

	a.mu.Lock()
	a.state.locked = prefs.locked
	a.state.opacity = prefs.opacity
	a.state.compare = prefs.compare
	a.mu.Unlock()

	build, err := a.deps.BuildRepo.FindActive(a.ctx)
	if err != nil {
		a.setScreen(screenImport)

		return
	}

	treeData, treeErr := a.deps.TreeLoader.Load(a.ctx, build.TreeVersion)

	a.mu.Lock()
	a.state.build = &build
	a.state.treeData = treeData
	a.state.treeErr = treeErr
	a.state.screen = screenBuild
	a.mu.Unlock()
}

// handleChrome applies control-bar actions: lock toggle, opacity, and screen
// navigation to import or browse.
func (a *App) handleChrome(action overlay.ChromeAction) {
	if action.DragStart {
		a.plat.BeginDrag()
	}
	if action.DragEnd {
		a.plat.EndDrag()
	}
	if action.ToggleLock {
		a.toggleLock()
	}
	if action.OpacityChanged {
		a.applyOpacity(action.Opacity, action.OpacitySettled)
	}
	if action.GotoImport {
		a.gotoImport()
	}
	if action.GotoBrowse {
		a.gotoBrowse()
	}
	if action.Quit {
		a.requestClose()
	}
}

// handleBrowse applies saved-build actions: activate, delete, import, or back.
func (a *App) handleBrowse(action overlay.BuildsAction) {
	switch {
	case action.SelectID != "":
		a.activateBuild(action.SelectID)
	case action.DeleteID != "":
		a.deleteBuild(action.DeleteID)
	case action.Import:
		a.gotoImport()
	case action.Back:
		a.backFromBrowse()
	}
}

func (a *App) toggleLock() {
	a.mu.Lock()
	a.state.locked = !a.state.locked
	locked := a.state.locked
	a.mu.Unlock()

	a.savePreference(keyLocked, boolToSetting(locked))
	a.invalidate()
}

func (a *App) applyOpacity(opacity float64, settled bool) {
	a.plat.SetOpacity(opacity)

	a.mu.Lock()
	a.state.opacity = opacity
	a.mu.Unlock()

	// Persist only once the slider settles to avoid a write per drag frame.
	if settled {
		a.savePreference(keyOpacity, formatFloat(opacity))
	}
	a.invalidate()
}

func (a *App) gotoImport() {
	a.setScreen(screenImport)
}

func (a *App) gotoBrowse() {
	summaries, err := a.deps.BuildRepo.List(a.ctx)
	if err != nil {
		log.Printf("listing builds: %v", err)
		summaries = nil
	}

	a.mu.Lock()
	a.state.summaries = summaries
	a.state.screen = screenBrowse
	a.mu.Unlock()
	a.invalidate()
}

func (a *App) backFromBrowse() {
	a.mu.Lock()
	hasBuild := a.state.build != nil
	a.mu.Unlock()

	if hasBuild {
		a.setScreen(screenBuild)

		return
	}
	a.setScreen(screenImport)
}

// activateBuild makes the chosen build active, reloads it and its tree, and
// shows the build screen.
func (a *App) activateBuild(id string) {
	if err := a.deps.BuildRepo.SetActive(a.ctx, id); err != nil {
		log.Printf("activating build: %v", err)

		return
	}

	build, err := a.deps.BuildRepo.FindByID(a.ctx, id)
	if err != nil {
		log.Printf("loading build: %v", err)

		return
	}

	treeData, treeErr := a.deps.TreeLoader.Load(a.ctx, build.TreeVersion)

	a.mu.Lock()
	a.state.build = &build
	a.state.treeData = treeData
	a.state.treeErr = treeErr
	a.state.screen = screenBuild
	a.mu.Unlock()
	a.invalidate()
}

// deleteBuild removes a saved build and refreshes the browse list, updating the
// active build when the deleted one was active.
func (a *App) deleteBuild(id string) {
	if err := a.deps.BuildRepo.Delete(a.ctx, id); err != nil {
		log.Printf("deleting build: %v", err)

		return
	}

	summaries, err := a.deps.BuildRepo.List(a.ctx)
	if err != nil {
		log.Printf("listing builds: %v", err)
	}

	a.mu.Lock()
	a.state.summaries = summaries
	if a.state.build != nil && a.state.build.ID == id {
		a.state.build = nil
		a.state.treeData = nil
		a.state.treeErr = nil
	}
	a.mu.Unlock()
	a.invalidate()
}

// handleNav applies a navigation action, persists the selected stage, and
// applies the compare toggle.
func (a *App) handleNav(action overlay.NavAction, build *builds.Build) {
	if action.ToggleCompare {
		a.setCompare(action.CompareOn)
	}

	target := build.CurrentStage
	switch action.Kind {
	case overlay.NavPrev:
		target = build.ClampStage(build.CurrentStage - 1)
	case overlay.NavNext:
		target = build.ClampStage(build.CurrentStage + 1)
	default:
		return
	}
	if target == build.CurrentStage {
		return
	}

	a.mu.Lock()
	build.CurrentStage = target
	a.mu.Unlock()

	if err := a.deps.Service.SetCurrentStage(a.ctx, build.ID, target); err != nil {
		log.Printf("persisting current stage: %v", err)
	}
	a.invalidate()
}

// setCompare persists and applies the tree compare-mode toggle.
func (a *App) setCompare(on bool) {
	a.mu.Lock()
	a.state.compare = on
	a.mu.Unlock()

	a.savePreference(keyCompare, boolToSetting(on))
	a.invalidate()
}

// handleEscape hides the overlay when Escape is pressed. It grabs keyboard focus
// for the build screen (which has no text input to conflict with).
func (a *App) handleEscape(gtx layout.Context) {
	gtx.Execute(key.FocusCmd{Tag: a})

	for {
		ev, ok := gtx.Event(key.Filter{Focus: a, Name: key.NameEscape})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			a.plat.Hide()
			a.invalidate()
		}
	}
}

// gotoPobFiles lists the local Path of Building builds and shows them.
func (a *App) gotoPobFiles() {
	files, err := pobfiles.List()
	errText := ""
	if errors.Is(err, pobfiles.ErrDirNotFound) {
		errText = "Path of Building builds folder not found. Open a build in Path of Building first, or use a link/code."
	} else if err != nil {
		errText = "Could not read Path of Building builds: " + err.Error()
	}

	a.mu.Lock()
	a.state.pobFiles = files
	a.state.pobErr = errText
	a.state.screen = screenPobFiles
	a.mu.Unlock()
	a.invalidate()
}

// startImport runs the link/code import pipeline off the UI thread.
func (a *App) startImport(input string) {
	if !a.beginImport() {
		return
	}

	go func() {
		build, err := a.deps.Service.Import(a.ctx, input)
		a.onImportDone(build, err)
	}()
}

// importPobFile imports a local Path of Building build file off the UI thread.
func (a *App) importPobFile(path string) {
	if !a.beginImport() {
		return
	}

	go func() {
		build, err := a.deps.Service.ImportFile(a.ctx, path)
		a.onImportDone(build, err)
	}()
}

// beginImport marks an import in progress, returning false if one is already
// running.
func (a *App) beginImport() bool {
	a.mu.Lock()
	if a.state.importing {
		a.mu.Unlock()

		return false
	}
	a.state.importing = true
	a.state.importErr = ""
	a.mu.Unlock()

	a.invalidate()

	return true
}

func (a *App) onImportDone(build builds.Build, err error) {
	a.mu.Lock()
	a.state.importing = false
	if err != nil {
		a.state.importErr = friendlyError(err)
		a.mu.Unlock()
		a.invalidate()

		return
	}

	treeData, treeErr := a.deps.TreeLoader.Load(a.ctx, build.TreeVersion)
	a.state.build = &build
	a.state.treeData = treeData
	a.state.treeErr = treeErr
	a.state.screen = screenBuild
	a.mu.Unlock()

	a.plat.Show()
	a.invalidate()
}

func (a *App) setScreen(s screen) {
	a.mu.Lock()
	a.state.screen = s
	a.mu.Unlock()
	a.invalidate()
}

func (a *App) invalidate() {
	if a.window != nil {
		a.window.Invalidate()
	}
}

func (a *App) requestClose() {
	if w, ok := a.window.(*gioapp.Window); ok {
		w.Perform(system.ActionClose)
	}
}
