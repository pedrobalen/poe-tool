package app

import (
	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	pt "github.com/pedrobalen/poe-build-overlay/internal/passive_tree"
	"github.com/pedrobalen/poe-build-overlay/internal/pobfiles"
)

// screen identifies which view the overlay is currently showing.
type screen int

const (
	// screenLoading is shown briefly during startup.
	screenLoading screen = iota
	// screenImport prompts for a build link or code.
	screenImport
	// screenBuild shows the active build's progression.
	screenBuild
	// screenBrowse lists the locally saved builds.
	screenBrowse
	// screenPobFiles lists builds saved by the Path of Building desktop app.
	screenPobFiles
)

// uiState is the mutable state shared between the UI loop and the background
// import goroutine. All access is guarded by App.mu.
type uiState struct {
	screen    screen
	build     *builds.Build
	treeData  *pt.TreeData
	treeErr   error
	importing bool
	importErr string
	locked    bool
	opacity   float64
	summaries []builds.Summary
	pobFiles  []pobfiles.Build
	pobErr    string
}
