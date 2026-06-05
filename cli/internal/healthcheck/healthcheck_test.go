package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWaitUntilHealthy_ImmediateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := WaitUntilHealthy(ctx, Options{
		URL:      srv.URL,
		Interval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitUntilHealthy_EventualSuccess(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(503)
		} else {
			w.WriteHeader(200)
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := WaitUntilHealthy(ctx, Options{
		URL:      srv.URL,
		Interval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls < 3 {
		t.Errorf("expected at least 3 calls, got %d", calls)
	}
}

func TestWaitUntilHealthy_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := WaitUntilHealthy(ctx, Options{
		URL:      srv.URL,
		Interval: 50 * time.Millisecond,
	})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestProbe_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	if !Probe(srv.URL) {
		t.Error("expected Probe to return true for 200 response")
	}
}

func TestProbe_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	if Probe(srv.URL) {
		t.Error("expected Probe to return false for 503 response")
	}
}

func TestProbe_ConnectionRefused(t *testing.T) {
	if Probe("http://127.0.0.1:19997/health") {
		t.Error("expected Probe to return false for connection refused")
	}
}

func TestWaitUntilHealthy_DefaultInterval(t *testing.T) {
	// Verify defaults don't panic and work correctly
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(200)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := WaitUntilHealthy(ctx, Options{URL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls == 0 {
		t.Error("expected at least one health check call")
	}
}
