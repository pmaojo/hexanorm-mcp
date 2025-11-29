package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/analysis"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/config"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/domain"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/graph"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/store"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/watcher"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HexanormServer implements the MCP server interface for the Hexanorm system.
// It acts as the central hub, coordinating the Graph, Analyzer, and Watcher components,
// and exposing them via MCP tools and resources.
type HexanormServer struct {
	Graph    *graph.Graph       // The semantic graph.
	Analyzer *analysis.Analyzer // The static analyzer.
	Store    *store.Store       // The persistent store.
	Config   *config.Config     // Server configuration.
	Watcher  *watcher.Watcher   // File system watcher.
	RootDir  string             // The root directory of the analyzed codebase.
}

// NewServer initializes and returns a new MCP server instance.
// It loads configuration, initializes the database, builds the initial graph,
// and starts the file watcher.
func NewServer(rootDir string) (*mcp.Server, error) {
	cfg, err := config.LoadConfig(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v. Using defaults.\n", err)
		cfg = &config.DefaultConfig
	}

	st, err := store.NewStore(filepath.Join(rootDir, cfg.PersistenceDir))
	if err != nil {
		return nil, fmt.Errorf("failed to init store: %w", err)
	}

	g := graph.NewGraph(st)
	an := analysis.NewAnalyzer(g)

	// Scan initial root
	scanDirectory(rootDir, an)
	// Index steps
	an.IndexStepDefinitions()

	w, err := watcher.NewWatcher(rootDir, an, g, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to start file watcher: %v\n", err)
	} else {
		w.Start()
	}

	hs := &HexanormServer{
		Graph:    g,
		Analyzer: an,
		Store:    st,
		Config:   cfg,
		Watcher:  w,
		RootDir:  rootDir,
	}

	s := mcp.NewServer(&mcp.Implementation{
		Name:    "hexanorm-mcp", // Updated name
		Version: "0.0.1",
	}, &mcp.ServerOptions{})

	// Register Tools
	mcp.AddTool(s, &mcp.Tool{
		Name:        "scaffold_feature",
		Description: "Creates structure for a new feature",
	}, hs.scaffoldFeature)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "link_requirement",
		Description: "Links a file to a requirement",
	}, hs.linkRequirement)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "blast_radius",
		Description: "Analyze impact of changing a code node",
	}, hs.blastRadius)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "index_step_definitions",
		Description: "Re-index BDD step definitions",
	}, hs.indexStepDefinitions)

	// Register Resources
	s.AddResource(&mcp.Resource{
		Name: "status",
		URI:  "mcp://hexanorm/status",
	}, hs.handleStatus)

	s.AddResource(&mcp.Resource{
		Name: "violations",
		URI:  "mcp://hexanorm/violations",
	}, hs.handleViolations)

	s.AddResource(&mcp.Resource{
		Name: "live_docs",
		URI:  "mcp://hexanorm/live_docs",
	}, hs.handleLiveDocs)

	s.AddResource(&mcp.Resource{
		Name: "traceability_matrix",
		URI:  "mcp://hexanorm/traceability_matrix",
	}, hs.handleTraceability)

	return s, nil
}

func scanDirectory(root string, an *analysis.Analyzer) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "node_modules" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		content, err := os.ReadFile(path)
		if err == nil {
			an.AnalyzeFile(path, content)
		}
		return nil
	})
}

// Tool Inputs

// ScaffoldInput defines the input parameters for the scaffold_feature tool.
type ScaffoldInput struct {
	Name        string `json:"name" jsonschema:"required"`
	Description string `json:"description" jsonschema:"required"`
}

// LinkRequirementInput defines the input parameters for the link_requirement tool.
type LinkRequirementInput struct {
	FilePath string `json:"file_path" jsonschema:"required"`
	ReqID    string `json:"req_id" jsonschema:"required"`
}

// BlastRadiusInput defines the input parameters for the blast_radius tool.
type BlastRadiusInput struct {
	CodeID string `json:"code_id" jsonschema:"required"`
}

// EmptyInput defines an empty input structure for tools that require no parameters.
type EmptyInput struct{}

// Tool Handlers

func (hs *HexanormServer) scaffoldFeature(ctx context.Context, req *mcp.CallToolRequest, input ScaffoldInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Name required"}}}, nil, nil
	}

	// Create directories (simplified)
	base := filepath.Join(hs.RootDir, "src")
	dirs := []string{
		filepath.Join(base, "domain", strings.ToLower(input.Name)),
		filepath.Join(base, "domain", strings.ToLower(input.Name), "ports"),
		filepath.Join(base, "application", strings.ToLower(input.Name)),
		filepath.Join(base, "infrastructure", "adapters"),
	}

	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}

	msg := fmt.Sprintf("Scaffolded feature '%s': %s", input.Name, input.Description)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}, nil, nil
}

func (hs *HexanormServer) linkRequirement(ctx context.Context, req *mcp.CallToolRequest, input LinkRequirementInput) (*mcp.CallToolResult, any, error) {
	// Create Requirement Node if not exists
	reqNode, exists := hs.Graph.GetNode(input.ReqID)
	if !exists {
		reqNode = &domain.Node{
			ID:         input.ReqID,
			Kind:       domain.NodeKindRequirement,
			Properties: map[string]interface{}{"title": "Manually Linked Requirement"},
		}
		hs.Graph.AddNode(reqNode)
	}

	hs.Graph.AddEdge(input.ReqID, input.FilePath, domain.EdgeTypeImplementedBy)

	msg := fmt.Sprintf("Linked %s to %s", input.ReqID, input.FilePath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}, nil, nil
}

func (hs *HexanormServer) blastRadius(ctx context.Context, req *mcp.CallToolRequest, input BlastRadiusInput) (*mcp.CallToolResult, any, error) {
	features, reqs := hs.Graph.BlastRadius(input.CodeID)

	res := map[string]interface{}{
		"code_id":               input.CodeID,
		"impacted_features":     features,
		"impacted_requirements": reqs,
	}

	jsonBytes, _ := json.MarshalIndent(res, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonBytes)},
		},
	}, nil, nil
}

func (hs *HexanormServer) indexStepDefinitions(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, any, error) {
	// Re-scan? For now just re-index
	hs.Analyzer.IndexStepDefinitions()
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Indexed step definitions"},
		},
	}, nil, nil
}

// Resource Handlers

func (hs *HexanormServer) handleStatus(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	nodes := hs.Graph.GetAllNodes()
	status := map[string]interface{}{
		"node_count": len(nodes),
		"status":     "healthy",
	}
	bytes, _ := json.MarshalIndent(status, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: req.Params.URI, MIMEType: "application/json", Text: string(bytes)},
		},
	}, nil
}

func (hs *HexanormServer) handleViolations(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	violations := hs.Analyzer.FindViolations()
	bytes, _ := json.MarshalIndent(violations, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: req.Params.URI, MIMEType: "application/json", Text: string(bytes)},
		},
	}, nil
}

func (hs *HexanormServer) handleLiveDocs(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	nodes := hs.Graph.GetAllNodes()
	var sb strings.Builder
	sb.WriteString("# Hexanorm Live Docs\n\n")
	sb.WriteString("## Nodes\n")
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("- **%s** (%s)\n", n.ID, n.Kind))
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: req.Params.URI, MIMEType: "text/markdown", Text: sb.String()},
		},
	}, nil
}

func (hs *HexanormServer) handleTraceability(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Build Matrix
	// For each Requirement, find Features, Code, Tests
	matrix := []map[string]interface{}{}

	nodes := hs.Graph.GetAllNodes()
	for _, n := range nodes {
		if n.Kind == domain.NodeKindRequirement {
			entry := map[string]interface{}{
				"requirement_id": n.ID,
			}
			// Find implemented by
			edges := hs.Graph.GetEdgesFrom(n.ID)
			var code []string
			for _, e := range edges {
				if e.Type == domain.EdgeTypeImplementedBy {
					code = append(code, e.TargetID)
				}
			}
			entry["code"] = code

			// Find verifiers (Tests) - Reverse edge VERIFIES
			revEdges := hs.Graph.GetEdgesTo(n.ID)
			var verifiers []string
			for _, e := range revEdges {
				if e.Type == domain.EdgeTypeVerifies {
					verifiers = append(verifiers, e.SourceID)
				}
			}
			entry["verifiers"] = verifiers

			matrix = append(matrix, entry)
		}
	}

	bytes, _ := json.MarshalIndent(matrix, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: req.Params.URI, MIMEType: "application/json", Text: string(bytes)},
		},
	}, nil
}
