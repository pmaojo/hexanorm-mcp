package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/analysis"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/config"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/export"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/graph"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/mcp"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/store"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/tui"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// main is the entry point for the Hexanorm MCP server.
// It parses command-line arguments to determine the root directory to analyze
// and starts the MCP server over stdio.
func main() {
	// Handle CLI commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "export":
			handleExport(os.Args[2:])
			return
		case "tui":
			handleTUI(os.Args[2:])
			return
		}
	}

	// Default: Run MCP Server
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	fmt.Printf("Starting Hexanorm Server in %s...\n", root)

	// Create server
	server, err := mcp.NewServer(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Run server
	if err := server.Run(context.Background(), &sdk.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func handleExport(args []string) {
	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
	format := exportCmd.String("format", "json", "Export format (json, excalidraw)")
	out := exportCmd.String("out", "architecture.json", "Output file path")

	exportCmd.Parse(args)

	rootDir := "."
	if exportCmd.NArg() > 0 {
		rootDir = exportCmd.Arg(0)
	}

	absRoot, _ := filepath.Abs(rootDir)

	// Init Graph
	cfg, err := config.LoadConfig(absRoot)
	if err != nil {
		cfg = &config.DefaultConfig
	}
	st, err := store.NewStore(filepath.Join(absRoot, cfg.PersistenceDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init store: %v\n", err)
		os.Exit(1)
	}
	g := graph.NewGraph(st)
	an := analysis.NewAnalyzer(g)

	scanDirectory(absRoot, an)

	fmt.Printf("Exporting architecture from %s to %s (format: %s)...\n", rootDir, *out, *format)

	if *format == "excalidraw" {
		err = export.ExportExcalidraw(g, *out)
	} else {
		// Default JSON placeholder
		fmt.Println("JSON export not implemented yet")
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Export successful!")
}

func handleTUI(args []string) {
	rootDir := "."
	if len(args) > 0 {
		rootDir = args[0]
	}
	absRoot, _ := filepath.Abs(rootDir)

	// Init Graph
	cfg, err := config.LoadConfig(absRoot)
	if err != nil {
		cfg = &config.DefaultConfig
	}
	st, err := store.NewStore(filepath.Join(absRoot, cfg.PersistenceDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init store: %v\n", err)
		os.Exit(1)
	}
	g := graph.NewGraph(st)
	an := analysis.NewAnalyzer(g)

	scanDirectory(absRoot, an)

	// Start TUI
	p := tea.NewProgram(tui.NewModel(g, an), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
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
