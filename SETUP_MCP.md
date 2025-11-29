# Hexanorm MCP Setup

I have successfully built and verified the Hexanorm MCP server.

> [!IMPORTANT]
> I had to fix a bug in `internal/hexanorm/mcp/server.go` where the `jsonschema` struct tags were causing the server to crash on startup. The server is now healthy.

## Installation

To use this MCP server with your Claude Desktop, please add the following configuration to your `claude_desktop_config.json`:

**Config File Location:**

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

**Configuration:**

```json
{
  "mcpServers": {
    "hexanorm": {
      "command": "go",
      "args": ["run", ".", "/Users/pelayo/projects/hexanorm"]
    }
  }
}
```

After saving the file, **restart Claude Desktop** to load the new server.

## Verification

I have verified the server is working by running a local client script (`mcp_client.py`).

- **Status**: Healthy
- **Node Count**: 20
- **Tools Available**: `scaffold_feature`, `link_requirement`, `blast_radius`, `index_step_definitions`
