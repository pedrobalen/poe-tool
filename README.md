# PoE Build Progression Overlay

A small Windows overlay that shows a Path of Exile build's **leveling
progression** on top of the game. Press **Ctrl+B** to open it, glance at what to
do next, and press **Ctrl+B** (or **Esc**) to close it.

## What it does

You import a build that was made in **Path of Building**, and the overlay shows,
for each saved step of that build:

- the passive tree, highlighting which points to take next (green) compared to
  the previous step, PoB-style;
- the skill gems and their support links;
- the mastery effect chosen for each mastery node (hover to see it).

It only **reads and displays** what the build's author already set up in Path of
Building — it doesn't calculate DPS, defenses, or anything else. Think of it as a
read-only, in-game view of a PoB build's leveling stages.

## How to use

1. Run the app. It sits in the system tray and starts hidden.
2. Press **Ctrl+B** to open the overlay.
3. Import a build, either by:
   - pasting a `pobb.in` / Pastebin link or a raw PoB code, or
   - clicking **Import from Path of Building** to pick one of your locally saved
     PoB builds.
4. Use the **←/→** arrows to move between the build's steps and follow the
   highlighted passive points and skills.

Tips: drag the title to move the window, use the lock icon to pin it in place,
and the slider to adjust opacity so it stays out of your way while you play.

The graphical passive tree needs tree data for the build's version. If it shows
"unavailable", generate it once with `cmd/treegen` from the official GGG skill
tree export (see `cmd/treegen`).

## Build and run

```bash
go run ./cmd/poe-build-overlay
# or a windowed binary (no console window):
go build -ldflags "-H=windowsgui" -o poe-build-overlay.exe ./cmd/poe-build-overlay
# or the optimized, smaller release binary:
go build -trimpath -ldflags "-s -w -H=windowsgui" -o poe-build-overlay.exe ./cmd/poe-build-overlay
```

Windows only. No C toolchain needed. Data lives in
`%AppData%\poe-build-overlay\`. The result is a single, self-contained portable
`.exe` — no installer, no admin rights, nothing to place alongside it. Passive
tree data for the shipped game versions is embedded in the binary.

### App icon and version info

The taskbar/tray icon and the executable's version metadata come from
`assets/icons/appicon.ico` and `versioninfo.json`, compiled into
`cmd/poe-build-overlay/resource_windows.syso`. That `.syso` is committed so a
plain `go build` already carries the icon. Regenerate it after changing the icon
or metadata:

```bash
go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0 \
  -o cmd/poe-build-overlay/resource_windows.syso versioninfo.json
```

### Releasing

Pushing a version tag builds the portable `.exe` and publishes it on the GitHub
Releases page (see `.github/workflows/release.yml`), stamping it with the tag's
version:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## About

This is a simple personal project. It is not affiliated with, endorsed by, or
connected to Grinding Gear Games or Path of Building in any way, and there is no
commercial or external interest behind it — it started as something I built for
myself.
