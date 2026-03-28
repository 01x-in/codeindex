package graph

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using modernc.org/sqlite (pure Go).
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or creates a SQLite database at the given path.
// Use ":memory:" for an in-memory database (useful for tests).
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	// Enable WAL mode and foreign keys.
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, fmt.Errorf("setting pragma %q: %w", p, err)
		}
	}

	return &SQLiteStore{db: db}, nil
}

// Migrate creates the schema tables if they don't exist.
func (s *SQLiteStore) Migrate() error {
	_, err := s.db.Exec(SchemaSQL)
	if err != nil {
		return fmt.Errorf("running schema migration: %w", err)
	}

	// Set schema version if not already set.
	_, err = s.db.Exec(`INSERT OR IGNORE INTO index_metadata (key, value) VALUES ('schema_version', '1')`)
	return err
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// UpsertNode inserts or updates a node, returning the node ID.
func (s *SQLiteStore) UpsertNode(node Node) (int64, error) {
	exported := 0
	if node.Exported {
		exported = 1
	}

	result, err := s.db.Exec(`
		INSERT INTO nodes (name, kind, file_path, line_start, line_end, col_start, col_end, scope, signature, exported, language)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO UPDATE SET
			line_start = excluded.line_start,
			line_end = excluded.line_end,
			col_start = excluded.col_start,
			col_end = excluded.col_end,
			scope = excluded.scope,
			signature = excluded.signature,
			exported = excluded.exported,
			updated_at = datetime('now')
	`, node.Name, node.Kind, node.FilePath, node.LineStart, node.LineEnd,
		node.ColStart, node.ColEnd, node.Scope, node.Signature, exported, node.Language)
	if err != nil {
		return 0, fmt.Errorf("upserting node: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting last insert id: %w", err)
	}
	return id, nil
}

// UpsertEdge inserts an edge, ignoring duplicates.
func (s *SQLiteStore) UpsertEdge(edge Edge) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO edges (source_id, target_id, kind, file_path, line)
		VALUES (?, ?, ?, ?, ?)
	`, edge.SourceID, edge.TargetID, edge.Kind, edge.FilePath, edge.Line)
	if err != nil {
		return fmt.Errorf("upserting edge: %w", err)
	}
	return nil
}

// SetFileMetadata upserts file metadata.
func (s *SQLiteStore) SetFileMetadata(meta FileMetadata) error {
	_, err := s.db.Exec(`
		INSERT INTO file_metadata (file_path, content_hash, language, node_count, edge_count, index_status, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			content_hash = excluded.content_hash,
			last_indexed_at = datetime('now'),
			language = excluded.language,
			node_count = excluded.node_count,
			edge_count = excluded.edge_count,
			index_status = excluded.index_status,
			error_message = excluded.error_message
	`, meta.FilePath, meta.ContentHash, meta.Language, meta.NodeCount, meta.EdgeCount, meta.IndexStatus, meta.ErrorMessage)
	if err != nil {
		return fmt.Errorf("setting file metadata: %w", err)
	}
	return nil
}

// SetIndexMetadata sets a key-value pair in the index_metadata table.
func (s *SQLiteStore) SetIndexMetadata(key string, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO index_metadata (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("setting index metadata %q: %w", key, err)
	}
	return nil
}

// GetIndexMetadata gets a value from the index_metadata table.
func (s *SQLiteStore) GetIndexMetadata(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM index_metadata WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("getting index metadata %q: %w", key, err)
	}
	return value, nil
}

// DeleteFileData removes all nodes, edges, and metadata for a file.
func (s *SQLiteStore) DeleteFileData(filePath string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete edges referencing nodes in this file.
	_, err = tx.Exec(`
		DELETE FROM edges WHERE source_id IN (SELECT id FROM nodes WHERE file_path = ?)
		   OR target_id IN (SELECT id FROM nodes WHERE file_path = ?)
	`, filePath, filePath)
	if err != nil {
		return fmt.Errorf("deleting edges: %w", err)
	}

	// Delete nodes in this file.
	_, err = tx.Exec(`DELETE FROM nodes WHERE file_path = ?`, filePath)
	if err != nil {
		return fmt.Errorf("deleting nodes: %w", err)
	}

	// Delete file metadata.
	_, err = tx.Exec(`DELETE FROM file_metadata WHERE file_path = ?`, filePath)
	if err != nil {
		return fmt.Errorf("deleting file metadata: %w", err)
	}

	return tx.Commit()
}

// GetNode retrieves a node by ID.
func (s *SQLiteStore) GetNode(id int64) (Node, error) {
	var n Node
	var exported int
	var createdAt, updatedAt string
	err := s.db.QueryRow(`
		SELECT id, name, kind, file_path, line_start, line_end, col_start, col_end,
			   scope, signature, exported, language, created_at, updated_at
		FROM nodes WHERE id = ?
	`, id).Scan(&n.ID, &n.Name, &n.Kind, &n.FilePath, &n.LineStart, &n.LineEnd,
		&n.ColStart, &n.ColEnd, &n.Scope, &n.Signature, &exported, &n.Language,
		&createdAt, &updatedAt)
	if err != nil {
		return Node{}, fmt.Errorf("getting node %d: %w", id, err)
	}
	n.Exported = exported == 1
	n.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	n.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return n, nil
}

// FindNodesByName finds all nodes with the given name.
func (s *SQLiteStore) FindNodesByName(name string) ([]Node, error) {
	rows, err := s.db.Query(`
		SELECT id, name, kind, file_path, line_start, line_end, col_start, col_end,
			   scope, signature, exported, language, created_at, updated_at
		FROM nodes WHERE name = ?
	`, name)
	if err != nil {
		return nil, fmt.Errorf("finding nodes by name: %w", err)
	}
	defer rows.Close()
	return scanNodes(rows)
}

// FindNodesByFile finds all nodes in the given file.
func (s *SQLiteStore) FindNodesByFile(filePath string) ([]Node, error) {
	rows, err := s.db.Query(`
		SELECT id, name, kind, file_path, line_start, line_end, col_start, col_end,
			   scope, signature, exported, language, created_at, updated_at
		FROM nodes WHERE file_path = ?
	`, filePath)
	if err != nil {
		return nil, fmt.Errorf("finding nodes by file: %w", err)
	}
	defer rows.Close()
	return scanNodes(rows)
}

// GetEdgesFrom returns edges originating from the given node.
// If kind is empty, returns all edge kinds.
func (s *SQLiteStore) GetEdgesFrom(nodeID int64, kind string) ([]Edge, error) {
	var query string
	var args []interface{}
	if kind == "" {
		query = `SELECT id, source_id, target_id, kind, file_path, line, created_at FROM edges WHERE source_id = ?`
		args = []interface{}{nodeID}
	} else {
		query = `SELECT id, source_id, target_id, kind, file_path, line, created_at FROM edges WHERE source_id = ? AND kind = ?`
		args = []interface{}{nodeID, kind}
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("getting edges from node %d: %w", nodeID, err)
	}
	defer rows.Close()
	return scanEdges(rows)
}

// GetEdgesTo returns edges pointing to the given node.
// If kind is empty, returns all edge kinds.
func (s *SQLiteStore) GetEdgesTo(nodeID int64, kind string) ([]Edge, error) {
	var query string
	var args []interface{}
	if kind == "" {
		query = `SELECT id, source_id, target_id, kind, file_path, line, created_at FROM edges WHERE target_id = ?`
		args = []interface{}{nodeID}
	} else {
		query = `SELECT id, source_id, target_id, kind, file_path, line, created_at FROM edges WHERE target_id = ? AND kind = ?`
		args = []interface{}{nodeID, kind}
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("getting edges to node %d: %w", nodeID, err)
	}
	defer rows.Close()
	return scanEdges(rows)
}

// GetFileMetadata returns metadata for a specific file.
func (s *SQLiteStore) GetFileMetadata(filePath string) (FileMetadata, error) {
	var m FileMetadata
	var lastIndexed string
	err := s.db.QueryRow(`
		SELECT file_path, content_hash, last_indexed_at, language, node_count, edge_count, index_status, error_message
		FROM file_metadata WHERE file_path = ?
	`, filePath).Scan(&m.FilePath, &m.ContentHash, &lastIndexed, &m.Language,
		&m.NodeCount, &m.EdgeCount, &m.IndexStatus, &m.ErrorMessage)
	if err != nil {
		return FileMetadata{}, fmt.Errorf("getting file metadata for %q: %w", filePath, err)
	}
	m.LastIndexedAt, _ = time.Parse("2006-01-02 15:04:05", lastIndexed)
	return m, nil
}

// GetAllFileMetadata returns metadata for all indexed files.
func (s *SQLiteStore) GetAllFileMetadata() ([]FileMetadata, error) {
	rows, err := s.db.Query(`
		SELECT file_path, content_hash, last_indexed_at, language, node_count, edge_count, index_status, error_message
		FROM file_metadata ORDER BY file_path
	`)
	if err != nil {
		return nil, fmt.Errorf("getting all file metadata: %w", err)
	}
	defer rows.Close()

	var results []FileMetadata
	for rows.Next() {
		var m FileMetadata
		var lastIndexed string
		if err := rows.Scan(&m.FilePath, &m.ContentHash, &lastIndexed, &m.Language,
			&m.NodeCount, &m.EdgeCount, &m.IndexStatus, &m.ErrorMessage); err != nil {
			return nil, fmt.Errorf("scanning file metadata: %w", err)
		}
		m.LastIndexedAt, _ = time.Parse("2006-01-02 15:04:05", lastIndexed)
		results = append(results, m)
	}
	return results, rows.Err()
}

// GetNeighborhood retrieves nodes and edges within a bounded neighborhood.
func (s *SQLiteStore) GetNeighborhood(nodeID int64, depth int, edgeKinds []string) ([]Node, []Edge, error) {
	if depth < 1 {
		depth = 1
	}
	if depth > 10 {
		depth = 10
	}

	visited := map[int64]bool{nodeID: true}
	seenEdges := map[int64]bool{}
	frontier := []int64{nodeID}
	var allEdges []Edge

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var nextFrontier []int64
		for _, nid := range frontier {
			outEdges, err := s.getFilteredEdgesFrom(nid, edgeKinds)
			if err != nil {
				return nil, nil, err
			}
			for _, e := range outEdges {
				if !seenEdges[e.ID] {
					seenEdges[e.ID] = true
					allEdges = append(allEdges, e)
				}
				if !visited[e.TargetID] {
					visited[e.TargetID] = true
					nextFrontier = append(nextFrontier, e.TargetID)
				}
			}

			inEdges, err := s.getFilteredEdgesTo(nid, edgeKinds)
			if err != nil {
				return nil, nil, err
			}
			for _, e := range inEdges {
				if !seenEdges[e.ID] {
					seenEdges[e.ID] = true
					allEdges = append(allEdges, e)
				}
				if !visited[e.SourceID] {
					visited[e.SourceID] = true
					nextFrontier = append(nextFrontier, e.SourceID)
				}
			}
		}
		frontier = nextFrontier
	}

	var nodes []Node
	for nid := range visited {
		node, err := s.GetNode(nid)
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, allEdges, nil
}

// NodeCount returns the total number of nodes in the graph.
func (s *SQLiteStore) NodeCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&count)
	return count, err
}

// EdgeCount returns the total number of edges in the graph.
func (s *SQLiteStore) EdgeCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM edges`).Scan(&count)
	return count, err
}

func (s *SQLiteStore) getFilteredEdgesFrom(nodeID int64, edgeKinds []string) ([]Edge, error) {
	if len(edgeKinds) == 0 {
		return s.GetEdgesFrom(nodeID, "")
	}
	var all []Edge
	for _, kind := range edgeKinds {
		edges, err := s.GetEdgesFrom(nodeID, kind)
		if err != nil {
			return nil, err
		}
		all = append(all, edges...)
	}
	return all, nil
}

func (s *SQLiteStore) getFilteredEdgesTo(nodeID int64, edgeKinds []string) ([]Edge, error) {
	if len(edgeKinds) == 0 {
		return s.GetEdgesTo(nodeID, "")
	}
	var all []Edge
	for _, kind := range edgeKinds {
		edges, err := s.GetEdgesTo(nodeID, kind)
		if err != nil {
			return nil, err
		}
		all = append(all, edges...)
	}
	return all, nil
}

func scanNodes(rows *sql.Rows) ([]Node, error) {
	var nodes []Node
	for rows.Next() {
		var n Node
		var exported int
		var createdAt, updatedAt string
		if err := rows.Scan(&n.ID, &n.Name, &n.Kind, &n.FilePath, &n.LineStart, &n.LineEnd,
			&n.ColStart, &n.ColEnd, &n.Scope, &n.Signature, &exported, &n.Language,
			&createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning node: %w", err)
		}
		n.Exported = exported == 1
		n.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		n.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func scanEdges(rows *sql.Rows) ([]Edge, error) {
	var edges []Edge
	for rows.Next() {
		var e Edge
		var createdAt string
		if err := rows.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.Kind, &e.FilePath, &e.Line, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning edge: %w", err)
		}
		e.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

// placeholders generates a comma-separated list of ? placeholders.
func placeholders(n int) string {
	p := make([]string, n)
	for i := range p {
		p[i] = "?"
	}
	return strings.Join(p, ",")
}

// Ensure SQLiteStore satisfies the Store interface at compile time.
var _ Store = (*SQLiteStore)(nil)
