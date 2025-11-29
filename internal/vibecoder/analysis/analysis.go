package analysis

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	curex "github.com/cucumber/cucumber-expressions-go"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/domain"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/graph"
	"github.com/modelcontextprotocol/go-sdk/examples/server/vibecoder/internal/vibecoder/parser"
)

type Analyzer struct {
	Graph *graph.Graph
	// Cache TSConfig for resolution
	tsConfigs map[string]TSConfig
	goMods    map[string]GoMod
}

type TSConfig struct {
	BaseUrl string              `json:"baseUrl"`
	Paths   map[string][]string `json:"paths"`
}

type GoMod struct {
	Module string
}

func NewAnalyzer(g *graph.Graph) *Analyzer {
	return &Analyzer{
		Graph:     g,
		tsConfigs: make(map[string]TSConfig),
		goMods:    make(map[string]GoMod),
	}
}

func (a *Analyzer) AnalyzeFile(path string, content []byte) error {
	// Pre-scan for config files
	if filepath.Base(path) == "tsconfig.json" {
		a.parseTSConfig(path, content)
		return nil
	}
	if filepath.Base(path) == "go.mod" {
		a.parseGoMod(path, content)
		return nil
	}

	// 1. Determine Layer/Type
	layer := detectLayer(path)

	// 2. Create/Update Node
	nodeID := path
	var node *domain.Node

	// Handle Gherkin
	if strings.HasSuffix(path, ".feature") {
		return a.analyzeGherkin(path, content)
	}

	// Handle Code
	lang := parser.DetectLanguage(path)
	if lang == parser.LangUnknown {
		if layer != "" {
			node = &domain.Node{
				ID:   nodeID,
				Kind: domain.NodeKindCode,
				Metadata: map[string]interface{}{
					"layer":    layer,
					"language": "unknown",
				},
			}
			a.Graph.AddNode(node)
		}
		return nil
	}

	node = &domain.Node{
		ID:   nodeID,
		Kind: domain.NodeKindCode,
		Metadata: map[string]interface{}{
			"layer":    layer,
			"language": string(lang),
		},
	}
	a.Graph.AddNode(node)

	// 3. Parse Imports
	imports, err := parser.ParseImports(content, lang)
	if err == nil {
		for _, imp := range imports {
			targetID := a.resolveImport(path, imp, lang)
			a.Graph.AddEdge(nodeID, targetID, domain.EdgeTypeImports)
		}
	}

	// 4. Parse Step Definitions (if Test layer)
	if layer == "interface" || strings.Contains(path, "test") || strings.Contains(path, "steps") {
		steps, err := parser.ParseStepDefinitions(content, lang)
		if err == nil && len(steps) > 0 {
			for _, s := range steps {
				// Use hash or cleaner ID to avoid filesystem weirdness in ID
				stepID := fmt.Sprintf("stepdef:%s:%s", s.FunctionName, s.Pattern)
				stepNode := &domain.Node{
					ID:   stepID,
					Kind: domain.NodeKindStepDefinition,
					Properties: map[string]interface{}{
						"regex_pattern": s.Pattern,
						"function_name": s.FunctionName,
						"filepath":      path,
						"line":          s.Line,
					},
				}
				a.Graph.AddNode(stepNode)
				a.Graph.AddEdge(stepID, nodeID, domain.EdgeTypeCalls)
			}
		}
	}

	return nil
}

func (a *Analyzer) analyzeGherkin(path string, content []byte) error {
	feat, err := parser.ParseGherkin(content)
	if err != nil {
		return err
	}

	featID := "gh:feat:" + strings.ReplaceAll(feat.Name, " ", "_")
	featNode := &domain.Node{
		ID:   featID,
		Kind: domain.NodeKindGherkinFeature,
		Properties: map[string]interface{}{
			"name": feat.Name,
			"file": path,
		},
	}
	a.Graph.AddNode(featNode)

	for _, sc := range feat.Scenarios {
		scID := "gh:scen:" + strings.ReplaceAll(sc.Name, " ", "_")
		scNode := &domain.Node{
			ID:   scID,
			Kind: domain.NodeKindGherkinScenario,
			Properties: map[string]interface{}{
				"name":       sc.Name,
				"file":       path,
				"steps_hash": sc.StepsHash,
				"line":       sc.Line,
				"steps":      sc.Steps,
			},
		}
		a.Graph.AddNode(scNode)
	}
	return nil
}

func detectLayer(path string) string {
	if strings.Contains(path, "/domain/") {
		return "domain"
	}
	if strings.Contains(path, "/application/") {
		return "application"
	}
	if strings.Contains(path, "/infrastructure/") {
		return "infrastructure"
	}
	if strings.Contains(path, "/interface/") || strings.Contains(path, "/api/") {
		return "interface"
	}
	return ""
}

// Config Parsing Helpers

func (a *Analyzer) parseTSConfig(path string, content []byte) {
	// Simplified parsing for compilerOptions.paths and baseUrl
	var raw struct {
		CompilerOptions struct {
			BaseUrl string              `json:"baseUrl"`
			Paths   map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(content, &raw); err == nil {
		dir := filepath.Dir(path)
		a.tsConfigs[dir] = TSConfig{
			BaseUrl: raw.CompilerOptions.BaseUrl,
			Paths:   raw.CompilerOptions.Paths,
		}
	}
}

func (a *Analyzer) parseGoMod(path string, content []byte) {
	// Simple regex to find module name
	re := regexp.MustCompile(`module\s+([^\s]+)`)
	matches := re.FindSubmatch(content)
	if len(matches) > 1 {
		dir := filepath.Dir(path)
		a.goMods[dir] = GoMod{Module: string(matches[1])}
	}
}

// Import Resolution

func (a *Analyzer) resolveImport(sourcePath, importStr string, lang parser.Language) string {
	importStr = strings.Trim(importStr, "\"'`")

	switch lang {
	case parser.LangTypeScript:
		return a.resolveTSImport(sourcePath, importStr)
	case parser.LangGo:
		return a.resolveGoImport(sourcePath, importStr)
	case parser.LangPython:
		// Relative imports
		if strings.HasPrefix(importStr, ".") {
			return filepath.Join(filepath.Dir(sourcePath), importStr)
		}
		// Absolute/Package? Return as is for now.
		return importStr
	case parser.LangRust:
		// crate:: or super::
		if strings.HasPrefix(importStr, "crate::") {
			// Try to find Cargo.toml logic? simplified:
			return strings.Replace(importStr, "crate::", "", 1)
		}
		return importStr
	default:
		// Basic relative fallback
		if strings.HasPrefix(importStr, ".") {
			return filepath.Join(filepath.Dir(sourcePath), importStr)
		}
		return importStr
	}
}

func (a *Analyzer) resolveTSImport(sourcePath, importStr string) string {
	// 1. Relative
	if strings.HasPrefix(importStr, ".") {
		return filepath.Join(filepath.Dir(sourcePath), importStr)
	}

	// 2. TSConfig Paths
	// Find nearest tsconfig
	dir := filepath.Dir(sourcePath)
	var config TSConfig
	var found bool

	// Walk up to find tsconfig
	for {
		if c, ok := a.tsConfigs[dir]; ok {
			config = c
			found = true
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if found {
		// Check paths
		for pattern, targets := range config.Paths {
			// Simple exact match or wildcard
			// "domain/*": ["src/domain/*"]
			patternPrefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(importStr, patternPrefix) {
				suffix := strings.TrimPrefix(importStr, patternPrefix)
				if len(targets) > 0 {
					target := targets[0] // take first
					targetPrefix := strings.TrimSuffix(target, "*")
					// Resolve relative to baseUrl (which is relative to tsconfig dir)
					// Assumes baseUrl is "." or "src"
					// This is complex. Simplified:
					// If baseUrl is set, paths are relative to it.
					// If not, relative to tsconfig.
					base := config.BaseUrl
					if base == "" {
						base = "."
					}
					resolved := filepath.Join(dir, base, targetPrefix+suffix)
					return resolved
				}
			}
		}
	}

	return importStr
}

func (a *Analyzer) resolveGoImport(sourcePath, importStr string) string {
	// Find nearest go.mod
	dir := filepath.Dir(sourcePath)
	var mod GoMod
	var found bool

	for {
		if m, ok := a.goMods[dir]; ok {
			mod = m
			found = true
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if found {
		if strings.HasPrefix(importStr, mod.Module) {
			rel := strings.TrimPrefix(importStr, mod.Module)
			return filepath.Join(dir, rel)
		}
	}
	return importStr
}

func (a *Analyzer) FindViolations() []domain.Violation {
	var violations []domain.Violation

	nodes := a.Graph.GetAllNodes()
	for _, node := range nodes {
		if node.Kind == domain.NodeKindCode {
			layer := node.Metadata["layer"]
			if layer == nil {
				continue
			}
			lStr := layer.(string)

			// Get imports
			edges := a.Graph.GetEdgesFrom(node.ID)
			for _, edge := range edges {
				if edge.Type == domain.EdgeTypeImports {
					target, ok := a.Graph.GetNode(edge.TargetID)
					// If we can't find the target node, we might try fuzzy matching or skip
					// For now skip if not found (external lib)
					if !ok {
						// Heuristic: check if targetID looks like infra/app
						if strings.Contains(edge.TargetID, "infrastructure") {
							// Check rules
							if lStr == "domain" {
								violations = append(violations, domain.Violation{
									Severity: domain.SeverityCritical,
									Message:  fmt.Sprintf("Domain Rule Broken: '%s' imports '%s' (Infrastructure).", node.ID, edge.TargetID),
									File:     node.ID,
									Kind:     domain.ViolationKindArchLayer,
								})
							}
						}
						continue
					}

					targetLayer := target.Metadata["layer"]
					if targetLayer == nil {
						continue
					}
					tlStr := targetLayer.(string)

					// Rule: Domain cannot import Infra or App
					if lStr == "domain" {
						if tlStr == "infrastructure" || tlStr == "application" {
							violations = append(violations, domain.Violation{
								Severity: domain.SeverityCritical,
								Message:  fmt.Sprintf("Domain Rule Broken: '%s' imports '%s' (%s).", node.ID, target.ID, tlStr),
								File:     node.ID,
								Kind:     domain.ViolationKindArchLayer,
							})
						}
					}
					// Rule: App cannot import Infra (strict) or should use ports.
					if lStr == "application" && tlStr == "infrastructure" {
						violations = append(violations, domain.Violation{
							Severity: domain.SeverityWarning,
							Message:  fmt.Sprintf("Application Alert: '%s' imports '%s' (Infrastructure). Should use Ports.", node.ID, target.ID),
							File:     node.ID,
							Kind:     domain.ViolationKindArchLayer,
						})
					}
				}
			}
		}
	}

	// BDD Drift Check
	scenarios := a.filterNodes(domain.NodeKindGherkinScenario)
	stepDefs := a.filterNodes(domain.NodeKindStepDefinition)

	// Build parameter type registry
	paramRegistry := curex.NewParameterTypeRegistry()

	for _, sc := range scenarios {
		scSteps, ok := sc.Properties["steps"].([]string)
		if !ok {
			continue
		}

		for _, stepText := range scSteps {
			cleanedStep := cleanStepText(stepText)
			matched := false
			for _, sd := range stepDefs {
				pattern, ok := sd.Properties["regex_pattern"].(string)
				if !ok {
					continue
				}
				if matchStep(cleanedStep, pattern, paramRegistry) {
					matched = true
					break
				}
			}

			if !matched {
				violations = append(violations, domain.Violation{
					Severity: domain.SeverityWarning,
					Message:  fmt.Sprintf("BDD Drift/Missing: Step '%s' in '%s' has no matching StepDefinition.", stepText, sc.ID),
					File:     sc.Properties["file"].(string),
					Kind:     domain.ViolationKindBDDDrift,
					Line:     sc.Properties["line"].(int),
				})
			}
		}
	}

	return violations
}

// IndexStepDefinitions tries to link Scenarios to Steps
func (a *Analyzer) IndexStepDefinitions() {
	scenarios := a.filterNodes(domain.NodeKindGherkinScenario)
	stepDefs := a.filterNodes(domain.NodeKindStepDefinition)

	paramRegistry := curex.NewParameterTypeRegistry()

	for _, sc := range scenarios {
		scSteps, ok := sc.Properties["steps"].([]string)
		if !ok {
			continue
		}

		for _, stepText := range scSteps {
			cleanedStep := cleanStepText(stepText)

			for _, sd := range stepDefs {
				pattern, ok := sd.Properties["regex_pattern"].(string)
				if !ok {
					continue
				}

				if matchStep(cleanedStep, pattern, paramRegistry) {
					a.Graph.AddEdge(sc.ID, sd.ID, domain.EdgeTypeExecutes)
				}
			}
		}
	}
}

func cleanStepText(step string) string {
	parts := strings.Fields(step)
	if len(parts) > 1 {
		return strings.Join(parts[1:], " ")
	}
	return step
}

func matchStep(text, pattern string, registry *curex.ParameterTypeRegistry) bool {
	// Try Cucumber Expression first if it looks like one (has {})
	if strings.Contains(pattern, "{") && strings.Contains(pattern, "}") {
		expression, err := curex.NewCucumberExpression(pattern, registry)
		if err == nil {
			args, err := expression.Match(text)
			return err == nil && args != nil
		}
	}

	// Fallback to Regex
	re, err := regexp.Compile(pattern)
	if err == nil {
		return re.MatchString(text)
	}

	// Fallback to simple substring
	return strings.Contains(text, pattern)
}

func (a *Analyzer) filterNodes(kind domain.NodeKind) []*domain.Node {
	var res []*domain.Node
	for _, n := range a.Graph.GetAllNodes() {
		if n.Kind == kind {
			res = append(res, n)
		}
	}
	return res
}
