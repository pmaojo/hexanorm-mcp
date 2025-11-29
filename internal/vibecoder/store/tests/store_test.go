package tests

import (
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/domain"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/graph"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/store"
)

func TestPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vibecoder_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s, err := store.NewStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	g := graph.NewGraph(s)

	node := &domain.Node{
		ID:   "test:node:1",
		Kind: domain.NodeKindCode,
		Metadata: map[string]interface{}{
			"foo": "bar",
		},
	}
	g.AddNode(node)
	g.AddEdge("test:node:1", "test:node:2", domain.EdgeTypeImports)

	s.Close()

	// Re-open
	s2, err := store.NewStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	g2 := graph.NewGraph(s2)

	n, exists := g2.GetNode("test:node:1")
	if !exists {
		t.Error("Node not found after restart")
	}
	if n.Metadata["foo"] != "bar" {
		t.Error("Metadata mismatch")
	}

	edges := g2.GetEdgesFrom("test:node:1")
	if len(edges) != 1 {
		t.Error("Edges lost after restart")
	}
}

func TestRemoveNode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vibecoder_test_remove")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s, err := store.NewStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	g := graph.NewGraph(s)

	node1 := &domain.Node{ID: "node1", Kind: domain.NodeKindCode}
	node2 := &domain.Node{ID: "node2", Kind: domain.NodeKindCode}
	g.AddNode(node1)
	g.AddNode(node2)
	g.AddEdge("node1", "node2", domain.EdgeTypeImports)

	// Verify setup
	if len(g.GetEdgesFrom("node1")) != 1 {
		t.Fatal("Edge not added")
	}

	// Remove node1
	g.RemoveNode("node1")

	// Verify node1 gone
	if _, exists := g.GetNode("node1"); exists {
		t.Error("Node1 should be gone")
	}

	// Verify edges gone
	if len(g.GetEdgesTo("node2")) != 0 {
		t.Error("Edge to node2 should be gone")
	}

	// Verify persistence
	// Re-open store
	s2, err := store.NewStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	g2 := graph.NewGraph(s2)

	if _, exists := g2.GetNode("node1"); exists {
		t.Error("Node1 should be gone from store")
	}
}
