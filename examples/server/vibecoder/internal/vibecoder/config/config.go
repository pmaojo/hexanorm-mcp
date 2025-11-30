package config

import (
	"encoding/json"
	"os"
)

type LayerConfig struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"` // Simple substring match for now
}

type Config struct {
	Layers   []LayerConfig `json:"layers"`
	Excludes []string      `json:"excludes"`
}

func DefaultConfig() *Config {
	return &Config{
		Layers: []LayerConfig{
			{Name: "domain", Pattern: "/domain/"},
			{Name: "application", Pattern: "/application/"},
			{Name: "infrastructure", Pattern: "/infrastructure/"},
			{Name: "interface", Pattern: "/interface/"},
			{Name: "interface", Pattern: "/api/"},
		},
		Excludes: []string{"node_modules", ".git", "dist", "build", ".vibecoder"},
	}
}

func LoadConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
