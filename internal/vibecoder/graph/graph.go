package graph

import (
	"sync"

	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/domain"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/store"
)

type Graph struct {
	mu           sync.RWMutex
	nodes        map[string]*domain.Node
	edges        map[string][]*domain.Edge // SourceID -> Edges
	reverseEdges map[string][]*domain.Edge // TargetID -> Edges
	store        *store.Store
}

func NewGraph(s *store.Store) *Graph {
	g := &Graph{
		nodes:        make(map[string]*domain.Node),
		edges:        make(map[string][]*domain.Edge),
		reverseEdges: make(map[string][]*domain.Edge),
		store:        s,
	}
	if s != nil {
		g.loadFromStore()
	}
	return g
}

func (g *Graph) loadFromStore() error {
	nodes, edges, err := g.store.LoadAll()
	if err != nil {
		return err
	}
	for _, n := range nodes {
		g.nodes[n.ID] = n
	}
	for _, e := range edges {
		g.addEdgeInternal(e)
	}
	return nil
}

func (g *Graph) AddNode(node *domain.Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes[node.ID] = node
	if g.store != nil {
		g.store.SaveNode(node)
	}
}

func (g *Graph) RemoveNode(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[id]; !exists {
		return
	}
	delete(g.nodes, id)

	// 1. Remove edges where this node is Source
	// For each outgoing edge, remove it from the Target's reverseEdges
	if outgoing, ok := g.edges[id]; ok {
		for _, edge := range outgoing {
			g.removeReverseEdge(edge.TargetID, id)
		}
		delete(g.edges, id)
	}

	// 2. Remove edges where this node is Target
	// For each incoming edge, remove it from the Source's edges
	if incoming, ok := g.reverseEdges[id]; ok {
		for _, edge := range incoming {
			g.removeForwardEdge(edge.SourceID, id)
		}
		delete(g.reverseEdges, id)
	}

	// 3. Persist deletion
	if g.store != nil {
		g.store.DeleteNode(id)
	}
}

func (g *Graph) removeForwardEdge(sourceID, targetID string) {
	edges := g.edges[sourceID]
	newEdges := edges[:0]
	for _, e := range edges {
		if e.TargetID != targetID {
			newEdges = append(newEdges, e)
		}
	}
	if len(newEdges) == 0 {
		delete(g.edges, sourceID)
	} else {
		g.edges[sourceID] = newEdges
	}
}

func (g *Graph) removeReverseEdge(targetID, sourceID string) {
	edges := g.reverseEdges[targetID]
	newEdges := edges[:0]
	for _, e := range edges {
		if e.SourceID != sourceID {
			newEdges = append(newEdges, e)
		}
	}
	if len(newEdges) == 0 {
		delete(g.reverseEdges, targetID)
	} else {
		g.reverseEdges[targetID] = newEdges
	}
}

func (g *Graph) GetNode(id string) (*domain.Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.nodes[id]
	return n, ok
}

func (g *Graph) GetAllNodes() []*domain.Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	nodes := make([]*domain.Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

func (g *Graph) AddEdge(sourceID, targetID string, edgeType domain.EdgeType) {
	g.mu.Lock()
	defer g.mu.Unlock()

	edge := &domain.Edge{
		SourceID: sourceID,
		TargetID: targetID,
		Type:     edgeType,
	}

	if g.addEdgeInternal(edge) {
		if g.store != nil {
			g.store.SaveEdge(edge)
		}
	}
}

func (g *Graph) addEdgeInternal(edge *domain.Edge) bool {
	// Avoid duplicates
	for _, e := range g.edges[edge.SourceID] {
		if e.TargetID == edge.TargetID && e.Type == edge.Type {
			return false
		}
	}

	g.edges[edge.SourceID] = append(g.edges[edge.SourceID], edge)
	g.reverseEdges[edge.TargetID] = append(g.reverseEdges[edge.TargetID], edge)
	return true
}

func (g *Graph) GetEdgesFrom(sourceID string) []*domain.Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	// Return copy
	edges := g.edges[sourceID]
	result := make([]*domain.Edge, len(edges))
	copy(result, edges)
	return result
}

func (g *Graph) GetEdgesTo(targetID string) []*domain.Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	edges := g.reverseEdges[targetID]
	result := make([]*domain.Edge, len(edges))
	copy(result, edges)
	return result
}

// BlastRadius calculates impacted features and requirements given a code node ID.
func (g *Graph) BlastRadius(codeID string) ([]string, []string) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	queue := []string{codeID}

	impactedFeatures := make(map[string]bool)
	impactedRequirements := make(map[string]bool)

	visited[codeID] = true

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		for _, edge := range g.reverseEdges[currentID] {
			if !visited[edge.SourceID] {
				sourceNode, exists := g.nodes[edge.SourceID]
				if !exists {
					continue
				}

				if edge.Type == domain.EdgeTypeImplementedBy ||
					edge.Type == domain.EdgeTypeDefines ||
					edge.Type == domain.EdgeTypeCalls {

					visited[edge.SourceID] = true
					queue = append(queue, edge.SourceID)

					if sourceNode.Kind == domain.NodeKindFeature {
						impactedFeatures[sourceNode.ID] = true
					}
					if sourceNode.Kind == domain.NodeKindRequirement {
						impactedRequirements[sourceNode.ID] = true
					}
				}
			}
		}
	}

	features := make([]string, 0, len(impactedFeatures))
	for k := range impactedFeatures {
		features = append(features, k)
	}
	requirements := make([]string, 0, len(impactedRequirements))
	for k := range impactedRequirements {
		requirements = append(requirements, k)
	}

	return features, requirements
}

func (g *Graph) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes = make(map[string]*domain.Node)
	g.edges = make(map[string][]*domain.Edge)
	g.reverseEdges = make(map[string][]*domain.Edge)
	// Warning: Does not clear Store.
}
