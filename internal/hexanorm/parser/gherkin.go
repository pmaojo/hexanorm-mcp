package parser

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// GherkinFeature represents a parsed .feature file containing one or more scenarios.
type GherkinFeature struct {
	Name      string            // The name of the feature.
	Scenarios []GherkinScenario // List of scenarios defined in the feature.
}

// GherkinScenario represents a single scenario within a feature file.
type GherkinScenario struct {
	Name      string   // The name of the scenario.
	Steps     []string // The raw text of the steps (Given/When/Then).
	StepsHash string   // A hash of the steps used to detect changes or duplicates.
	Line      int      // The line number where the scenario starts.
}

// ParseGherkin parses the content of a .feature file and returns a GherkinFeature struct.
// It handles Feature and Scenario definitions and collects steps.
func ParseGherkin(content []byte) (*GherkinFeature, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	feature := &GherkinFeature{}
	var currentScenario *GherkinScenario

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "@") {
			continue
		}

		if strings.HasPrefix(line, "Feature:") {
			feature.Name = strings.TrimSpace(strings.TrimPrefix(line, "Feature:"))
		} else if strings.HasPrefix(line, "Scenario:") {
			if currentScenario != nil {
				finalizeScenario(currentScenario)
				feature.Scenarios = append(feature.Scenarios, *currentScenario)
			}
			currentScenario = &GherkinScenario{
				Name: strings.TrimSpace(strings.TrimPrefix(line, "Scenario:")),
				Line: lineNum,
			}
		} else if isStep(line) {
			if currentScenario != nil {
				currentScenario.Steps = append(currentScenario.Steps, line)
			}
		}
	}
	if currentScenario != nil {
		finalizeScenario(currentScenario)
		feature.Scenarios = append(feature.Scenarios, *currentScenario)
	}

	return feature, nil
}

func isStep(line string) bool {
	words := strings.Fields(line)
	if len(words) == 0 {
		return false
	}
	kw := words[0]
	switch kw {
	case "Given", "When", "Then", "And", "But":
		return true
	}
	return false
}

func finalizeScenario(sc *GherkinScenario) {
	// Calculate hash of steps
	h := sha256.New()
	for _, s := range sc.Steps {
		h.Write([]byte(s))
	}
	sc.StepsHash = hex.EncodeToString(h.Sum(nil))[:8]
}
