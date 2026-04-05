package services

import (
	"context"
	"testing"
)

// --- NewAIErrorAnalyzer tests ---

func TestNewAIErrorAnalyzer(t *testing.T) {
	analyzer := NewAIErrorAnalyzer(nil, nil)
	if analyzer == nil {
		t.Fatal("Expected non-nil AIErrorAnalyzer")
	}
}

func TestNewAIErrorAnalyzer_WithClient(t *testing.T) {
	client := NewAIClient("http://localhost", "key", "model", true)
	analyzer := NewAIErrorAnalyzer(client, nil)
	if analyzer == nil {
		t.Fatal("Expected non-nil AIErrorAnalyzer")
	}
}

// --- AnalyzeError tests ---

func TestAnalyzeError_NilClient(t *testing.T) {
	analyzer := NewAIErrorAnalyzer(nil, nil)
	analysis, resp, err := analyzer.AnalyzeError(context.Background(), "my-app", "default", 100)
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}
	if analysis != nil || resp != nil {
		t.Error("Expected nil analysis and resp for nil AI client")
	}
}

func TestAnalyzeError_DisabledClient(t *testing.T) {
	client := NewAIClient("http://localhost", "key", "model", false)
	analyzer := NewAIErrorAnalyzer(client, nil)
	analysis, resp, err := analyzer.AnalyzeError(context.Background(), "my-app", "default", 50)
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}
	if analysis != nil || resp != nil {
		t.Error("Expected nil analysis and resp for disabled AI client")
	}
}

func TestAnalyzeError_NilLokiClient(t *testing.T) {
	client := NewAIClient("http://localhost", "key", "model", true)
	analyzer := NewAIErrorAnalyzer(client, nil)
	// With nil Loki client, rawLogs will be empty, returns nil
	analysis, resp, err := analyzer.AnalyzeError(context.Background(), "my-app", "default", 100)
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}
	if analysis != nil || resp != nil {
		t.Error("Expected nil for nil Loki client (no logs to analyze)")
	}
}

func TestAnalyzeError_DefaultLogLines(t *testing.T) {
	client := NewAIClient("http://localhost", "key", "model", true)
	analyzer := NewAIErrorAnalyzer(client, nil)
	// logLines <= 0 should default to 100, no panic
	analysis, _, _ := analyzer.AnalyzeError(context.Background(), "app", "ns", 0)
	if analysis != nil {
		t.Error("Expected nil analysis without Loki")
	}
}

// --- parseJSONResponse tests ---

func TestParseJSONResponse_Direct(t *testing.T) {
	var result ErrorAnalysis
	err := parseJSONResponse(`{"problem":"OOM","cause":"heap","fix":"increase memory","confidence":"high"}`, &result)
	if err != nil {
		t.Fatalf("parseJSONResponse failed: %v", err)
	}
	if result.Problem != "OOM" {
		t.Errorf("Expected problem 'OOM', got '%s'", result.Problem)
	}
	if result.Confidence != "high" {
		t.Errorf("Expected confidence 'high', got '%s'", result.Confidence)
	}
}

func TestParseJSONResponse_MarkdownFenceJSON(t *testing.T) {
	input := "```json\n{\"problem\":\"crash\",\"cause\":\"bug\",\"fix\":\"fix it\",\"confidence\":\"medium\"}\n```"
	var result ErrorAnalysis
	err := parseJSONResponse(input, &result)
	if err != nil {
		t.Fatalf("parseJSONResponse with json fence failed: %v", err)
	}
	if result.Problem != "crash" {
		t.Errorf("Expected problem 'crash', got '%s'", result.Problem)
	}
}

func TestParseJSONResponse_MarkdownFencePlain(t *testing.T) {
	input := "```\n{\"problem\":\"timeout\",\"cause\":\"slow\",\"fix\":\"optimize\",\"confidence\":\"low\"}\n```"
	var result ErrorAnalysis
	err := parseJSONResponse(input, &result)
	if err != nil {
		t.Fatalf("parseJSONResponse with plain fence failed: %v", err)
	}
	if result.Problem != "timeout" {
		t.Errorf("Expected problem 'timeout', got '%s'", result.Problem)
	}
}

func TestParseJSONResponse_InvalidJSON(t *testing.T) {
	var result ErrorAnalysis
	err := parseJSONResponse("not json at all", &result)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseJSONResponse_WhitespaceWrapped(t *testing.T) {
	input := "  \n  {\"problem\":\"err\",\"cause\":\"x\",\"fix\":\"y\",\"confidence\":\"high\"}\n  "
	var result ErrorAnalysis
	err := parseJSONResponse(input, &result)
	if err != nil {
		t.Fatalf("parseJSONResponse with whitespace failed: %v", err)
	}
	if result.Problem != "err" {
		t.Errorf("Expected problem 'err', got '%s'", result.Problem)
	}
}

func TestParseJSONResponse_Array(t *testing.T) {
	var result []string
	err := parseJSONResponse(`["suggestion1", "suggestion2"]`, &result)
	if err != nil {
		t.Fatalf("parseJSONResponse array failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}
}

// --- jsonUnmarshalImpl tests ---

func TestJsonUnmarshalImpl(t *testing.T) {
	var result map[string]string
	err := jsonUnmarshalImpl([]byte(`{"key":"value"}`), &result)
	if err != nil {
		t.Fatalf("jsonUnmarshalImpl failed: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("Expected value 'value', got '%s'", result["key"])
	}
}

func TestJsonUnmarshalImpl_Invalid(t *testing.T) {
	var result map[string]string
	err := jsonUnmarshalImpl([]byte("not json"), &result)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
