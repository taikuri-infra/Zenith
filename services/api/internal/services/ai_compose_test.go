package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- NewAIComposeValidator tests ---

func TestNewAIComposeValidator(t *testing.T) {
	v := NewAIComposeValidator(nil)
	if v == nil {
		t.Fatal("Expected non-nil AIComposeValidator")
	}
}

func TestNewAIComposeValidator_WithClient(t *testing.T) {
	client := NewAIClient("http://localhost", "key", "model", true)
	v := NewAIComposeValidator(client)
	if v == nil {
		t.Fatal("Expected non-nil AIComposeValidator")
	}
}

// --- ValidateCompose tests ---

func TestValidateCompose_NilClient(t *testing.T) {
	v := NewAIComposeValidator(nil)
	result := v.ValidateCompose(context.Background(), "version: '3'\nservices:\n  web:\n    image: nginx")
	if result != nil {
		t.Error("Expected nil result for nil AI client")
	}
}

func TestValidateCompose_DisabledClient(t *testing.T) {
	client := NewAIClient("http://localhost", "key", "model", false)
	v := NewAIComposeValidator(client)
	result := v.ValidateCompose(context.Background(), "version: '3'")
	if result != nil {
		t.Error("Expected nil result for disabled AI client")
	}
}

func TestValidateCompose_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suggestions := []string{"Line 5: port format should be 'HOST:CONTAINER'"}
		sugJSON, _ := json.Marshal(suggestions)
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": string(sugJSON)}},
			},
			"usage": map[string]int{},
			"model": "model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "model", true)
	v := NewAIComposeValidator(client)
	result := v.ValidateCompose(context.Background(), "bad compose")
	if len(result) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result))
	}
	if result[0] != "Line 5: port format should be 'HOST:CONTAINER'" {
		t.Errorf("Unexpected suggestion: %s", result[0])
	}
}

func TestValidateCompose_EmptySuggestions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "[]"}},
			},
			"usage": map[string]int{},
			"model": "model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "model", true)
	v := NewAIComposeValidator(client)
	result := v.ValidateCompose(context.Background(), "good compose")
	if len(result) != 0 {
		t.Errorf("Expected 0 suggestions, got %d", len(result))
	}
}

func TestValidateCompose_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "not a json array"}},
			},
			"usage": map[string]int{},
			"model": "model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "model", true)
	v := NewAIComposeValidator(client)
	result := v.ValidateCompose(context.Background(), "compose")
	if result != nil {
		t.Error("Expected nil for unparseable AI response")
	}
}

// --- FormatCompose tests ---

func TestFormatCompose_NilClient(t *testing.T) {
	v := NewAIComposeValidator(nil)
	result := v.FormatCompose(context.Background(), "bad yaml")
	if result != "" {
		t.Error("Expected empty string for nil client")
	}
}

func TestFormatCompose_DisabledClient(t *testing.T) {
	client := NewAIClient("http://localhost", "key", "model", false)
	v := NewAIComposeValidator(client)
	result := v.FormatCompose(context.Background(), "bad yaml")
	if result != "" {
		t.Error("Expected empty string for disabled client")
	}
}

func TestFormatCompose_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "services:\n  web:\n    image: nginx\n"}},
			},
			"usage": map[string]int{},
			"model": "model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "model", true)
	v := NewAIComposeValidator(client)
	result := v.FormatCompose(context.Background(), "  services:\nweb:\n image: nginx")
	if result != "services:\n  web:\n    image: nginx\n" {
		t.Errorf("Unexpected formatted output: %q", result)
	}
}

// --- ValidateComposeWithUsage tests ---

func TestValidateComposeWithUsage_NilClient(t *testing.T) {
	v := NewAIComposeValidator(nil)
	suggestions, resp := v.ValidateComposeWithUsage(context.Background(), "compose")
	if suggestions != nil || resp != nil {
		t.Error("Expected nil for nil client")
	}
}

func TestValidateComposeWithUsage_DisabledClient(t *testing.T) {
	client := NewAIClient("http://localhost", "key", "model", false)
	v := NewAIComposeValidator(client)
	suggestions, resp := v.ValidateComposeWithUsage(context.Background(), "compose")
	if suggestions != nil || resp != nil {
		t.Error("Expected nil for disabled client")
	}
}

func TestValidateComposeWithUsage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suggestions := []string{"tip1", "tip2"}
		sugJSON, _ := json.Marshal(suggestions)
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": string(sugJSON)}},
			},
			"usage": map[string]int{"prompt_tokens": 50, "completion_tokens": 20},
			"model": "gpt-4o-mini",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "gpt-4o-mini", true)
	v := NewAIComposeValidator(client)
	suggestions, resp := v.ValidateComposeWithUsage(context.Background(), "compose")
	if len(suggestions) != 2 {
		t.Fatalf("Expected 2 suggestions, got %d", len(suggestions))
	}
	if resp == nil {
		t.Fatal("Expected non-nil AIResponse")
	}
	if resp.TokensIn != 50 {
		t.Errorf("Expected 50 input tokens, got %d", resp.TokensIn)
	}
}

func TestValidateComposeWithUsage_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "not json"}},
			},
			"usage": map[string]int{},
			"model": "model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "model", true)
	v := NewAIComposeValidator(client)
	suggestions, resp := v.ValidateComposeWithUsage(context.Background(), "compose")
	if suggestions != nil {
		t.Error("Expected nil suggestions for invalid JSON")
	}
	if resp == nil {
		t.Error("Expected non-nil resp even on parse failure")
	}
}
