-- Initial schema for the PoE Build Progression Overlay.
-- All timestamps are ISO-8601 text in UTC.

CREATE TABLE app_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE builds (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    class         TEXT NOT NULL DEFAULT '',
    ascendancy    TEXT NOT NULL DEFAULT '',
    tree_version  TEXT NOT NULL DEFAULT '',
    source_type   TEXT NOT NULL,
    source_url    TEXT NOT NULL DEFAULT '',
    source_hash   TEXT NOT NULL,
    current_stage INTEGER NOT NULL DEFAULT 0,
    imported_at   TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_builds_source_hash ON builds (source_hash);

CREATE TABLE build_sources (
    build_id     TEXT PRIMARY KEY REFERENCES builds (id) ON DELETE CASCADE,
    source_type  TEXT NOT NULL,
    source_url   TEXT NOT NULL DEFAULT '',
    source_hash  TEXT NOT NULL,
    raw_code     BLOB
);

CREATE TABLE build_stages (
    id            TEXT PRIMARY KEY,
    build_id      TEXT NOT NULL REFERENCES builds (id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    stage_order   INTEGER NOT NULL,
    character_level INTEGER,
    association   TEXT NOT NULL DEFAULT 'none',
    notes         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_build_stages_build ON build_stages (build_id, stage_order);

-- Passive nodes per stage, split by role so opening a stage needs no diffing.
-- role: 'current' (all nodes), 'new' (added vs previous), 'removed'.
CREATE TABLE build_stage_nodes (
    stage_id TEXT NOT NULL REFERENCES build_stages (id) ON DELETE CASCADE,
    node_id  INTEGER NOT NULL,
    role     TEXT NOT NULL,
    PRIMARY KEY (stage_id, node_id, role)
);

CREATE TABLE build_skill_groups (
    id          TEXT PRIMARY KEY,
    stage_id    TEXT NOT NULL REFERENCES build_stages (id) ON DELETE CASCADE,
    label       TEXT NOT NULL DEFAULT '',
    slot        TEXT NOT NULL DEFAULT '',
    enabled     INTEGER NOT NULL DEFAULT 1,
    is_main     INTEGER NOT NULL DEFAULT 0,
    group_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_skill_groups_stage ON build_skill_groups (stage_id, group_order);

CREATE TABLE build_gems (
    group_id       TEXT NOT NULL REFERENCES build_skill_groups (id) ON DELETE CASCADE,
    gem_order      INTEGER NOT NULL,
    name           TEXT NOT NULL,
    level          INTEGER NOT NULL DEFAULT 0,
    required_level INTEGER,
    quality        INTEGER NOT NULL DEFAULT 0,
    enabled        INTEGER NOT NULL DEFAULT 1,
    is_support     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (group_id, gem_order)
);

CREATE TABLE build_progress (
    build_id      TEXT PRIMARY KEY REFERENCES builds (id) ON DELETE CASCADE,
    current_stage INTEGER NOT NULL DEFAULT 0,
    updated_at    TEXT NOT NULL
);

-- Versioned structural passive tree data, keyed by game tree version.
CREATE TABLE passive_tree_versions (
    version    TEXT PRIMARY KEY,
    min_x      REAL NOT NULL DEFAULT 0,
    min_y      REAL NOT NULL DEFAULT 0,
    max_x      REAL NOT NULL DEFAULT 0,
    max_y      REAL NOT NULL DEFAULT 0,
    imported_at TEXT NOT NULL
);

CREATE TABLE passive_tree_nodes (
    version    TEXT NOT NULL REFERENCES passive_tree_versions (version) ON DELETE CASCADE,
    node_id    INTEGER NOT NULL,
    x          REAL NOT NULL,
    y          REAL NOT NULL,
    kind       TEXT NOT NULL DEFAULT 'normal',
    name       TEXT NOT NULL DEFAULT '',
    group_id   INTEGER NOT NULL DEFAULT 0,
    is_mastery INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (version, node_id)
);

CREATE TABLE passive_tree_connections (
    version   TEXT NOT NULL REFERENCES passive_tree_versions (version) ON DELETE CASCADE,
    from_node INTEGER NOT NULL,
    to_node   INTEGER NOT NULL,
    PRIMARY KEY (version, from_node, to_node)
);

-- Persisted overlay window geometry and the tree camera per build.
CREATE TABLE window_state (
    id       INTEGER PRIMARY KEY CHECK (id = 1),
    x        INTEGER NOT NULL DEFAULT 0,
    y        INTEGER NOT NULL DEFAULT 0,
    width    INTEGER NOT NULL DEFAULT 900,
    height   INTEGER NOT NULL DEFAULT 640,
    maximized INTEGER NOT NULL DEFAULT 0
);
