package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/analysis"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/config"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/domain"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/graph"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VibecoderServer struct {
	Graph    *graph.Graph
	Analyzer *analysis.Analyzer
	RootDir  string
}

func NewServer(rootDir string) (*mcp.Server, error) {
	// Load Config
	cfgPath := filepath.Join(rootDir, "vibecoder.json")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		fmt.Printf("Warning: Could not load config: %v\n", err)
	}

	g := graph.NewGraph()

	// Load Persistence
	graphPath := filepath.Join(rootDir, ".vibecoder_graph.json")
	if err := g.Load(graphPath); err == nil {
		fmt.Println("Loaded existing graph.")
	}

	an := analysis.NewAnalyzer(g, cfg)

	// Scan initial root (Refresh)
	scanDirectory(rootDir, an, cfg)
	// Index steps
	an.IndexStepDefinitions()

	// Start Watcher
	go startWatcher(rootDir, an, g, graphPath)

	vs := &VibecoderServer{
		Graph:    g,
		Analyzer: an,
		RootDir:  rootDir,
	}

	s := mcp.NewServer(&mcp.Implementation{
		Name:    "vibecoder",
		Version: "0.0.1",
	}, &mcp.ServerOptions{})

	// Register Tools
	mcp.AddTool(s, &mcp.Tool{
		Name:        "scaffold_feature",
		Description: "Creates structure for a new feature",
	}, vs.scaffoldFeature)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "link_requirement",
		Description: "Links a file to a requirement",
	}, vs.linkRequirement)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "blast_radius",
		Description: "Analyze impact of changing a code node",
	}, vs.blastRadius)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "index_step_definitions",
		Description: "Re-index BDD step definitions",
	}, vs.indexStepDefinitions)

	// Register Resources
	s.AddResource(&mcp.Resource{
		Name: "status",
		URI:  "mcp://vibecoder/status",
	}, vs.handleStatus)

	s.AddResource(&mcp.Resource{
		Name: "violations",
		URI:  "mcp://vibecoder/violations",
	}, vs.handleViolations)

	s.AddResource(&mcp.Resource{
		Name: "live_docs",
		URI:  "mcp://vibecoder/live_docs",
	}, vs.handleLiveDocs)

	s.AddResource(&mcp.Resource{
		Name: "traceability_matrix",
		URI:  "mcp://vibecoder/traceability_matrix",
	}, vs.handleTraceability)

	s.AddResource(&mcp.Resource{
		Name: "architecture_diagram",
		URI:  "mcp://vibecoder/architecture_diagram",
	}, vs.handleExcalidraw)

	return s, nil
}

func scanDirectory(root string, an *analysis.Analyzer, cfg *config.Config) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			for _, ex := range cfg.Excludes {
				if info.Name() == ex {
					return filepath.SkipDir
				}
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

func startWatcher(root string, an *analysis.Analyzer, g *graph.Graph, savePath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Error creating watcher: %v\n", err)
		return
	}
	// defer watcher.Close() // Keep running

	// Add all subdirectories
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info != nil && info.IsDir() {
			if info.Name() == ".git" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			watcher.Add(path)
		}
		return nil
	})

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					watcher.Add(event.Name)
				} else {
					content, err := os.ReadFile(event.Name)
					if err == nil {
						an.AnalyzeFile(event.Name, content)
						an.IndexStepDefinitions()
						g.Save(savePath)
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("Watcher error:", err)
		}
	}
}

// Tool Inputs

type ScaffoldInput struct {
	Name        string `json:"name" jsonschema:"description=The name of the feature,required"`
	Description string `json:"description" jsonschema:"description=Description of the feature,required"`
}

type LinkRequirementInput struct {
	FilePath string `json:"file_path" jsonschema:"description=Path of the file to link,required"`
	ReqID    string `json:"req_id" jsonschema:"description=Requirement ID,required"`
}

type BlastRadiusInput struct {
	CodeID string `json:"code_id" jsonschema:"description=ID of the code node,required"`
}

type EmptyInput struct{}

// Tool Handlers

func (vs *VibecoderServer) scaffoldFeature(ctx context.Context, req *mcp.CallToolRequest, input ScaffoldInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Name required"}}}, nil, nil
	}

	// Create directories (simplified)
	base := filepath.Join(vs.RootDir, "src")
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

func (vs *VibecoderServer) linkRequirement(ctx context.Context, req *mcp.CallToolRequest, input LinkRequirementInput) (*mcp.CallToolResult, any, error) {
	// Create Requirement Node if not exists
	reqNode, exists := vs.Graph.GetNode(input.ReqID)
	if !exists {
		reqNode = &domain.Node{
			ID:         input.ReqID,
			Kind:       domain.NodeKindRequirement,
			Properties: map[string]interface{}{"title": "Manually Linked Requirement"},
		}
		vs.Graph.AddNode(reqNode)
	}

	vs.Graph.AddEdge(input.ReqID, input.FilePath, domain.EdgeTypeImplementedBy)

	// Save graph immediately to persist link
	graphPath := filepath.Join(vs.RootDir, ".vibecoder_graph.json")
	vs.Graph.Save(graphPath)

	msg := fmt.Sprintf("Linked %s to %s", input.ReqID, input.FilePath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}, nil, nil
}

func (vs *VibecoderServer) blastRadius(ctx context.Context, req *mcp.CallToolRequest, input BlastRadiusInput) (*mcp.CallToolResult, any, error) {
	features, reqs := vs.Graph.BlastRadius(input.CodeID)

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

func (vs *VibecoderServer) indexStepDefinitions(ctx context.Context, req *mcp.CallToolRequest, input EmptyInput) (*mcp.CallToolResult, any, error) {
	// Re-scan? For now just re-index
	vs.Analyzer.IndexStepDefinitions()
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Indexed step definitions"},
		},
	}, nil, nil
}

// Resource Handlers

func (vs *VibecoderServer) handleStatus(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	nodes := vs.Graph.GetAllNodes()
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

func (vs *VibecoderServer) handleExcalidraw(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	bytes, err := generateExcalidraw(vs.Graph)
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: req.Params.URI, MIMEType: "application/json", Text: string(bytes)},
		},
	}, nil
}

func (vs *VibecoderServer) handleViolations(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	violations := vs.Analyzer.FindViolations()
	bytes, _ := json.MarshalIndent(violations, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: req.Params.URI, MIMEType: "application/json", Text: string(bytes)},
		},
	}, nil
}

func (vs *VibecoderServer) handleLiveDocs(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	nodes := vs.Graph.GetAllNodes()
	var sb strings.Builder
	sb.WriteString("# Vibecoder Live Docs\n\n")
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

func (vs *VibecoderServer) handleTraceability(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Build Matrix
	// For each Requirement, find Features, Code, Tests
	matrix := []map[string]interface{}{}

	nodes := vs.Graph.GetAllNodes()
	for _, n := range nodes {
		if n.Kind == domain.NodeKindRequirement {
			entry := map[string]interface{}{
				"requirement_id": n.ID,
			}
			// Find implemented by
			edges := vs.Graph.GetEdgesFrom(n.ID)
			var code []string
			for _, e := range edges {
				if e.Type == domain.EdgeTypeImplementedBy {
					code = append(code, e.TargetID)
				}
			}
			entry["code"] = code

			// Find verifiers (Tests) - Reverse edge VERIFIES
			revEdges := vs.Graph.GetEdgesTo(n.ID)
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
