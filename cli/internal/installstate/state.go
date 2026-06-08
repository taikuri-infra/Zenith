package installstate

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultPath returns the default path for the install state file.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zen", "install-state.yaml"), nil
}

// State holds all persisted installation state.
type State struct {
	Domain            string    `yaml:"domain"`
	ServerIP          string    `yaml:"server_ip"`
	MissionControlURL string    `yaml:"mission_control_url"`
	CloudURL          string    `yaml:"cloud_url"`
	AdminUser         string    `yaml:"admin_user"`
	ZenithVersion     string    `yaml:"zenith_version,omitempty"`
	SSHKeyPath        string    `yaml:"ssh_key_path"`
	// ServerHostKey is the base64-encoded SSH host public key captured on first connection.
	// Used to prevent MITM attacks on subsequent connections.
	ServerHostKey     string    `yaml:"server_host_key,omitempty"`
	Provider          string    `yaml:"provider"`
	Region            string    `yaml:"region"`
	ServerID          string    `yaml:"server_id"`
	SSHKeyID          string    `yaml:"ssh_key_id"`
	InstalledAt       time.Time `yaml:"installed_at"`

	// CompletedSteps tracks which installation steps have been completed.
	// Used by --resume to skip already-completed steps.
	CompletedSteps []string `yaml:"completed_steps,omitempty"`
}

// Save persists the state to the default path (~/.zen/install-state.yaml).
func Save(s *State) error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return SaveTo(s, path)
}

// SaveTo persists the state to an explicit path.
func SaveTo(s *State, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Load reads state from the default path.
func Load() (*State, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads state from an explicit path.
func LoadFrom(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Exists reports whether the default state file exists.
func Exists() bool {
	path, err := DefaultPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// MarkStepComplete adds a step name to CompletedSteps and saves state.
func MarkStepComplete(s *State, stepName string) error {
	// Avoid duplicates
	for _, name := range s.CompletedSteps {
		if name == stepName {
			return nil
		}
	}
	s.CompletedSteps = append(s.CompletedSteps, stepName)
	return Save(s)
}

// IsStepComplete reports whether stepName is in CompletedSteps.
func IsStepComplete(s *State, stepName string) bool {
	for _, name := range s.CompletedSteps {
		if name == stepName {
			return true
		}
	}
	return false
}
