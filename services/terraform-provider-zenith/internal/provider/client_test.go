package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/apps/app-1" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer my-token" {
			t.Errorf("Unexpected auth: %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "app-1", "name": "test"})
	}))
	defer server.Close()

	client := NewZenithClient(server.URL, "my-token")
	var result map[string]string
	if err := client.Get(context.Background(), "/api/v1/apps/app-1", &result); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result["id"] != "app-1" {
		t.Errorf("Expected id app-1, got %s", result["id"])
	}
}

func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "my-app" {
			t.Errorf("Expected name my-app, got %s", body["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "new-id"})
	}))
	defer server.Close()

	client := NewZenithClient(server.URL, "tok")
	var result map[string]string
	if err := client.Post(context.Background(), "/api/v1/apps", map[string]string{"name": "my-app"}, &result); err != nil {
		t.Fatalf("Post failed: %v", err)
	}
	if result["id"] != "new-id" {
		t.Errorf("Expected id new-id, got %s", result["id"])
	}
}

func TestClientPut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewZenithClient(server.URL, "tok")
	if err := client.Put(context.Background(), "/api/v1/storage/s-1", map[string]string{"access": "public"}, nil); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
}

func TestClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/apps/app-123" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewZenithClient(server.URL, "tok")
	if err := client.Delete(context.Background(), "/api/v1/apps/app-123"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "access denied"})
	}))
	defer server.Close()

	client := NewZenithClient(server.URL, "tok")
	err := client.Get(context.Background(), "/admin", nil)
	if err == nil {
		t.Fatal("Expected error for 403")
	}
	expected := "API error (HTTP 403): access denied"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestClientAPIErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
	}))
	defer server.Close()

	client := NewZenithClient(server.URL, "tok")
	err := client.Get(context.Background(), "/api/v1/apps/nope", nil)
	if err == nil {
		t.Fatal("Expected error for 404")
	}
	expected := "API error (HTTP 404): not found"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestClientAPIErrorRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server crash"))
	}))
	defer server.Close()

	client := NewZenithClient(server.URL, "tok")
	err := client.Get(context.Background(), "/fail", nil)
	if err == nil {
		t.Fatal("Expected error for 500")
	}
	expected := "API error (HTTP 500): server crash"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestClientNilResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewZenithClient(server.URL, "tok")
	if err := client.Get(context.Background(), "/api/v1/health", nil); err != nil {
		t.Fatalf("Get with nil result failed: %v", err)
	}
}

func TestClientPostNilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	client := NewZenithClient(server.URL, "tok")
	if err := client.Post(context.Background(), "/api/v1/deploy", nil, nil); err != nil {
		t.Fatalf("Post with nil body failed: %v", err)
	}
}
