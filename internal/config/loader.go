package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}
