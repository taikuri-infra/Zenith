package version

import (
	"testing"
)

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()

	if info["version"] == "" {
		t.Error("Expected non-empty version")
	}
	if info["go"] == "" {
		t.Error("Expected non-empty go version")
	}
	if info["os"] == "" {
		t.Error("Expected non-empty OS")
	}
	if info["arch"] == "" {
		t.Error("Expected non-empty arch")
	}
}

func TestVersionCommand(t *testing.T) {
	if Cmd.Use != "version" {
		t.Errorf("Expected command name 'version', got '%s'", Cmd.Use)
	}
}
