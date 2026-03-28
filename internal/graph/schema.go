package graph

// SchemaSQL contains the DDL for the knowledge graph database.
const SchemaSQL = `
CREATE TABLE IF NOT EXISTS nodes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    kind        TEXT    NOT NULL,
    file_path   TEXT    NOT NULL,
    line_start  INTEGER NOT NULL,
    line_end    INTEGER NOT NULL,
    col_start   INTEGER NOT NULL,
    col_end     INTEGER NOT NULL,
    scope       TEXT    NOT NULL DEFAULT '',
    signature   TEXT    NOT NULL DEFAULT '',
    exported    INTEGER NOT NULL DEFAULT 0,
    language    TEXT    NOT NULL,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_nodes_name ON nodes(name);
CREATE INDEX IF NOT EXISTS idx_nodes_file ON nodes(file_path);
CREATE INDEX IF NOT EXISTS idx_nodes_kind ON nodes(kind);
CREATE INDEX IF NOT EXISTS idx_nodes_name_kind ON nodes(name, kind);

CREATE TABLE IF NOT EXISTS edges (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id   INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    target_id   INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    kind        TEXT    NOT NULL,
    file_path   TEXT    NOT NULL,
    line        INTEGER NOT NULL,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(source_id, target_id, kind, file_path, line)
);

CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
CREATE INDEX IF NOT EXISTS idx_edges_kind ON edges(kind);
CREATE INDEX IF NOT EXISTS idx_edges_file ON edges(file_path);

CREATE TABLE IF NOT EXISTS file_metadata (
    file_path       TEXT PRIMARY KEY,
    content_hash    TEXT    NOT NULL,
    last_indexed_at TEXT    NOT NULL DEFAULT (datetime('now')),
    language        TEXT    NOT NULL,
    node_count      INTEGER NOT NULL DEFAULT 0,
    edge_count      INTEGER NOT NULL DEFAULT 0,
    index_status    TEXT    NOT NULL DEFAULT 'ok',
    error_message   TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS index_metadata (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`
