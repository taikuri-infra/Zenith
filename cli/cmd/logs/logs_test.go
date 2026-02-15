package logs

import (
	"strings"
	"testing"
)

func TestFormatLogLine(t *testing.T) {
	tests := []struct {
		level    string
		contains string
	}{
		{"INF", "INF"},
		{"WRN", "WRN"},
		{"ERR", "ERR"},
		{"DBG", "DBG"},
		{"UNKNOWN", "UNKNOWN"},
	}

	for _, tt := range tests {
		line := FormatLogLine("2025-01-01T00:00:00Z", tt.level, "test message", "pod-1")
		if !strings.Contains(line, tt.contains) {
			t.Errorf("FormatLogLine with level %s should contain '%s', got: %s", tt.level, tt.contains, line)
		}
		if !strings.Contains(line, "test message") {
			t.Errorf("FormatLogLine should contain message, got: %s", line)
		}
		if !strings.Contains(line, "pod-1") {
			t.Errorf("FormatLogLine should contain pod name, got: %s", line)
		}
	}
}
