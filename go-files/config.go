package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Enabled       bool              `yaml:"enabled"`
	LogPath       string            `yaml:"log_path"`
	BlockMethod   string            `yaml:"block_method"`
	IPSet         IPSetConfig       `yaml:"ipset"`
	Whitelist     []string          `yaml:"whitelist"`
	Fields        map[string]string `yaml:"fields"`
	Required      []string          `yaml:"required"`
	ValidateIP    []string          `yaml:"validate_ip"`
	PrintOriginal bool              `yaml:"print_original"`
}

// IPSetConfig contains settings used by the Linux ipset backend.
type IPSetConfig struct {
	SetName string `yaml:"set_name"`
}

// loadConfig reads config.yaml, decodes it, and validates settings.
func loadConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	var cfg Config
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	decoder.KnownFields(true)

	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	if strings.TrimSpace(cfg.LogPath) == "" {
		return nil, errors.New("log_path cannot be empty")
	}

	if len(cfg.Fields) == 0 {
		return nil, errors.New("fields cannot be empty")
	}

	// Validate the ipset configuration.
	if strings.TrimSpace(cfg.IPSet.SetName) == "" {
		return nil, errors.New("ipset.set_name is required")
	}

	// Reject the configuration if any whitelist entry is malformed.
	if err := validateWhitelist(cfg.Whitelist); err != nil {
		return nil, fmt.Errorf(
			"validate whitelist: %w",
			err,
		)
	}
//go through all the values in fields, required, and validate_IP
	for name, path := range cfg.Fields {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("field names cannot be empty")
		}
		if strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("JSON path for field %q cannot be empty", name)
		}
	}

	for _, name := range cfg.Required {
		if _, exists := cfg.Fields[name]; !exists {
			return nil, fmt.Errorf("required field %q is not defined under fields", name)
		}
	}

	for _, name := range cfg.ValidateIP {
		if _, exists := cfg.Fields[name]; !exists {
			return nil, fmt.Errorf("IP validation field %q is not defined under fields", name)
		}
	}

	return &cfg, nil
}