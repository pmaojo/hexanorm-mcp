package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/domain"
)

// Store handles persistence of the semantic graph using SQLite.
type Store struct {
	db *sql.DB
}

// NewStore initializes a new Store in the specified storage directory.
// It creates the directory if it doesn't exist and opens/creates 'vibecoder.db'.
// It also initializes the database schema if needed.
func NewStore(storageDir string) (*Store, error) {
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage dir: %w", err)
	}

	dbPath := filepath.Join(storageDir, "vibecoder.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// initSchema creates the necessary tables and indexes if they do not exist.
func (s *Store) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS nodes (
			id TEXT PRIMARY KEY,
			kind TEXT,
			properties TEXT,
			metadata TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS edges (
			source_id TEXT,
			target_id TEXT,
			type TEXT,
			PRIMARY KEY (source_id, target_id, type)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);`,
		`CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("failed to exec schema query: %w", err)
		}
	}
	return nil
}

// SaveNode persists a node to the database.
// It performs an UPSERT (insert or update on conflict) operation.
func (s *Store) SaveNode(node *domain.Node) error {
	props, _ := json.Marshal(node.Properties)
	meta, _ := json.Marshal(node.Metadata)

	_, err := s.db.Exec(`
		INSERT INTO nodes (id, kind, properties, metadata)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kind=excluded.kind,
			properties=excluded.properties,
			metadata=excluded.metadata;
	`, node.ID, node.Kind, string(props), string(meta))
	return err
}

// DeleteNode removes a node and all its connected edges (cascading delete) from the database.
func (s *Store) DeleteNode(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM nodes WHERE id = ?", id); err != nil {
		return err
	}
	// Cascade delete edges
	if _, err := tx.Exec("DELETE FROM edges WHERE source_id = ? OR target_id = ?", id, id); err != nil {
		return err
	}

	return tx.Commit()
}

// SaveEdge persists an edge to the database.
// It ignores the operation if the edge already exists.
func (s *Store) SaveEdge(edge *domain.Edge) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO edges (source_id, target_id, type)
		VALUES (?, ?, ?)
	`, edge.SourceID, edge.TargetID, edge.Type)
	return err
}

// LoadAll retrieves all nodes and edges from the database.
// It returns a slice of Nodes and a slice of Edges, or an error if the query fails.
func (s *Store) LoadAll() ([]*domain.Node, []*domain.Edge, error) {
	// Load Nodes
	rows, err := s.db.Query("SELECT id, kind, properties, metadata FROM nodes")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var nodes []*domain.Node
	for rows.Next() {
		var id, kind, propsStr, metaStr string
		if err := rows.Scan(&id, &kind, &propsStr, &metaStr); err != nil {
			return nil, nil, err
		}

		node := &domain.Node{
			ID:   id,
			Kind: domain.NodeKind(kind),
		}
		if propsStr != "" {
			json.Unmarshal([]byte(propsStr), &node.Properties)
		}
		if metaStr != "" {
			json.Unmarshal([]byte(metaStr), &node.Metadata)
		}
		nodes = append(nodes, node)
	}

	// Load Edges
	edgeRows, err := s.db.Query("SELECT source_id, target_id, type FROM edges")
	if err != nil {
		return nil, nil, err
	}
	defer edgeRows.Close()

	var edges []*domain.Edge
	for edgeRows.Next() {
		var src, tgt, typ string
		if err := edgeRows.Scan(&src, &tgt, &typ); err != nil {
			return nil, nil, err
		}
		edges = append(edges, &domain.Edge{
			SourceID: src,
			TargetID: tgt,
			Type:     domain.EdgeType(typ),
		})
	}

	return nodes, edges, nil
}
