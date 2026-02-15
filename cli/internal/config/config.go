package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// APIEndpoint is the Zenith API server URL
	APIEndpoint string `yaml:"api_endpoint"`

	// Token is the authentication token
	Token string `yaml:"token,omitempty"`

	// Project is the current active project
	Project string `yaml:"project,omitempty"`

	// Region is the default Hetzner region
	Region string `yaml:"region,omitempty"`

	// OutputFormat is the default output format (table, json, yaml)
	OutputFormat string `yaml:"output_format,omitempty"`
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".zen", "config.yaml")
}

func Load() (*Config, error) {
	return LoadFrom(DefaultConfigPath())
}

func LoadFrom(path string) (*Config, error) {
	cfg := &Config{
		APIEndpoint:  "http://localhost:8080",
		Region:       "fsn1",
		OutputFormat: "table",
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Save() error {
	return c.SaveTo(DefaultConfigPath())
}

func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
