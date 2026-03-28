package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the .code-index.yaml configuration.
type Config struct {
	Version         int      `yaml:"version" json:"version"`
	Languages       []string `yaml:"languages" json:"languages"`
	Ignore          []string `yaml:"ignore" json:"ignore"`
	QueryPrimitives []string `yaml:"query_primitives" json:"query_primitives"`
	IndexPath       string   `yaml:"index_path" json:"index_path"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Version:   1,
		Languages: []string{},
		Ignore:    []string{"node_modules", "vendor", ".git", "dist", "build"},
		QueryPrimitives: []string{
			"get_file_structure",
			"find_symbol",
			"get_references",
			"get_callers",
			"get_subgraph",
			"reindex",
		},
		IndexPath: ".code-index",
	}
}

// Load reads and parses a .code-index.yaml file.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Save writes the config to a YAML file.
func (c Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// Validate checks the config for errors.
func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version: %d (supported: 1)", c.Version)
	}

	supportedLangs := map[string]bool{
		"typescript": true,
		"go":         true,
		"python":     true,
		"rust":       true,
	}

	for _, lang := range c.Languages {
		if !supportedLangs[lang] {
			return fmt.Errorf("unknown language %q — supported: typescript, go, python, rust", lang)
		}
	}

	return nil
}
