# Hexanorm MCP

Hexanorm MCP provides a specialized environment for analyzing codebase architecture, enforcing layering rules, and tracing Behavior-Driven Development (BDD) relationships.

## Purpose

Hexanorm can expose advanced code analysis capabilities to LLMs (Large Language Models). It builds a semantic graph of the codebase that links:
- **Requirements**
- **Features**
- **Code** (Functions, Classes, Files)
- **Tests** (Unit tests, Gherkin Scenarios)

By maintaining this graph, Hexanorm allows LLMs to query the "blast radius" of changes, verify architectural constraints (e.g., Domain layer should not import Infrastructure), and ensure BDD scenarios are implemented.

## Features

### 1. Architecture Analysis
Hexanorm scans source files to detect their architectural layer (`domain`, `application`, `infrastructure`, `interface`). It parses imports to build a dependency graph and detects violations of strict layering rules (e.g., Clean Architecture or Hexagonal Architecture principles).

### 2. BDD Traceability
It parses Gherkin feature files (`.feature`) and matches them against code that implements step definitions. It can identify "BDD Drift" where scenarios exist without corresponding code implementation.

### 3. Blast Radius Calculation
The server exposes a tool to calculate the impact of changing a specific piece of code, tracing dependencies back to the features and requirements that might be affected.

### 4. File Watching
It uses `fsnotify` to incrementally update the analysis graph as files are created, modified, or deleted in the workspace.

## Setup

### Prerequisites
- Go 1.23 or later

## Usage

### Running the Server
You can run the server directly using Go:

```bash
go run . /path/to/your/project
```

The server runs over stdio, making it compatible with MCP clients like Claude Desktop.

### Integration with Claude Desktop
To use Hexanorm with Claude Desktop, add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "hexanorm": {
      "command": "go",
      "args": ["run", ".", "/absolute/path/to/target/project"]
    }
  }
}
```

Replace `/absolute/path/to/target/project` with the directory you want to analyze.

### Available Tools
Once connected, the following tools are available to the LLM:
- `scaffold_feature`: Creates a directory structure for a new feature (DDD style).
- `link_requirement`: Manually links a requirement ID to a file.
- `blast_radius`: Analyzes the downstream impact of changing a specific code node.
- `index_step_definitions`: Re-indexes BDD step definitions.

### Available Resources
- `mcp://vibecoder/status`: Current health and node count.
- `mcp://vibecoder/violations`: List of architectural and BDD violations.
- `mcp://vibecoder/traceability_matrix`: Matrix showing relationships between requirements, code, and tests.
- `mcp://vibecoder/live_docs`: Markdown representation of the current graph nodes.

*Note: The resource URIs currently use the `vibecoder` scheme for compatibility.*
