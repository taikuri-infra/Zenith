package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the CLI configuration persisted to disk.
type Config struct {
	APIBaseURL   string `json:"api_base_url"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ProjectID    string `json:"project_id,omitempty"`
}

// DefaultBaseURL is the default Zenith API endpoint.
const DefaultBaseURL = "https://api.freezenith.com"

// configDir returns ~/.zenith
func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".zenith")
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	return filepath.Join(configDir(), "config.json")
}

// Load reads the config from disk. Returns default config if file doesn't exist.
func Load() *Config {
	cfg := &Config{APIBaseURL: DefaultBaseURL}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg
	}

	_ = json.Unmarshal(data, cfg)
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = DefaultBaseURL
	}
	return cfg
}

// Save writes the config to disk.
func (c *Config) Save() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0600)
}

// IsLoggedIn returns true if an access token is present.
func (c *Config) IsLoggedIn() bool {
	return c.AccessToken != ""
}
