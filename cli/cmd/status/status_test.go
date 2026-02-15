package status

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderProgressBar(t *testing.T) {
	// Low usage - should not panic
	bar := RenderProgressBar(25.0, 20)
	if bar == "" {
		t.Error("Expected non-empty progress bar")
	}

	// High usage
	bar = RenderProgressBar(85.0, 20)
	if bar == "" {
		t.Error("Expected non-empty progress bar")
	}

	// Zero
	bar = RenderProgressBar(0, 20)
	if bar == "" {
		t.Error("Expected non-empty progress bar")
	}

	// Full
	bar = RenderProgressBar(100.0, 20)
	if bar == "" {
		t.Error("Expected non-empty progress bar")
	}

	// Over 100
	bar = RenderProgressBar(150.0, 20)
	if bar == "" {
		t.Error("Expected non-empty progress bar for over 100%")
	}
}

func TestRenderTable_Empty(t *testing.T) {
	result := renderTable(
		[]string{"NAME", "STATUS"},
		[][]string{},
		lipglossTestHeaderStyle(),
		lipglossTestCellStyle(),
	)
	if result == "" {
		t.Error("Expected non-empty table render")
	}
}

func TestRenderTable_WithRows(t *testing.T) {
	result := renderTable(
		[]string{"NAME", "STATUS"},
		[][]string{
			{"my-app", "Running"},
			{"api", "Pending"},
		},
		lipglossTestHeaderStyle(),
		lipglossTestCellStyle(),
	)
	if result == "" {
		t.Error("Expected non-empty table render")
	}
}

// Helper functions for testing without importing lipgloss in tests
func lipglossTestHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Padding(0, 1)
}

func lipglossTestCellStyle() lipgloss.Style {
	return lipgloss.NewStyle().Padding(0, 1)
}
