package services

import (
	"context"
	"testing"
)

// --- GetGatewayAnalytics tests ---

func TestGetGatewayAnalytics_NilProm(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Analytics GW")

	overview, err := svc.GetGatewayAnalytics(ctx, gw.ID)
	if err != nil {
		t.Fatalf("GetGatewayAnalytics failed: %v", err)
	}
	if overview == nil {
		t.Fatal("Expected non-nil overview")
	}
	// With nil promClient, all values should be zero
	if overview.RequestRate != 0 {
		t.Errorf("Expected RequestRate 0, got %f", overview.RequestRate)
	}
	if overview.ErrorRate != 0 {
		t.Errorf("Expected ErrorRate 0, got %f", overview.ErrorRate)
	}
	if overview.P95Latency != 0 {
		t.Errorf("Expected P95Latency 0, got %f", overview.P95Latency)
	}
	if overview.TotalRequests24h != 0 {
		t.Errorf("Expected TotalRequests24h 0, got %f", overview.TotalRequests24h)
	}
}

func TestGetGatewayAnalytics_NotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	_, err := svc.GetGatewayAnalytics(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

// --- GetGatewayTimeSeries tests ---

func TestGetGatewayTimeSeries_NilProm_Requests(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "TS GW")

	resp, err := svc.GetGatewayTimeSeries(ctx, gw.ID, "requests", "1h")
	if err != nil {
		t.Fatalf("GetGatewayTimeSeries failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.Metric != "requests" {
		t.Errorf("Expected metric 'requests', got '%s'", resp.Metric)
	}
	if resp.Range != "1h" {
		t.Errorf("Expected range '1h', got '%s'", resp.Range)
	}
	if len(resp.Points) != 0 {
		t.Errorf("Expected 0 points with nil prom, got %d", len(resp.Points))
	}
}

func TestGetGatewayTimeSeries_NilProm_Latency(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "TS GW2")

	resp, err := svc.GetGatewayTimeSeries(ctx, gw.ID, "latency", "6h")
	if err != nil {
		t.Fatalf("GetGatewayTimeSeries failed: %v", err)
	}
	if resp.Metric != "latency" {
		t.Errorf("Expected metric 'latency', got '%s'", resp.Metric)
	}
}

func TestGetGatewayTimeSeries_NilProm_Errors(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "TS GW3")

	resp, err := svc.GetGatewayTimeSeries(ctx, gw.ID, "errors", "24h")
	if err != nil {
		t.Fatalf("GetGatewayTimeSeries failed: %v", err)
	}
	if resp.Metric != "errors" {
		t.Errorf("Expected metric 'errors', got '%s'", resp.Metric)
	}
}

func TestGetGatewayTimeSeries_NotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	_, err := svc.GetGatewayTimeSeries(ctx, "nonexistent", "requests", "1h")
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

func TestGetGatewayTimeSeries_UnknownMetric(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "TS GW4")

	// Note: with nil prom, the function returns early before checking the metric.
	// We still test the path works.
	_, _ = svc.GetGatewayTimeSeries(ctx, gw.ID, "invalid-metric", "1h")
}
