package services

import (
	"testing"
)

// --- parseDSNHost tests ---

func TestParseDSNHost_ValidDSN(t *testing.T) {
	host := parseDSNHost("postgres://user:pass@my-pg-host.svc.local:5432/mydb?sslmode=disable")
	if host != "my-pg-host.svc.local" {
		t.Errorf("Expected host 'my-pg-host.svc.local', got '%s'", host)
	}
}

func TestParseDSNHost_SimpleHost(t *testing.T) {
	host := parseDSNHost("postgres://admin:secret@localhost:5432/db")
	if host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", host)
	}
}

func TestParseDSNHost_EmptyDSN(t *testing.T) {
	host := parseDSNHost("")
	if host != "" {
		t.Errorf("Expected empty host for empty DSN, got '%s'", host)
	}
}

func TestParseDSNHost_InvalidDSN(t *testing.T) {
	// url.Parse is lenient, so even invalid URLs may parse without error
	// This test just ensures it doesn't panic
	_ = parseDSNHost("not-a-url")
}

func TestParseDSNHost_WithPort(t *testing.T) {
	host := parseDSNHost("postgres://u:p@cnpg-cluster-rw.zenith-shared.svc.cluster.local:5432/zenith_platform")
	if host != "cnpg-cluster-rw.zenith-shared.svc.cluster.local" {
		t.Errorf("Expected K8s service host, got '%s'", host)
	}
}

// --- sanitizeIdentifier tests ---

func TestSanitizeIdentifier_LowercasesAndFilters(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"MyDB", "mydb"},
		{"user-name", "username"},
		{"db_user_123", "db_user_123"},
		{"Hello World!", "helloworld"},
		{"UPPER_CASE", "upper_case"},
		{"special@#$chars", "specialchars"},
		{"123_start", "123_start"},
		{"", ""},
	}

	for _, tc := range cases {
		got := sanitizeIdentifier(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeIdentifier(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// --- generatePassword tests ---

func TestGeneratePassword_Length(t *testing.T) {
	pwd, err := generatePassword(16)
	if err != nil {
		t.Fatalf("generatePassword failed: %v", err)
	}
	// 16 bytes = 32 hex chars
	if len(pwd) != 32 {
		t.Errorf("Expected 32 char hex string, got %d chars: %s", len(pwd), pwd)
	}
}

func TestGeneratePassword_Uniqueness(t *testing.T) {
	pwd1, _ := generatePassword(16)
	pwd2, _ := generatePassword(16)
	if pwd1 == pwd2 {
		t.Error("Two generated passwords should not be identical")
	}
}

func TestGeneratePassword_DifferentLengths(t *testing.T) {
	for _, n := range []int{8, 16, 32} {
		pwd, err := generatePassword(n)
		if err != nil {
			t.Fatalf("generatePassword(%d) failed: %v", n, err)
		}
		if len(pwd) != n*2 {
			t.Errorf("generatePassword(%d): expected %d chars, got %d", n, n*2, len(pwd))
		}
	}
}
