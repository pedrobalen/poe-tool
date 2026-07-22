-- Mastery support: the effect options available on each mastery tree node, and
-- the effect a build selected per stage.

CREATE TABLE passive_tree_mastery_effects (
    version   TEXT NOT NULL REFERENCES passive_tree_versions (version) ON DELETE CASCADE,
    node_id   INTEGER NOT NULL,
    effect_id INTEGER NOT NULL,
    text      TEXT NOT NULL,
    PRIMARY KEY (version, node_id, effect_id)
);

CREATE TABLE build_stage_masteries (
    stage_id  TEXT NOT NULL REFERENCES build_stages (id) ON DELETE CASCADE,
    node_id   INTEGER NOT NULL,
    effect_id INTEGER NOT NULL,
    PRIMARY KEY (stage_id, node_id)
);
