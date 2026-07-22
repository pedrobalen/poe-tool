# Passive tree data

Structural passive-tree data is **not bundled** with the repository. At runtime
the overlay looks for `<treeVersion>.json` files under
`%AppData%\poe-build-overlay\tree\` (for example `3_25.json`).

See the "Passive tree data" section of the top-level `README.md` for the JSON
schema. Files placed there are parsed once and cached in SQLite, versioned by
tree version.
