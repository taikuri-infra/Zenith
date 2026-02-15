package tui

import (
	"testing"
)

func TestStatusBadge(t *testing.T) {
	tests := []struct {
		phase string
	}{
		{"Running"},
		{"Ready"},
		{"Active"},
		{"Pending"},
		{"Provisioning"},
		{"Failed"},
		{"Stopped"},
		{"Unknown"},
	}

	for _, tt := range tests {
		result := StatusBadge(tt.phase)
		if result == "" {
			t.Errorf("StatusBadge(%q) returned empty string", tt.phase)
		}
	}
}
