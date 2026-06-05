package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestCFClient creates a client pointed at the given test server URL.
func newTestCFClient(serverURL string) *Client {
	c := NewClient("test-token")
	c.apiBase = serverURL
	return c
}

func cfOK(w http.ResponseWriter, result interface{}) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"errors":  []interface{}{},
		"result":  result,
	})
}

func TestNewClient(t *testing.T) {
	c := NewClient("tok")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.apiBase != defaultAPIBase {
		t.Errorf("expected default API base, got %s", c.apiBase)
	}
}

func TestFindZone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("name") != "example.com" {
			t.Errorf("expected name=example.com, got %s", r.URL.Query().Get("name"))
		}
		cfOK(w, []Zone{{ID: "zone123", Name: "example.com"}})
	}))
	defer srv.Close()

	c := newTestCFClient(srv.URL)
	zone, err := c.FindZone("example.com")
	if err != nil {
		t.Fatalf("FindZone error: %v", err)
	}
	if zone.ID != "zone123" {
		t.Errorf("expected zone ID 'zone123', got %q", zone.ID)
	}
}

func TestFindZone_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfOK(w, []Zone{})
	}))
	defer srv.Close()

	c := newTestCFClient(srv.URL)
	_, err := c.FindZone("notexist.com")
	if err == nil {
		t.Error("expected error for missing zone")
	}
}

func TestFindZone_SubdomainStripsToApex(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("name")
		cfOK(w, []Zone{{ID: "zone1", Name: "example.com"}})
	}))
	defer srv.Close()

	c := newTestCFClient(srv.URL)
	c.FindZone("mission.example.com")
	if capturedQuery != "example.com" {
		t.Errorf("expected apex 'example.com' in query, got %q", capturedQuery)
	}
}

func TestFindZone_DeepSubdomainStripsToApex(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("name")
		cfOK(w, []Zone{{ID: "z1", Name: "example.com"}})
	}))
	defer srv.Close()

	c := newTestCFClient(srv.URL)
	c.FindZone("a.b.example.com")
	if capturedQuery != "example.com" {
		t.Errorf("expected 'example.com', got %q", capturedQuery)
	}
}

func TestCreateRecord(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&capturedBody)
		cfOK(w, DNSRecord{ID: "rec1", Type: "A", Name: "mission.example.com", Content: "1.2.3.4"})
	}))
	defer srv.Close()

	c := newTestCFClient(srv.URL)
	rec, err := c.CreateRecord("zone123", "mission.example.com", "1.2.3.4")
	if err != nil {
		t.Fatalf("CreateRecord error: %v", err)
	}
	if rec.ID != "rec1" {
		t.Errorf("expected record ID 'rec1', got %q", rec.ID)
	}
	if capturedBody["type"] != "A" {
		t.Errorf("expected type A, got %v", capturedBody["type"])
	}
	if capturedBody["content"] != "1.2.3.4" {
		t.Errorf("expected content 1.2.3.4, got %v", capturedBody["content"])
	}
}

func TestUpsertRecord_Creates(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == "GET" {
			cfOK(w, []DNSRecord{}) // no existing record
		} else if r.Method == "POST" {
			cfOK(w, DNSRecord{ID: "new1"})
		}
	}))
	defer srv.Close()

	c := newTestCFClient(srv.URL)
	err := c.UpsertRecord("zone123", "mission.example.com", "1.2.3.4")
	if err != nil {
		t.Fatalf("UpsertRecord error: %v", err)
	}
	if len(methods) < 2 {
		t.Errorf("expected at least 2 calls (GET + POST), got %v", methods)
	}
}

func TestUpsertRecord_Updates(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == "GET" {
			cfOK(w, []DNSRecord{{ID: "existing1", Type: "A", Name: "mission.example.com", Content: "old-ip"}})
		} else if r.Method == "PUT" {
			cfOK(w, DNSRecord{ID: "existing1"})
		} else {
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	c := newTestCFClient(srv.URL)
	err := c.UpsertRecord("zone123", "mission.example.com", "1.2.3.4")
	if err != nil {
		t.Fatalf("UpsertRecord error: %v", err)
	}
	if len(methods) < 2 {
		t.Errorf("expected at least 2 calls (GET + PUT), got %v", methods)
	}
	// Should have used PUT (update), not POST (create)
	hasPUT := false
	for _, m := range methods {
		if m == "PUT" {
			hasPUT = true
		}
	}
	if !hasPUT {
		t.Errorf("expected PUT call for update, got methods: %v", methods)
	}
}

func TestAPIError_Returned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 7003, "message": "No zone with that name"}},
			"result":  nil,
		})
	}))
	defer srv.Close()

	c := newTestCFClient(srv.URL)
	_, err := c.FindZone("bad.com")
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "7003") {
		t.Errorf("expected error code 7003, got: %v", err)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
