# **Hexanorm MCP — Architectural Guardian & Semantic Traceability Engine**

Hexanorm MCP is an **MCP (Model Context Protocol) server** designed to serve as an architectural sentinel for modern software systems.
It combines:

- **Static Analysis**
- **Semantic Graph Modeling**
- **Hexagonal Architecture Enforcement**
- **BDD Traceability**
- **Impact Analysis (Blast Radius)**
- **LLM-Friendly MCP Resources & Tools**

Its purpose is to allow an AI agent (Claude, Gemini, GPT) to reason over the **intent**, **structure** and **behavior** of a codebase—what Martin Fowler describes as “_making architecture visible_”—while maintaining the rigor of **Domain-Driven Design** (Evans, 2004) and **Specification by Example** (Adzic, 2012).

---

## 📘 **1. Purpose**

Hexanorm constructs a **Semantic Knowledge Graph** of the project.
This graph models:

- **Requirements** (external intent)
- **Features** (logical modules)
- **Code Elements** (classes, functions, files)
- **Tests** (unit tests, integration tests, BDD scenarios)
- **Gherkin Specifications** (Given/When/Then semantics)

This enables an LLM to:

### ✔ Identify architecture violations

“Domain layer depends on Infrastructure” → _Critical_.

### ✔ Detect BDD inconsistencies (BDD Drift)

Scenario changed but Step Definition did not.

### ✔ Trace requirements to implementation (Golden Thread)

REQ → Feature → Code → Test.

### ✔ Compute functional _blast radius_

“What requirements and scenarios might break if I modify this file?”

### ✔ Guide refactorings with structural insight

e.g., “Move this file from Infrastructure → Application”.

---

## 🧠 **2. Core Concepts**

### **2.1 Semantic Graph Model**

Each entity (Requirement, Feature, Code, Test, Scenario, StepDefinition) becomes a **typed node**, following a polymorphic schema:

```json
{
  "id": "code:src/domain/User.ts",
  "kind": "Code",
  "labels": ["Domain", "Entity"],
  "properties": {
    "name": "User",
    "filepath": "src/domain/User.ts"
  },
  "metadata": {
    "layer": "domain",
    "status": "OK"
  }
}
```

Edges represent semantic relationships:

- `DEFINES`
- `IMPLEMENTED_BY`
- `VERIFIES`
- `EXECUTES`
- `CALLS`

This is the **Golden Thread**.

---

## 🏛️ **3. Features**

### **3.1 Architecture Analysis**

Hexanorm enforces the rules described by Alistair Cockburn (Hexagonal Architecture):

- **Domain** → can import nothing but domain
- **Application** → can depend on Domain and Ports
- **Infrastructure** → can depend on anything
- **Interface/Adapter** → binds the outside world

AST parsing (via **Tree-sitter**) provides precise import and dependency extraction.

Violations are reported as structured objects:

```json
{
  "severity": "CRITICAL",
  "message": "Domain Rule Broken: User.ts imports S3Bucket (Infrastructure)",
  "file": "src/domain/User.ts",
  "kind": "ARCH_LAYER_VIOLATION"
}
```

---

### **3.2 BDD Traceability & Drift Detection**

Hexanorm parses:

- `.feature` files → `GherkinFeature`, `GherkinScenario`
- Step Definitions in code → via AST (`@Given`, `@When`, `@Then` patterns)
- Links Scenarios → Step Definitions → Code → Requirements

It can detect:

#### **BDD Drift**

When the Gherkin text changes (step text hash mismatch) but StepDefinition does not.

This implements the consistency layer described in _BDD in Action_ (Smart, 2014).

---

### **3.3 Blast Radius Analysis**

Given any code element:

```json
blast_radius("src/domain/VatService.ts")
```

Hexanorm returns all potentially impacted nodes:

- Features using it
- Requirements implemented by it
- Gherkin Scenarios that indirectly execute code paths touching it

This converts architectural impact into a queryable structure—what NASA’s IV&V facility calls _Functional Integrity_.

---

### **3.4 File Watching / Real-Time Updates**

Using `fsnotify`, Hexanorm updates:

- parsed AST
- graph nodes
- violations
- traceability matrix

As soon as the developer saves a file.

This achieves “active architectural governance”.

---

## 🚀 **4. Usage**

### **4.1 Installation**

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/modelcontextprotocol/go-sdk.git
    cd examples/server/hexanorm
    ```

2.  **Build the server:**
    ```bash
    go build -o hexanorm-server
    ```

### **4.2 Running the MCP Server**

```bash
go run . /path/to/project
```

Hexanorm runs on STDIO and integrates with any MCP client (Claude Desktop, model servers, agent runtimes).

---

## 🧩 **5. Integration with Claude Desktop**

Add to `claude_desktop_config.json`:

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

_Note: Replace `/absolute/path/to/target/project` with the directory you want to analyze._

---

## 🛠️ **6. Available Tools**

| Tool                       | Purpose                                             |
| -------------------------- | --------------------------------------------------- |
| **scaffold_feature**       | Generates full Hexagonal skeleton for a new feature |
| **link_requirement**       | Manually link Code → Requirement                    |
| **blast_radius**           | Query impact analysis                               |
| **index_step_definitions** | Parse and rebuild BDD step definitions              |

---

## 📡 **7. Available Resources**

| Resource                             | Description                            |
| ------------------------------------ | -------------------------------------- |
| `mcp://hexanorm/status`              | Health of graph + node counts          |
| `mcp://hexanorm/violations`          | All architecture + BDD violations      |
| `mcp://hexanorm/traceability_matrix` | Full Golden Thread map                 |
| `mcp://hexanorm/live_docs`           | Markdown documentation of architecture |

---

## 🎨 **8. Excalidraw Visualization** (Not working. WIP)

Hexanorm bridges the gap between code and diagrams by allowing you to **export your architecture directly to Excalidraw**.

### **Features**

- **Auto-Layout**: Automatically arranges your Domain, Application, and Infrastructure layers.
- **Live Sync**: Changes in code are reflected in the diagram (via re-export).
- **Semantic Coloring**:
  - 🟦 **Domain**: Blue (Core Logic)
  - 🟩 **Application**: Green (Use Cases)
  - 🟨 **Infrastructure**: Yellow (Adapters)
  - 🟥 **Violations**: Red (Illegal Dependencies)

### **Usage**

To generate a diagram:

```bash
hexanorm export --format=excalidraw --out=architecture.excalidraw
```

You can then open `architecture.excalidraw` in [excalidraw.com](https://excalidraw.com) or the VS Code Excalidraw extension to visually inspect your system's "Golden Thread".

---

## 🧭 **9. Bibliographic Foundations**

Hexanorm’s conceptual design aligns with:

- Eric Evans — _Domain-Driven Design_ (2004)
- Alistair Cockburn — _Hexagonal Architecture_ (2005)
- Jez Humble — _Continuous Delivery_ (2011)
- Gojko Adzic — _Specification by Example_ (2012)
- Sam Newman — _Building Microservices_ (2015)

---

## 🎯 **9. Summary**

Hexanorm MCP transforms architecture into a **live, queryable knowledge graph**.
It allows LLM agents to:

- understand intent,
- enforce structure,
- verify behavior,
- detect regressions,
- and guide developers through complex change operations.

It is an _intelligent architectural guardian_, a "Cognitive Linter", and a bridge between **human architecture** and **automated reasoning**.
