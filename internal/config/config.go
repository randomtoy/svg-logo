package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type SvgLogo struct {
	OutputDir string `yaml:"output_dir"` // Directory to save downloaded SVG files
}

type Item struct {
	Path    string `yaml:"path"`              // Relative path to the SVG file
	URL     string `yaml:"url"`               // Source URL
	License string `yaml:"license,omitempty"` // Optional license information
	Notes   string `yaml:"notes,omitempty"`   // Optional notes
}

type Config struct {
	Items   []Item  `yaml:"items"`
	SvgLogo SvgLogo `yaml:"svg"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}
