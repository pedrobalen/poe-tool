# Architecture

Modules, boundaries, and data flow for the PoE Build Progression Overlay.

## Dependency direction

Lower layers never import higher ones. The import pipeline is a leaf; the app
layer wires everything.

```
cmd/poe-build-overlay        entrypoint: config, storage, wiring, Gio bootstrap
        │
        ▼
internal/app                 UI state machine + Gio event loop (background import)
   ├─ internal/overlay        views: controller, window options, import & build views
   │     └─ internal/ui        theme, reusable widgets, passive-tree Gio widget
   ├─ internal/platform        OS integration interface (Overlay)
   │     └─ .../windows         Win32 hotkey, tray, topmost tool-window styling
   ├─ internal/builds          domain: Build/Stage/Gem, Service, stage & gem diff
   │     ├─ internal/importers  pobb.in / Pastebin / direct-code resolution (leaf)
   │     └─ internal/pob         decode (base64+zlib) → parse (XML) → normalize
   ├─ internal/passive_tree     tree model, viewport math, loader, JSON source
   ├─ internal/storage          SQLite: Open, migrations, repositories
   └─ internal/config           filesystem paths (data dir, db, log)
```

Key acyclic choices:

- `internal/importers` and `internal/pob` have **no** dependency on the domain;
  they return plain strings/structs the domain maps onto its types.
- Repository interfaces (`builds.BuildRepository`, `passive_tree.Store`) live in
  the domain packages; implementations live in `internal/storage/repositories`
  and accept `*sql.DB` directly, so `repositories` does not import `storage`.
- `internal/platform` exposes an `Overlay` interface with a GOOS-selected
  constructor (`new_windows.go` / `new_other.go`); the app depends only on the
  interface.

## Data flow

### Import (heavy, off the UI thread)

```
link/code → importers.Registry.Import → pob.Decode → pob.Parse → pob.Normalize
          → builds.Service assembles Build + Stages, precomputes node/gem diffs
          → BuildRepo.Save (transactional) → SetActive
```

### Open overlay (fast, local only)

```
Ctrl+B → app already holds the active build (loaded once at startup from SQLite)
       → platform shows the window; the build view reads precomputed rows
```

No network, XML parsing, or diffing happens on open. Stage switching reads
already-loaded stages; the tree widget refits its camera on stage change.

## Concurrency

- The Gio event loop runs on one goroutine and owns all reads of `app.uiState`.
- Import runs on a separate goroutine; it writes `uiState` under `app.mu` and
  calls `Window.Invalidate` (goroutine-safe) to request a redraw.
- The Win32 hotkey/tray message loop runs on its own locked OS thread and
  communicates via the `OnToggle`/`OnQuit` callbacks, which only call
  goroutine-safe window methods (`Toggle`/`Show`/`Hide`/`Invalidate`).

## Persistence

SQLite with embedded, versioned migrations (`internal/storage/migrations`,
applied via `schema_migrations`). WAL journal, enforced foreign keys, single
writer connection. Times are stored as RFC3339 UTC text.
