package domain

// NodeKind represents the type of a node in the semantic graph.
type NodeKind string

// Constants defining the various kinds of nodes supported by the graph.
const (
	NodeKindCode            NodeKind = "Code"            // Represents a source code file or entity.
	NodeKindRequirement     NodeKind = "Requirement"     // Represents a business requirement.
	NodeKindFeature         NodeKind = "Feature"         // Represents a functional feature.
	NodeKindTest            NodeKind = "Test"            // Represents a test case.
	NodeKindGherkinFeature  NodeKind = "GherkinFeature"  // Represents a Gherkin .feature file.
	NodeKindGherkinScenario NodeKind = "GherkinScenario" // Represents a single Scenario in a Gherkin file.
	NodeKindStepDefinition  NodeKind = "StepDefinition"  // Represents a code function implementing a Gherkin step.
)

// EdgeType represents the relationship type between two nodes.
type EdgeType string

// Constants defining the standard relationship types in the graph.
const (
	EdgeTypeDefines       EdgeType = "DEFINES"        // Requirement -> Feature
	EdgeTypeImplementedBy EdgeType = "IMPLEMENTED_BY" // Feature -> Code, Requirement -> Code
	EdgeTypeVerifies      EdgeType = "VERIFIES"       // Test/Scenario -> Requirement
	EdgeTypeExecutes      EdgeType = "EXECUTES"       // GherkinScenario -> StepDefinition
	EdgeTypeCalls         EdgeType = "CALLS"          // StepDefinition -> Code
	EdgeTypeDescribedBy   EdgeType = "DESCRIBED_BY"   // Requirement -> GherkinFeature
	EdgeTypeImports       EdgeType = "IMPORTS"        // Code -> Code (for architectural analysis)
)

// Node represents a single entity in the semantic graph.
// It can represent code, requirements, features, or tests.
type Node struct {
	ID         string                 `json:"id"`
	Kind       NodeKind               `json:"kind"`
	Properties map[string]interface{} `json:"properties,omitempty"` // Flexible storage for node-specific data.
	Metadata   map[string]interface{} `json:"metadata,omitempty"`   // Analysis metadata like layer, language, etc.
}

// Edge represents a directed relationship between two nodes in the graph.
type Edge struct {
	SourceID string   `json:"source_id"`
	TargetID string   `json:"target_id"`
	Type     EdgeType `json:"type"`
}

// ViolationSeverity indicates the seriousness of a detected violation.
type ViolationSeverity string

// Constants for violation severity levels.
const (
	SeverityCritical ViolationSeverity = "CRITICAL"
	SeverityWarning  ViolationSeverity = "WARNING"
)

// ViolationKind indicates the category of the violation.
type ViolationKind string

// Constants for violation kinds.
const (
	ViolationKindArchLayer ViolationKind = "ARCH_LAYER_VIOLATION" // Violation of architectural layering rules.
	ViolationKindBDDDrift  ViolationKind = "BDD_DRIFT"            // Mismatch between Gherkin specs and implementation.
)

// Violation represents a detected issue in the codebase, such as an architectural breach or missing test coverage.
type Violation struct {
	Severity ViolationSeverity `json:"severity"`       // The severity of the violation.
	Message  string            `json:"message"`        // Human-readable description of the violation.
	File     string            `json:"file"`           // The file associated with the violation.
	Kind     ViolationKind     `json:"kind"`           // The category of the violation.
	Line     int               `json:"line,omitempty"` // The line number where the violation occurred (optional).
}

// Helper structs for specific node properties (optional, for type safety if needed)
// For now, we rely on the generic Properties map for flexibility,
// but we can define structs to marshal/unmarshal specific kinds.

// RequirementProps defines the standard properties for a Requirement node.
type RequirementProps struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Status             string   `json:"status"`
	Priority           string   `json:"priority"`
	ExternalLink       string   `json:"externalLink"`
	AcceptanceCriteria []string `json:"acceptanceCriteria"`
}
