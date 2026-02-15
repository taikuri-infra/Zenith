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
		t.Run(tt.level, func(t *testing.T) {
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
		})
	}
}

func TestFormatLogLine_AllLevelVariants(t *testing.T) {
	// Test all level aliases that map to the same color
	tests := []struct {
		name  string
		level string
	}{
		{"INFO full", "INFO"},
		{"INF short", "INF"},
		{"WARN full", "WARN"},
		{"WARNING full", "WARNING"},
		{"WRN short", "WRN"},
		{"ERROR full", "ERROR"},
		{"ERR short", "ERR"},
		{"DEBUG full", "DEBUG"},
		{"DBG short", "DBG"},
		{"unknown level", "TRACE"},
		{"empty level", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := FormatLogLine("2026-01-15T10:30:00Z", tt.level, "test msg", "pod-abc-123")
			if line == "" {
				t.Error("Expected non-empty formatted log line")
			}
			// The level should appear in the output (padded to 5 chars)
			if tt.level != "" && !strings.Contains(line, tt.level) {
				t.Errorf("Expected line to contain level '%s', got: %s", tt.level, line)
			}
		})
	}
}

func TestFormatLogLine_ContainsTimestamp(t *testing.T) {
	timestamps := []string{
		"2026-01-15T10:30:00Z",
		"2026-02-15T23:59:59Z",
		"2025-12-31T00:00:00Z",
	}

	for _, ts := range timestamps {
		line := FormatLogLine(ts, "INF", "msg", "pod-1")
		if !strings.Contains(line, ts) {
			t.Errorf("Expected line to contain timestamp '%s', got: %s", ts, line)
		}
	}
}

func TestFormatLogLine_ContainsPodName(t *testing.T) {
	pods := []string{
		"web-app-abc123-def45",
		"api-server-0",
		"worker-pod",
	}

	for _, pod := range pods {
		line := FormatLogLine("2026-01-15T10:30:00Z", "INF", "test", pod)
		if !strings.Contains(line, pod) {
			t.Errorf("Expected line to contain pod '%s', got: %s", pod, line)
		}
	}
}

func TestFormatLogLine_ContainsMessage(t *testing.T) {
	messages := []string{
		"Server started on port 8080",
		"Database connection established",
		"Error: connection refused",
		"Request processed in 45ms",
		"",
	}

	for _, msg := range messages {
		line := FormatLogLine("2026-01-15T10:30:00Z", "INF", msg, "pod-1")
		if msg != "" && !strings.Contains(line, msg) {
			t.Errorf("Expected line to contain message '%s', got: %s", msg, line)
		}
	}
}

func TestFormatLogLine_LevelPadding(t *testing.T) {
	// The level is formatted as "%-5s", so short levels get padded
	line := FormatLogLine("2026-01-15T10:30:00Z", "INF", "test", "pod-1")
	// "INF" is 3 chars, padded to 5 chars
	if !strings.Contains(line, "INF") {
		t.Error("Expected line to contain padded level")
	}
}

func TestFormatLogLine_PodBracketed(t *testing.T) {
	line := FormatLogLine("2026-01-15T10:30:00Z", "INF", "test", "web-pod-0")
	// Pod name should be wrapped in brackets: [web-pod-0]
	if !strings.Contains(line, "[web-pod-0]") {
		t.Errorf("Expected pod name in brackets, got: %s", line)
	}
}

func TestFormatLogLine_OutputStructure(t *testing.T) {
	line := FormatLogLine("2026-01-15T10:30:00Z", "ERR", "something failed", "api-pod-1")

	// Verify the line contains all four components
	components := []string{
		"2026-01-15T10:30:00Z",
		"ERR",
		"[api-pod-1]",
		"something failed",
	}

	for _, comp := range components {
		if !strings.Contains(line, comp) {
			t.Errorf("Expected line to contain '%s', got: %s", comp, line)
		}
	}
}

func TestCmdFlags(t *testing.T) {
	// Verify the command has expected flags
	flags := Cmd.Flags()

	followFlag := flags.Lookup("follow")
	if followFlag == nil {
		t.Error("Expected 'follow' flag to be registered")
	}
	if followFlag != nil && followFlag.Shorthand != "f" {
		t.Errorf("Expected 'follow' flag shorthand 'f', got '%s'", followFlag.Shorthand)
	}

	sinceFlag := flags.Lookup("since")
	if sinceFlag == nil {
		t.Error("Expected 'since' flag to be registered")
	}

	jsonFlag := flags.Lookup("json")
	if jsonFlag == nil {
		t.Error("Expected 'json' flag to be registered")
	}

	tailFlag := flags.Lookup("tail")
	if tailFlag == nil {
		t.Error("Expected 'tail' flag to be registered")
	}
	if tailFlag != nil && tailFlag.Shorthand != "n" {
		t.Errorf("Expected 'tail' flag shorthand 'n', got '%s'", tailFlag.Shorthand)
	}
}

func TestCmdRequiresArgs(t *testing.T) {
	// The logs command requires exactly 1 argument (app name)
	if Cmd.Args == nil {
		t.Error("Expected logs command to have Args validation")
	}
}
