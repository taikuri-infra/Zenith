package db

import "testing"

func TestShellCommand(t *testing.T) {
	tests := []struct {
		engine  string
		cmd     string
		hasArgs bool
	}{
		{"postgresql", "psql", true},
		{"postgres", "psql", true},
		{"mysql", "mysql", true},
		{"redis", "redis-cli", true},
		{"mongodb", "mongosh", true},
		{"mongo", "mongosh", true},
		{"unknown", "", false},
	}

	for _, tt := range tests {
		cmd, args := ShellCommand(tt.engine)
		if cmd != tt.cmd {
			t.Errorf("ShellCommand(%q) cmd = %q, want %q", tt.engine, cmd, tt.cmd)
		}
		if tt.hasArgs && len(args) == 0 {
			t.Errorf("ShellCommand(%q) should have args", tt.engine)
		}
		if !tt.hasArgs && args != nil {
			t.Errorf("ShellCommand(%q) should have nil args", tt.engine)
		}
	}
}

func TestDefaultPort(t *testing.T) {
	tests := []struct {
		engine string
		port   int
	}{
		{"postgresql", 5432},
		{"postgres", 5432},
		{"mysql", 3306},
		{"redis", 6379},
		{"mongodb", 27017},
		{"mongo", 27017},
		{"unknown", 0},
	}

	for _, tt := range tests {
		port := DefaultPort(tt.engine)
		if port != tt.port {
			t.Errorf("DefaultPort(%q) = %d, want %d", tt.engine, port, tt.port)
		}
	}
}
