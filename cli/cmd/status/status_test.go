package status

import (
	"strings"
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

func TestRenderProgressBar_PercentageValues(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		width   int
	}{
		{"zero percent", 0, 20},
		{"10 percent", 10, 20},
		{"25 percent", 25, 20},
		{"50 percent", 50, 20},
		{"60 percent boundary", 60, 20},
		{"61 percent - amber zone", 61, 20},
		{"75 percent", 75, 20},
		{"80 percent boundary", 80, 20},
		{"81 percent - red zone", 81, 20},
		{"90 percent", 90, 20},
		{"100 percent", 100, 20},
		{"150 percent overflow", 150, 20},
		{"negative percent", -10, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := RenderProgressBar(tt.percent, tt.width)
			if bar == "" {
				t.Error("Expected non-empty progress bar")
			}
			// The bar should always contain a percentage display
			if !strings.Contains(bar, "%") {
				t.Errorf("Expected progress bar to contain '%%', got: %s", bar)
			}
		})
	}
}

func TestRenderProgressBar_DifferentWidths(t *testing.T) {
	widths := []int{1, 5, 10, 20, 50, 100}

	for _, w := range widths {
		bar := RenderProgressBar(50, w)
		if bar == "" {
			t.Errorf("Expected non-empty progress bar for width %d", w)
		}
	}
}

func TestRenderProgressBar_ColorThresholds(t *testing.T) {
	// Below 60% should be emerald (green)
	// 60-80% should be amber
	// Above 80% should be red
	// We can test that the function doesn't panic at threshold boundaries
	boundaries := []float64{0, 59.9, 60, 60.1, 79.9, 80, 80.1, 100}

	for _, pct := range boundaries {
		bar := RenderProgressBar(pct, 20)
		if bar == "" {
			t.Errorf("Expected non-empty bar at %.1f%%", pct)
		}
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
	if !strings.Contains(result, "No resources found") {
		t.Error("Expected 'No resources found' message for empty table")
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

func TestRenderTable_HeaderContent(t *testing.T) {
	headers := []string{"NAME", "STATUS", "REPLICAS"}
	result := renderTable(
		headers,
		[][]string{
			{"web-app", "Running", "3/3"},
		},
		lipglossTestHeaderStyle(),
		lipglossTestCellStyle(),
	)

	for _, h := range headers {
		if !strings.Contains(result, h) {
			t.Errorf("Expected table to contain header '%s'", h)
		}
	}
}

func TestRenderTable_RowContent(t *testing.T) {
	result := renderTable(
		[]string{"NAME", "ENGINE", "STATUS"},
		[][]string{
			{"my-db", "postgresql", "Ready"},
			{"cache", "redis", "Running"},
		},
		lipglossTestHeaderStyle(),
		lipglossTestCellStyle(),
	)

	// Verify row data appears in output
	if !strings.Contains(result, "my-db") {
		t.Error("Expected table to contain 'my-db'")
	}
	if !strings.Contains(result, "postgresql") {
		t.Error("Expected table to contain 'postgresql'")
	}
	if !strings.Contains(result, "cache") {
		t.Error("Expected table to contain 'cache'")
	}
}

func TestRenderTable_StatusColumnColoring(t *testing.T) {
	// Verify the table renders without panic when STATUS column has various values
	statuses := []string{"Running", "Pending", "Failed", "Stopped", "Ready", "Creating", "Unknown"}

	for _, status := range statuses {
		result := renderTable(
			[]string{"NAME", "STATUS"},
			[][]string{
				{"test-app", status},
			},
			lipglossTestHeaderStyle(),
			lipglossTestCellStyle(),
		)
		if result == "" {
			t.Errorf("Expected non-empty table render for status '%s'", status)
		}
	}
}

func TestRenderTable_WideColumns(t *testing.T) {
	result := renderTable(
		[]string{"NAME", "IMAGE", "DOMAIN"},
		[][]string{
			{"very-long-application-name-here", "registry.example.com/org/app:v2.1.0", "very-long-domain-name.example.com"},
		},
		lipglossTestHeaderStyle(),
		lipglossTestCellStyle(),
	)
	if result == "" {
		t.Error("Expected non-empty table render with wide columns")
	}
}

func TestRenderTable_SingleRow(t *testing.T) {
	result := renderTable(
		[]string{"NAME", "STATUS"},
		[][]string{
			{"only-app", "Running"},
		},
		lipglossTestHeaderStyle(),
		lipglossTestCellStyle(),
	)
	if result == "" {
		t.Error("Expected non-empty table render")
	}
	if !strings.Contains(result, "only-app") {
		t.Error("Expected table to contain 'only-app'")
	}
}

func TestRenderTable_ManyRows(t *testing.T) {
	rows := make([][]string, 10)
	for i := range rows {
		rows[i] = []string{
			strings.Repeat("a", i+1),
			"Running",
		}
	}

	result := renderTable(
		[]string{"NAME", "STATUS"},
		rows,
		lipglossTestHeaderStyle(),
		lipglossTestCellStyle(),
	)
	if result == "" {
		t.Error("Expected non-empty table render")
	}
}

func TestResourceStatus_StructFields(t *testing.T) {
	rs := ResourceStatus{
		Name:     "web-app",
		Type:     "App",
		Status:   "Running",
		Replicas: "3/3",
		CPU:      "250m",
		Memory:   "128Mi",
		Extra:    "nginx:latest",
	}

	if rs.Name != "web-app" {
		t.Errorf("Expected Name 'web-app', got '%s'", rs.Name)
	}
	if rs.Type != "App" {
		t.Errorf("Expected Type 'App', got '%s'", rs.Type)
	}
	if rs.Status != "Running" {
		t.Errorf("Expected Status 'Running', got '%s'", rs.Status)
	}
	if rs.Replicas != "3/3" {
		t.Errorf("Expected Replicas '3/3', got '%s'", rs.Replicas)
	}
	if rs.CPU != "250m" {
		t.Errorf("Expected CPU '250m', got '%s'", rs.CPU)
	}
	if rs.Memory != "128Mi" {
		t.Errorf("Expected Memory '128Mi', got '%s'", rs.Memory)
	}
	if rs.Extra != "nginx:latest" {
		t.Errorf("Expected Extra 'nginx:latest', got '%s'", rs.Extra)
	}
}

func TestResourceStatus_DifferentStates(t *testing.T) {
	states := []struct {
		status   string
		replicas string
	}{
		{"Running", "3/3"},
		{"Pending", "0/3"},
		{"Failed", "0/1"},
		{"Stopped", "0/0"},
		{"Creating", "0/1"},
	}

	for _, s := range states {
		rs := ResourceStatus{
			Name:     "test-app",
			Status:   s.status,
			Replicas: s.replicas,
		}
		if rs.Status != s.status {
			t.Errorf("Expected status '%s', got '%s'", s.status, rs.Status)
		}
	}
}

// Helper functions for testing without importing lipgloss in tests
func lipglossTestHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Padding(0, 1)
}

func lipglossTestCellStyle() lipgloss.Style {
	return lipgloss.NewStyle().Padding(0, 1)
}
