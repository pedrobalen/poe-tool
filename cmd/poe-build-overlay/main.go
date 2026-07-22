package main

import (
	"context"
	"log"
	"os"

	gioapp "gioui.org/app"

	"github.com/pedrobalen/poe-build-overlay/internal/app"
	"github.com/pedrobalen/poe-build-overlay/internal/builds"
	"github.com/pedrobalen/poe-build-overlay/internal/config"
	"github.com/pedrobalen/poe-build-overlay/internal/importers"
	"github.com/pedrobalen/poe-build-overlay/internal/overlay"
	pt "github.com/pedrobalen/poe-build-overlay/internal/passive_tree"
	"github.com/pedrobalen/poe-build-overlay/internal/storage"
	"github.com/pedrobalen/poe-build-overlay/internal/storage/repositories"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}
	closeLog := setupLogging(cfg.LogPath)

	ctx := context.Background()

	db, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}

	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("migrating database: %v", err)
	}

	// Cleanup runs when the overlay loop returns. It cannot be deferred here:
	// the loop exits the process via os.Exit from its own goroutine, so main's
	// deferred calls would never fire.
	cleanup := func() {
		_ = db.Close()
		closeLog()
	}

	deps := wire(cfg, db)

	windowState, err := deps.WindowRepo.Load(ctx)
	if err != nil {
		log.Printf("loading window state: %v", err)
		windowState = repositories.DefaultWindowState
	}

	application := app.New(ctx, deps)

	go func() {
		w := new(gioapp.Window)
		w.Option(overlay.Options(windowState)...)
		if err := application.Run(w); err != nil {
			log.Printf("overlay exited: %v", err)
		}
		cleanup()
		os.Exit(0)
	}()

	gioapp.Main()
}

// wire constructs the repositories, services, and tree loader.
func wire(cfg config.Config, db *storage.DB) app.Deps {
	buildRepo := repositories.NewBuildRepo(db.DB)
	windowRepo := repositories.NewWindowRepo(db.DB)
	settingsRepo := repositories.NewSettingsRepo(db.DB)
	treeRepo := repositories.NewTreeRepo(db.DB)

	// Structural tree data is optional and user-extensible: drop
	// "<version>.json" files into <dataDir>/tree to enable the graphical tree.
	treeSource := pt.NewJSONSource(os.DirFS(cfg.DataDir), "tree")
	treeLoader := pt.NewLoader(treeRepo, treeSource)

	registry := importers.NewRegistry(nil)
	service := builds.NewService(registry, buildRepo, nil)

	return app.Deps{
		AppName:      "PoE Build Overlay",
		Service:      service,
		BuildRepo:    buildRepo,
		WindowRepo:   windowRepo,
		SettingsRepo: settingsRepo,
		TreeLoader:   treeLoader,
	}
}

// setupLogging directs logs to a file, falling back to stderr on failure.
func setupLogging(path string) func() {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("could not open log file %q, logging to stderr: %v", path, err)

		return func() {}
	}

	log.SetOutput(f)

	return func() { _ = f.Close() }
}
