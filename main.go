package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// main is the entry point for the Hexanorm MCP server.
// It parses command-line arguments to determine the root directory to analyze
// and starts the MCP server over stdio.
func main() {
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
