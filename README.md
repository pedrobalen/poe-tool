# PoE Build Progression Overlay

A lightweight Windows desktop overlay, written in Go, that shows a Path of
Exile build's **progression** on top of the game. It imports builds that were
already authored in Path of Building (via `pobb.in`, a compatible Pastebin, or a
raw PoB code) and reuses the stages the author saved — passive-tree steps, skill
sets, and gem changes — so you can follow a build without opening Path of
Building.

The overlay stays hidden and is toggled with **Ctrl+B**.

> This project is not affiliated with or endorsed by Grinding Gear Games.

## What it does

- Lives in the system tray; the overlay opens/closes with `Ctrl+B` (and closes
  with `Esc` while the build view is focused).
- Imports a build from a `pobb.in` link, a compatible Pastebin, or a pasted PoB
  code — detected automatically.
- Preserves the author's stage **names and order** (e.g. `Act 1`, `Level 40`,
  `Endgame`); it never invents progression.
- Precomputes, at import time, the passive-node and gem differences between each
  stage and the previous one.
- Stores the normalized build locally in SQLite, so reopening never touches the
  network or reparses XML.
- Renders the passive tree graphically (when structural data for the build's
  tree version is available), highlighting the nodes added at the current stage.

Out of scope (by design): price checking, DPS/defensive calculations, the PoB
mod engine, live character import, and editing builds.

## Requirements

- Go 1.24+ (developed against 1.26).
- Windows for the full experience (global hotkey, tray, topmost tool-window
  styling). The code builds on other platforms, but OS integration is a no-op
  there.
- No C toolchain required: SQLite uses the pure-Go `modernc.org/sqlite` driver.

## Build & run

```bash
# Run from source
go run ./cmd/poe-build-overlay

# Build a windowed binary (no console window)
go build -ldflags "-H=windowsgui" -o poe-build-overlay.exe ./cmd/poe-build-overlay
```

Application data (SQLite database, log, and optional tree data) lives in
`%AppData%\poe-build-overlay\`.

## Passive tree data

The graphical tree needs structural data (node positions and connections) for
the build's tree version. This data is **not bundled**. To enable the tree,
drop a JSON file named `<treeVersion>.json` (e.g. `3_25.json`) into
`%AppData%\poe-build-overlay\tree\`. The expected shape:

```json
{
  "nodes": [
    { "id": 123, "x": 100.0, "y": 200.0, "kind": "notable", "name": "Barbarism", "group": 4, "mastery": false }
  ],
  "connections": [
    { "from": 123, "to": 456 }
  ]
}
```

Imported tree data is cached in SQLite and versioned by tree version. When data
is missing, the overlay shows a clear message instead of drawing a mismatched
tree.

## Tests

```bash
go test ./...        # unit + storage integration tests
go vet ./...
gofmt -l .           # should print nothing
```

The core pipeline (PoB decode → parse → normalize → stage diff) and the SQLite
repositories are covered by tests. The Gio UI and Win32 integration are
exercised by building and launching the app, not by automated tests.

## Architecture

See [`docs/architecture.md`](docs/architecture.md) for module boundaries and
data flow. In short:

```
importers → pob (decode/parse/normalize) → builds (domain, diff, service)
                                              │
                                     storage (SQLite, migrations, repositories)
                                              │
app (state + Gio loop) ── overlay (views) ── ui (theme, widgets, tree)
                       └─ platform/windows (hotkey, tray, window styling)
```

Heavy work (download, Base64, zlib, XML, normalization, diffing) happens only
during import. Opening the overlay reads already-processed rows from SQLite.
