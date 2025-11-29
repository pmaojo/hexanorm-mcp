package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	PersistenceDir string   `json:"persistence_dir"`
	IncludedLayers []string `json:"included_layers"`
	ExcludedDirs   []string `json:"excluded_dirs"`
}

var DefaultConfig = Config{
	PersistenceDir: ".vibecoder",
	IncludedLayers: []string{"domain", "application", "infrastructure", "interface", "test"},
	ExcludedDirs:   []string{"node_modules", ".git", "vendor", "dist", "build"},
}

func LoadConfig(rootDir string) (*Config, error) {
	configPath := filepath.Join(rootDir, "vibecoder.json")
	content, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		// Return defaults if file not found
		return &DefaultConfig, nil
	}
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	// Merge with defaults/Validation if needed
	if config.PersistenceDir == "" {
		config.PersistenceDir = DefaultConfig.PersistenceDir
	}

	return &config, nil
}
