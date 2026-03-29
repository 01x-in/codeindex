package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigFileName is the default config file name.
const ConfigFileName = ".codeindex.yaml"

// Config represents the .codeindex.yaml configuration.
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
		IndexPath: ".codeindex",
	}
}

// Load reads and parses a .codeindex.yaml file.
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

// LoadOrDetect resolves the config using the cascade:
// 1. Explicit .codeindex.yaml in dir (wins if present)
// 2. Auto-detection from project markers fills language gaps
// 3. Built-in defaults for everything else
//
// Returns the resolved config and a boolean indicating whether an explicit
// config file was found. If no config file exists, the returned config
// uses auto-detected languages merged with defaults — this is NOT an error.
func LoadOrDetect(dir string) (Config, bool, error) {
	configPath := filepath.Join(dir, ConfigFileName)

	// Try explicit config file first.
	if _, err := os.Stat(configPath); err == nil {
		cfg, loadErr := Load(configPath)
		if loadErr != nil {
			return Config{}, true, loadErr
		}
		// Fill in defaults for missing fields.
		cfg = mergeDefaults(cfg)
		return cfg, true, nil
	}

	// No config file — auto-detect + defaults.
	cfg := DefaultConfig()

	detected, err := DetectLanguages(dir)
	if err != nil {
		return Config{}, false, fmt.Errorf("detecting languages: %w", err)
	}

	for _, d := range detected {
		cfg.Languages = appendUnique(cfg.Languages, d.Language)
	}

	return cfg, false, nil
}

// mergeDefaults fills in zero-value fields from DefaultConfig.
func mergeDefaults(cfg Config) Config {
	defaults := DefaultConfig()

	if len(cfg.Ignore) == 0 {
		cfg.Ignore = defaults.Ignore
	}
	if len(cfg.QueryPrimitives) == 0 {
		cfg.QueryPrimitives = defaults.QueryPrimitives
	}
	if cfg.IndexPath == "" {
		cfg.IndexPath = defaults.IndexPath
	}

	return cfg
}

// appendUnique appends s to slice only if not already present.
func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
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

// IsNotFound returns true if the error is due to a missing config file.
func IsNotFound(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
