package promclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryInstantVector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query().Get("query")
		if q == "" {
			t.Error("Expected non-empty query parameter")
		}

		resp := queryResponse{
			Status: "success",
		}
		resp.Data.ResultType = "vector"
		results := []vectorResult{
			{
				Metric: map[string]string{"pod": "app-1-abc"},
				Value:  [2]json.RawMessage{json.RawMessage(`1700000000`), json.RawMessage(`"0.25"`)},
			},
			{
				Metric: map[string]string{"pod": "app-1-def"},
				Value:  [2]json.RawMessage{json.RawMessage(`1700000000`), json.RawMessage(`"0.75"`)},
			},
		}
		resp.Data.Result, _ = json.Marshal(results)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	val, err := client.QueryInstant(context.Background(), `rate(cpu[5m])`)
	if err != nil {
		t.Fatalf("QueryInstant failed: %v", err)
	}
	if val != 1.0 {
		t.Errorf("Expected sum 1.0, got %f", val)
	}
}

func TestQueryInstantEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := queryResponse{Status: "success"}
		resp.Data.ResultType = "vector"
		resp.Data.Result = json.RawMessage(`[]`)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	val, err := client.QueryInstant(context.Background(), `nonexistent_metric`)
	if err != nil {
		t.Fatalf("QueryInstant failed: %v", err)
	}
	if val != 0 {
		t.Errorf("Expected 0 for empty result, got %f", val)
	}
}

func TestQueryInstantScalar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := queryResponse{Status: "success"}
		resp.Data.ResultType = "scalar"
		resp.Data.Result = json.RawMessage(`[1700000000, "42"]`)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	val, err := client.QueryInstant(context.Background(), `scalar(1)`)
	if err != nil {
		t.Fatalf("QueryInstant failed: %v", err)
	}
	// scalar type not handled, should return 0
	if val != 0 {
		t.Errorf("Expected 0 for scalar type, got %f", val)
	}
}

func TestQueryInstantError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := queryResponse{
			Status:    "error",
			Error:     "bad query",
			ErrorType: "bad_data",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.QueryInstant(context.Background(), `invalid{`)
	if err == nil {
		t.Fatal("Expected error for failed query")
	}
}

func TestQueryRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query_range" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		q := r.URL.Query()
		if q.Get("start") == "" || q.Get("end") == "" || q.Get("step") == "" {
			t.Error("Missing start/end/step params")
		}

		resp := queryResponse{Status: "success"}
		resp.Data.ResultType = "matrix"
		results := []matrixResult{
			{
				Metric: map[string]string{"pod": "app-1"},
				Values: [][2]json.RawMessage{
					{json.RawMessage(`1700000000`), json.RawMessage(`"0.5"`)},
					{json.RawMessage(`1700000060`), json.RawMessage(`"0.7"`)},
				},
			},
			{
				Metric: map[string]string{"pod": "app-2"},
				Values: [][2]json.RawMessage{
					{json.RawMessage(`1700000000`), json.RawMessage(`"0.3"`)},
					{json.RawMessage(`1700000060`), json.RawMessage(`"0.1"`)},
				},
			},
		}
		resp.Data.Result, _ = json.Marshal(results)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	start := time.Unix(1700000000, 0)
	end := time.Unix(1700000120, 0)
	points, err := client.QueryRange(context.Background(), `rate(cpu[5m])`, start, end, 60*time.Second)
	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("Expected 2 aggregated data points, got %d", len(points))
	}
	// First timestamp should aggregate 0.5 + 0.3 = 0.8
	if diff := points[0].Value - 0.8; diff > 0.001 || diff < -0.001 {
		t.Errorf("Expected aggregated value ~0.8 at t=0, got %f", points[0].Value)
	}
	// Second timestamp should aggregate 0.7 + 0.1 = 0.8
	if diff := points[1].Value - 0.8; diff > 0.001 || diff < -0.001 {
		t.Errorf("Expected aggregated value ~0.8 at t=60, got %f", points[1].Value)
	}
}

func TestQueryRangeEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := queryResponse{Status: "success"}
		resp.Data.ResultType = "matrix"
		resp.Data.Result = json.RawMessage(`[]`)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	points, err := client.QueryRange(context.Background(), `nonexistent`, time.Now().Add(-time.Hour), time.Now(), time.Minute)
	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}
	if len(points) != 0 {
		t.Errorf("Expected 0 points, got %d", len(points))
	}
}

func TestQueryRangeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := queryResponse{
			Status:    "error",
			Error:     "invalid expression",
			ErrorType: "bad_data",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.QueryRange(context.Background(), `invalid{`, time.Now().Add(-time.Hour), time.Now(), time.Minute)
	if err == nil {
		t.Fatal("Expected error for failed range query")
	}
}

func TestQueryRangeNotMatrix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := queryResponse{Status: "success"}
		resp.Data.ResultType = "vector"
		resp.Data.Result = json.RawMessage(`[]`)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL)
	points, err := client.QueryRange(context.Background(), `metric`, time.Now().Add(-time.Hour), time.Now(), time.Minute)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if points != nil {
		t.Errorf("Expected nil for non-matrix result, got %v", points)
	}
}

func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`"0.5"`, "0.5"},
		{`42`, "42"},
		{`""`, ""},
		{`"a"`, "a"},
	}

	for _, tt := range tests {
		got := string(trimQuotes(json.RawMessage(tt.input)))
		if got != tt.expected {
			t.Errorf("trimQuotes(%s) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestQueryInstantServerDown(t *testing.T) {
	client := New("http://127.0.0.1:1") // nothing listening
	_, err := client.QueryInstant(context.Background(), `up`)
	if err == nil {
		t.Fatal("Expected error when server is unreachable")
	}
}
