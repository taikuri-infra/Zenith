package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// --- mergePlugins tests ---

func TestMergePlugins_Empty(t *testing.T) {
	result := mergePlugins(nil, nil)
	if len(result) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(result))
	}
}

func TestMergePlugins_OnlyGroupPlugins(t *testing.T) {
	group := []entities.GatewayRoutePlugin{
		{Name: "cors", Enable: true, Config: json.RawMessage(`{}`)},
		{Name: "limit-count", Enable: true, Config: json.RawMessage(`{}`)},
	}
	result := mergePlugins(group, nil)
	if len(result) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(result))
	}
}

func TestMergePlugins_OnlyRoutePlugins(t *testing.T) {
	route := []entities.GatewayRoutePlugin{
		{Name: "jwt-auth", Enable: true, Config: json.RawMessage(`{}`)},
	}
	result := mergePlugins(nil, route)
	if len(result) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(result))
	}
}

func TestMergePlugins_RouteTakesPrecedence(t *testing.T) {
	group := []entities.GatewayRoutePlugin{
		{Name: "cors", Enable: true, Config: json.RawMessage(`{"group":true}`)},
		{Name: "limit-count", Enable: true, Config: json.RawMessage(`{}`)},
	}
	route := []entities.GatewayRoutePlugin{
		{Name: "cors", Enable: false, Config: json.RawMessage(`{"route":true}`)},
	}
	result := mergePlugins(group, route)
	if len(result) != 2 {
		t.Errorf("Expected 2 plugins (cors from route, limit-count from group), got %d", len(result))
	}
	// Find cors plugin - should be from route (enable=false)
	for _, p := range result {
		if p.Name == "cors" {
			if p.Enable {
				t.Error("Expected cors to be disabled (route takes precedence)")
			}
			break
		}
	}
}

func TestMergePlugins_NoOverlap(t *testing.T) {
	group := []entities.GatewayRoutePlugin{
		{Name: "cors", Enable: true},
	}
	route := []entities.GatewayRoutePlugin{
		{Name: "jwt-auth", Enable: true},
	}
	result := mergePlugins(group, route)
	if len(result) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(result))
	}
}

// --- gwTimeRangeParams tests ---

func TestGwTimeRangeParams_1h(t *testing.T) {
	start, step := gwTimeRangeParams("1h")
	if time.Since(start) < 59*time.Minute || time.Since(start) > 61*time.Minute {
		t.Errorf("Expected ~1h ago, got %v ago", time.Since(start))
	}
	if step != 30*time.Second {
		t.Errorf("Expected 30s step, got %v", step)
	}
}

func TestGwTimeRangeParams_6h(t *testing.T) {
	start, step := gwTimeRangeParams("6h")
	if time.Since(start) < 5*time.Hour+59*time.Minute {
		t.Error("Expected ~6h ago")
	}
	if step != 2*time.Minute {
		t.Errorf("Expected 2m step, got %v", step)
	}
}

func TestGwTimeRangeParams_24h(t *testing.T) {
	start, step := gwTimeRangeParams("24h")
	if time.Since(start) < 23*time.Hour+59*time.Minute {
		t.Error("Expected ~24h ago")
	}
	if step != 5*time.Minute {
		t.Errorf("Expected 5m step, got %v", step)
	}
}

func TestGwTimeRangeParams_7d(t *testing.T) {
	start, step := gwTimeRangeParams("7d")
	if time.Since(start) < 6*24*time.Hour+23*time.Hour {
		t.Error("Expected ~7d ago")
	}
	if step != 30*time.Minute {
		t.Errorf("Expected 30m step, got %v", step)
	}
}

func TestGwTimeRangeParams_Default(t *testing.T) {
	start, step := gwTimeRangeParams("unknown")
	if time.Since(start) < 59*time.Minute || time.Since(start) > 61*time.Minute {
		t.Error("Expected ~1h ago for default")
	}
	if step != 30*time.Second {
		t.Errorf("Expected 30s step for default, got %v", step)
	}
}

// --- timeRangeParams tests (monitoring.go) ---

func TestTimeRangeParams_1h(t *testing.T) {
	start, step := timeRangeParams("1h")
	if time.Since(start) < 59*time.Minute || time.Since(start) > 61*time.Minute {
		t.Errorf("Expected ~1h ago, got %v ago", time.Since(start))
	}
	if step != 30*time.Second {
		t.Errorf("Expected 30s step, got %v", step)
	}
}

func TestTimeRangeParams_6h(t *testing.T) {
	_, step := timeRangeParams("6h")
	if step != 2*time.Minute {
		t.Errorf("Expected 2m step, got %v", step)
	}
}

func TestTimeRangeParams_24h(t *testing.T) {
	_, step := timeRangeParams("24h")
	if step != 5*time.Minute {
		t.Errorf("Expected 5m step, got %v", step)
	}
}

func TestTimeRangeParams_7d(t *testing.T) {
	_, step := timeRangeParams("7d")
	if step != 30*time.Minute {
		t.Errorf("Expected 30m step, got %v", step)
	}
}

func TestTimeRangeParams_Default(t *testing.T) {
	_, step := timeRangeParams("invalid")
	if step != 30*time.Second {
		t.Errorf("Expected 30s step for default, got %v", step)
	}
}
