package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- NewAIClient tests ---

func TestNewAIClient(t *testing.T) {
	client := NewAIClient("http://localhost:8080", "test-key", "gpt-4o-mini", true)
	if client == nil {
		t.Fatal("Expected non-nil AIClient")
	}
}

func TestNewAIClient_Disabled(t *testing.T) {
	client := NewAIClient("http://localhost:8080", "", "gpt-4o-mini", false)
	if client == nil {
		t.Fatal("Expected non-nil AIClient even when disabled")
	}
	if client.IsEnabled() {
		t.Error("Expected IsEnabled=false for disabled client")
	}
}

// --- IsEnabled tests ---

func TestAIClient_IsEnabled_True(t *testing.T) {
	client := NewAIClient("http://localhost:8080", "key", "model", true)
	if !client.IsEnabled() {
		t.Error("Expected IsEnabled=true")
	}
}

func TestAIClient_IsEnabled_Nil(t *testing.T) {
	var client *AIClient
	if client.IsEnabled() {
		t.Error("Expected IsEnabled=false for nil client")
	}
}

// --- ModelName tests ---

func TestAIClient_ModelName(t *testing.T) {
	client := NewAIClient("http://localhost:8080", "key", "gpt-4o-mini", true)
	if client.ModelName() != "gpt-4o-mini" {
		t.Errorf("Expected model name 'gpt-4o-mini', got '%s'", client.ModelName())
	}
}

func TestAIClient_ModelName_Nil(t *testing.T) {
	var client *AIClient
	if client.ModelName() != "" {
		t.Error("Expected empty model name for nil client")
	}
}

// --- Complete tests ---

func TestAIClient_Complete_Disabled(t *testing.T) {
	client := NewAIClient("http://localhost:8080", "key", "model", false)
	ctx := context.Background()

	resp, err := client.Complete(ctx, "system", "user")
	if err != nil {
		t.Fatalf("Expected nil error for disabled client, got: %v", err)
	}
	if resp != nil {
		t.Error("Expected nil response for disabled client")
	}
}

func TestAIClient_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization header 'Bearer test-key', got '%s'", r.Header.Get("Authorization"))
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "Hello!"}},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
			},
			"model": "gpt-4o-mini",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "test-key", "gpt-4o-mini", true)
	ctx := context.Background()

	resp, err := client.Complete(ctx, "You are helpful", "Say hello")
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.Content != "Hello!" {
		t.Errorf("Expected content 'Hello!', got '%s'", resp.Content)
	}
	if resp.TokensIn != 10 {
		t.Errorf("Expected 10 input tokens, got %d", resp.TokensIn)
	}
	if resp.TokensOut != 5 {
		t.Errorf("Expected 5 output tokens, got %d", resp.TokensOut)
	}
	if resp.Model != "gpt-4o-mini" {
		t.Errorf("Expected model 'gpt-4o-mini', got '%s'", resp.Model)
	}
}

func TestAIClient_Complete_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "model", true)
	ctx := context.Background()

	resp, err := client.Complete(ctx, "system", "user")
	if err != nil {
		t.Fatalf("Expected nil error for non-200, got: %v", err)
	}
	if resp != nil {
		t.Error("Expected nil response for non-200 status")
	}
}

func TestAIClient_Complete_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []interface{}{},
			"usage":   map[string]int{},
			"model":   "model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "model", true)
	ctx := context.Background()

	resp, err := client.Complete(ctx, "system", "user")
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}
	if resp != nil {
		t.Error("Expected nil response for empty choices")
	}
}

func TestAIClient_Complete_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "model", true)
	ctx := context.Background()

	resp, err := client.Complete(ctx, "system", "user")
	if err != nil {
		t.Fatalf("Expected nil error (graceful), got: %v", err)
	}
	if resp != nil {
		t.Error("Expected nil response for invalid JSON")
	}
}

func TestAIClient_Complete_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("Expected no Authorization header when apiKey is empty")
		}
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "ok"}},
			},
			"usage": map[string]int{},
			"model": "model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "", "model", true)
	ctx := context.Background()

	resp, err := client.Complete(ctx, "system", "user")
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}
	if resp == nil || resp.Content != "ok" {
		t.Error("Expected response content 'ok'")
	}
}

// --- CompleteJSON tests ---

func TestAIClient_CompleteJSON_Disabled(t *testing.T) {
	client := NewAIClient("http://localhost", "key", "model", false)
	ctx := context.Background()

	var dest map[string]string
	resp, err := client.CompleteJSON(ctx, "system", "user", &dest)
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}
	if resp != nil {
		t.Error("Expected nil response for disabled client")
	}
}

func TestAIClient_CompleteJSON_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": `{"name":"test","value":"42"}`}},
			},
			"usage": map[string]int{},
			"model": "model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "model", true)
	ctx := context.Background()

	var dest map[string]string
	resp, err := client.CompleteJSON(ctx, "system", "user", &dest)
	if err != nil {
		t.Fatalf("CompleteJSON failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if dest["name"] != "test" {
		t.Errorf("Expected name 'test', got '%s'", dest["name"])
	}
}

func TestAIClient_CompleteJSON_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "not json at all"}},
			},
			"usage": map[string]int{},
			"model": "model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAIClient(server.URL, "key", "model", true)
	ctx := context.Background()

	var dest map[string]string
	resp, err := client.CompleteJSON(ctx, "system", "user", &dest)
	if err == nil {
		t.Error("Expected error for invalid JSON content")
	}
	if resp == nil {
		t.Error("Expected response even on parse error")
	}
}
