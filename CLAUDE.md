# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A lightweight **Windows** desktop overlay (Go + Gio) that displays a Path of
Exile build's progression on top of the game, toggled with **Ctrl+B**. It
imports builds authored in Path of Building (pobb.in / Pastebin / raw PoB code)
and reuses the author's saved stages, skill sets, and gem changes. Scope,
phases, and rationale are in `PLAN.md`.

## Commands

```bash
go run ./cmd/poe-build-overlay                                   # run from source
go build -ldflags "-H=windowsgui" -o poe-build-overlay.exe ./cmd/poe-build-overlay
go test ./...                                                    # all tests
go test ./internal/pob/ -run TestNormalize                      # a single test
go test ./internal/storage/...                                  # SQLite integration tests
go vet ./...
gofmt -l .                                                      # must print nothing
```

No C toolchain is needed: SQLite is `modernc.org/sqlite` (pure Go). Gio on
Windows builds without cgo.

## Architecture

Full detail in `docs/architecture.md`. The dependency rule: the import
pipeline (`importers`, `pob`) and `passive_tree` are leaves; `builds` is the
domain; `storage` implements domain repository interfaces; `app` wires
everything and runs the Gio loop.

Load-bearing design decisions that span files:

- **Heavy work only at import.** `builds.Service` runs decode → parse →
  normalize → **precompute stage diffs** → persist. Opening the overlay reads
  already-processed rows from SQLite — no network, XML, or diffing on open. Keep
  it that way: new per-stage data should be computed at import and stored, not
  derived in the UI.
- **Repository interfaces live with the domain** (`builds.BuildRepository`,
  `passive_tree.Store`); implementations in `internal/storage/repositories`
  take `*sql.DB` so `repositories` never imports `storage` (avoids a cycle).
- **`pob` and `importers` do not import the domain.** They return plain
  strings/structs; `builds.Service` maps them onto domain types. Preserve this.
- **Author intent is never invented.** Stage names/order come straight from the
  PoB export; a single-tree build yields one stage. Tree↔skill-set association
  is by name, then position, then default, and the chosen `StageAssociation` is
  surfaced, not hidden. The app never judges whether a change is "better".
- **Platform integration is GOOS-selected.** `internal/platform` is the
  interface; `internal/platform/windows/*_windows.go` is the real Win32 impl
  (global hotkey via a message-only window on a locked OS thread, tray via
  `Shell_NotifyIcon`, topmost tool-window styling on the Gio HWND captured from
  `app.Win32ViewEvent`). `new_other.go` is a no-op for non-Windows builds.

## Conventions

- Follow idiomatic Go style (see the Go proverbs and Effective Go): guard
  clauses / early returns, `switch` over if-else chains, initialized (never nil)
  slices/maps, field-named composite literals, ≤4 params (options struct
  otherwise), unexport aggressively. Run `gofmt` (the repo is gofmt-clean).
- Migrations are embedded SQL in `internal/storage/migrations/NNNN_name.sql`,
  applied in order and tracked in `schema_migrations`. Add a new numbered file;
  never edit an applied migration.
- IDs come from `internal/id` (local, no UUID dependency). Timestamps persist as
  RFC3339 UTC text.
- Passive tree structural data is **not** in the repo. See the README; it loads
  from `%AppData%\poe-build-overlay\tree\<version>.json` and caches to SQLite.

## Verification notes

The PoB pipeline and SQLite repositories have tests. The Gio UI and Win32
integration are **not** unit-tested — verify UI/platform changes by building and
launching `./cmd/poe-build-overlay` on Windows. Startup (config → migrate →
window → hotkey/tray) can be smoke-tested by running the binary briefly and
checking `%AppData%\poe-build-overlay\overlay.log` for errors.

Not affiliated with or endorsed by Grinding Gear Games.
