package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/analysis"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/config"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/export"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/graph"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/mcp"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/store"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// main is the entry point for the Hexanorm MCP server.
// It parses command-line arguments to determine the root directory to analyze
// and starts the MCP server over stdio.
func main() {
	if len(os.Args) > 1 && os.Args[1] == "export" {
		handleExport()
		return
	}

	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}

	fmt.Printf("Starting Hexanorm Server in %s...\n", rootDir)

	// Create server
	server, err := mcp.NewServer(rootDir)
	if err != nil {
		log.Fatal(err)
	}

	// Run server
	if err := server.Run(context.Background(), &sdk.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func handleExport() {
	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
	format := exportCmd.String("format", "excalidraw", "Output format (excalidraw)")
	out := exportCmd.String("out", "architecture.excalidraw", "Output file path")
	
	if len(os.Args) < 3 {
		fmt.Println("Usage: hexanorm export --format=excalidraw --out=file.excalidraw [rootDir]")
		os.Exit(1)
	}

	exportCmd.Parse(os.Args[2:])
	
	args := exportCmd.Args()
	rootDir := "."
	if len(args) > 0 {
		rootDir = args[0]
	}

	fmt.Printf("Exporting architecture from %s to %s (format: %s)...\n", rootDir, *out, *format)

	// Initialize components (similar to NewServer but lightweight)
	cfg, err := config.LoadConfig(rootDir)
	if err != nil {
		cfg = &config.DefaultConfig
	}

	st, err := store.NewStore(filepath.Join(rootDir, cfg.PersistenceDir))
	if err != nil {
		log.Fatalf("Failed to init store: %v", err)
	}
	defer st.Close()

	g := graph.NewGraph(st)
	
	// If store is empty, we might want to scan? 
	// For now assume store has data or user ran server once.
	// Actually, let's scan to be safe/useful for CI usage.
	an := analysis.NewAnalyzer(g)
	scanDirectory(rootDir, an)

	if *format == "excalidraw" {
		if err := export.ExportExcalidraw(g, *out); err != nil {
			log.Fatalf("Export failed: %v", err)
		}
		fmt.Println("Export successful!")
	} else {
		log.Fatalf("Unknown format: %s", *format)
	}
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
