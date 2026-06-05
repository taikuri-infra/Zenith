package installstate

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// State holds persisted install results.
type State struct {
	Domain            string `yaml:"domain"`
	ServerIP          string `yaml:"server_ip"`
	MissionControlURL string `yaml:"mission_control_url"`
	CloudURL          string `yaml:"cloud_url"`
	AdminUser         string `yaml:"admin_user"`
	AdminPassword     string `yaml:"admin_password"`
	SSHKeyPath        string `yaml:"ssh_key_path,omitempty"`
	Provider          string `yaml:"provider"`
	Region            string `yaml:"region,omitempty"`
	ServerID          int64  `yaml:"server_id,omitempty"`
	SSHKeyID          int64  `yaml:"ssh_key_id,omitempty"`
	InstalledAt       string `yaml:"installed_at"`
}

// defaultStatePath returns ~/.zen/install-state.yaml.
func defaultStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".zen", "install-state.yaml"), nil
}

// Save writes the state to the default path (~/.zen/install-state.yaml).
func Save(s *State) error {
	return SaveTo(s, "")
}

// SaveTo writes the state to path (empty = default).
func SaveTo(s *State, path string) error {
	if path == "" {
		var err error
		path, err = defaultStatePath()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// Load reads the state from the default path.
func Load() (*State, error) {
	return LoadFrom("")
}

// LoadFrom reads state from path (empty = default).
func LoadFrom(path string) (*State, error) {
	if path == "" {
		var err error
		path, err = defaultStatePath()
		if err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no install state found at %s — run 'zen install' first", path)
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &s, nil
}

// Exists reports whether the default install state file exists.
func Exists() bool {
	path, err := defaultStatePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
