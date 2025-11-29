package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents the configuration for the Hexanorm server.
// It controls directory exclusion, layer inclusion, and persistence settings.
type Config struct {
	ExcludedDirs   []string `json:"excluded_dirs"`   // List of directory names to exclude from analysis.
	IncludedLayers []string `json:"included_layers"` // List of architectural layers to analyze.
	PersistenceDir string   `json:"persistence_dir"` // Directory path to store the SQLite database.
}

// DefaultConfig provides a standard configuration used when no config file is found.
var DefaultConfig = Config{
	ExcludedDirs:   []string{"node_modules", "dist", "build", ".git", "vendor"},
	IncludedLayers: []string{"domain", "application", "infrastructure", "interface"},
	PersistenceDir: ".hexanorm",
}

// LoadConfig reads and parses the `hexanorm.json` configuration file from the specified root directory.
// If the file does not exist or cannot be parsed, it returns an error.
// It expects the config file to be named `hexanorm.json`.
// If the configuration file is found but some fields are missing, it applies default values.
func LoadConfig(rootDir string) (*Config, error) {
	path := filepath.Join(rootDir, "hexanorm.json")
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}

	// Apply defaults if empty
	if len(cfg.ExcludedDirs) == 0 {
		cfg.ExcludedDirs = DefaultConfig.ExcludedDirs
	}
	if len(cfg.IncludedLayers) == 0 {
		cfg.IncludedLayers = DefaultConfig.IncludedLayers
	}
	if cfg.PersistenceDir == "" {
		cfg.PersistenceDir = DefaultConfig.PersistenceDir
	}

	return &cfg, nil
}
