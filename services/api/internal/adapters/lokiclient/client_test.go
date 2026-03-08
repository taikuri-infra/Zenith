package lokiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/query_range" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		q := r.URL.Query()
		if q.Get("query") == "" {
			t.Error("Expected non-empty query")
		}
		if q.Get("direction") != "backward" {
			t.Errorf("Expected direction=backward, got %s", q.Get("direction"))
		}
		if q.Get("limit") != "10" {
			t.Errorf("Expected limit=10, got %s", q.Get("limit"))
		}

		resp := lokiResponse{Status: "success"}
		resp.Data.ResultType = "streams"
		streams := []lokiStream{
			{
				Stream: map[string]string{"app": "my-app", "level": "info"},
				Values: [][2]string{
					{"1700000000000000000", "Starting server on :8080"},
					{"1700000001000000000", "Ready to accept connections"},
				},
			},
			{
				Stream: map[string]string{"app": "my-app", "level": "error"},
				Values: [][2]string{
					{"1700000002000000000", "Connection refused to database"},
				},
			},
		}
		resp.Data.Result, _ = json.Marshal(streams)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	entries, err := client.QueryRange(
		context.Background(),
		`{app="my-app"}`,
		time.Unix(1700000000, 0),
		time.Unix(1700000010, 0),
		10,
	)
	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("Expected 3 log entries, got %d", len(entries))
	}

	// Check first entry
	if entries[0].Line != "Starting server on :8080" {
		t.Errorf("Unexpected first line: %s", entries[0].Line)
	}
	if entries[0].Labels["level"] != "info" {
		t.Errorf("Expected level=info, got %s", entries[0].Labels["level"])
	}
	if entries[0].Timestamp != time.Unix(0, 1700000000000000000).UTC() {
		t.Errorf("Unexpected timestamp: %v", entries[0].Timestamp)
	}

	// Check last entry
	if entries[2].Line != "Connection refused to database" {
		t.Errorf("Unexpected last line: %s", entries[2].Line)
	}
	if entries[2].Labels["level"] != "error" {
		t.Errorf("Expected level=error, got %s", entries[2].Labels["level"])
	}
}

func TestQueryRangeDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("limit"); q != "100" {
			t.Errorf("Expected default limit=100, got %s", q)
		}
		resp := lokiResponse{Status: "success"}
		resp.Data.ResultType = "streams"
		resp.Data.Result = json.RawMessage(`[]`)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.QueryRange(context.Background(), `{app="x"}`, time.Now().Add(-time.Hour), time.Now(), 0)
	if err != nil {
		t.Fatalf("QueryRange with default limit failed: %v", err)
	}
}

func TestQueryRangeNegativeLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("limit"); q != "100" {
			t.Errorf("Expected default limit=100 for negative, got %s", q)
		}
		resp := lokiResponse{Status: "success"}
		resp.Data.ResultType = "streams"
		resp.Data.Result = json.RawMessage(`[]`)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.QueryRange(context.Background(), `{app="x"}`, time.Now().Add(-time.Hour), time.Now(), -5)
	if err != nil {
		t.Fatalf("QueryRange with negative limit failed: %v", err)
	}
}

func TestQueryRangeEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := lokiResponse{Status: "success"}
		resp.Data.ResultType = "streams"
		resp.Data.Result = json.RawMessage(`[]`)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	entries, err := client.QueryRange(context.Background(), `{app="none"}`, time.Now().Add(-time.Hour), time.Now(), 50)
	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestQueryRangeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := lokiResponse{Status: "error"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.QueryRange(context.Background(), `{bad`, time.Now().Add(-time.Hour), time.Now(), 10)
	if err == nil {
		t.Fatal("Expected error for failed query")
	}
}

func TestQueryRangeServerDown(t *testing.T) {
	client := New("http://127.0.0.1:1")
	_, err := client.QueryRange(context.Background(), `{app="x"}`, time.Now().Add(-time.Hour), time.Now(), 10)
	if err == nil {
		t.Fatal("Expected error when server unreachable")
	}
}

func TestTailCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := lokiResponse{Status: "success"}
		resp.Data.ResultType = "streams"
		streams := []lokiStream{
			{
				Stream: map[string]string{"app": "test"},
				Values: [][2]string{
					{"1700000000000000000", "tail entry"},
				},
			},
		}
		resp.Data.Result, _ = json.Marshal(streams)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	out := make(chan LogEntry, 100)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := client.Tail(ctx, `{app="test"}`, out)
	if err == nil || err != context.DeadlineExceeded {
		t.Errorf("Expected deadline exceeded, got %v", err)
	}

	// Should have received at least one entry
	if len(out) == 0 {
		t.Error("Expected at least one tailed entry")
	}
}

func TestQueryRangeInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.QueryRange(context.Background(), `{app="x"}`, time.Now().Add(-time.Hour), time.Now(), 10)
	if err == nil {
		t.Fatal("Expected error for invalid JSON response")
	}
}
