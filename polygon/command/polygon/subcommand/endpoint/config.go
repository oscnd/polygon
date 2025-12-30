package endpoint

import (
	"fmt"
	"os"
	"path/filepath"

	"go.scnd.dev/open/polygon/command/polygon/index"
	"gopkg.in/yaml.v3"
)

// Config represents the endpoint.yml configuration
type Config struct {
	EndpointDir  string    `yaml:"endpoint_dir"`
	EndpointFile string    `yaml:"endpoint_file"`
	App          index.App // Application reference for parser use
}

// LoadConfig loads and parses the endpoint.yml configuration file
func LoadConfig(app index.App) (*Config, error) {
	configPath := filepath.Join(*app.Directory(), "endpoint.yml")

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load endpoint.yml: %w", err)
	}

	cfg := &Config{App: app}
	if err := yaml.Unmarshal(configData, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse endpoint.yml: %w", err)
	}

	// Validate configuration
	if cfg.EndpointDir == "" {
		return nil, fmt.Errorf("endpoint_dir is required in endpoint.yml")
	}
	if cfg.EndpointFile == "" {
		return nil, fmt.Errorf("endpoint_file is required in endpoint.yml")
	}

	return cfg, nil
}
