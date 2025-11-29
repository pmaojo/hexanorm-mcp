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

### Installation

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/modelcontextprotocol/go-sdk.git
    cd examples/server/hexanorm
    ```

2.  **Build the server:**
    ```bash
    go build -o hexanorm-server
    ```

### Configuration (Optional)

You can configure the analysis by creating a `hexanorm.json` file in your project root:

```json
{
  "excluded_dirs": ["node_modules", "dist", ".git"],
  "included_layers": ["domain", "application", "infrastructure", "interface"],
  "persistence_dir": ".hexanorm"
}
```

## Usage

### Integration with Claude Desktop

To use Hexanorm with Claude Desktop, add the following to your `claude_desktop_config.json`:

**Config File Location:**

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

**Configuration:**

```json
{
  "mcpServers": {
    "hexanorm": {
      "command": "/absolute/path/to/hexanorm-server",
      "args": ["/absolute/path/to/target/project"]
    }
  }
}
```

_Note: You can also use `go run .` as the command if you prefer not to build the binary._

Replace `/absolute/path/to/target/project` with the directory you want to analyze.

### Available Tools

Once connected, the following tools are available to the LLM:

- `scaffold_feature`: Creates a directory structure for a new feature (DDD style).
- `link_requirement`: Manually links a requirement ID to a file.
- `blast_radius`: Analyzes the downstream impact of changing a specific code node.
- `index_step_definitions`: Re-indexes BDD step definitions.

### Available Resources

- `mcp://hexanorm/status`: Current health and node count.
- `mcp://hexanorm/violations`: List of architectural and BDD violations.
- `mcp://hexanorm/traceability_matrix`: Matrix showing relationships between requirements, code, and tests.
- `mcp://hexanorm/live_docs`: Markdown representation of the current graph nodes.
