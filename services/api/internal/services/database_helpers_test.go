package services

import (
	"testing"
)

// --- parseDSNHost additional tests (not in database_test.go) ---

func TestParseDSNHost_NoPort(t *testing.T) {
	host := parseDSNHost("postgresql://admin:secret@db.zenith.svc/zenith")
	if host != "db.zenith.svc" {
		t.Errorf("Expected 'db.zenith.svc', got '%s'", host)
	}
}

func TestParseDSNHost_IPv4(t *testing.T) {
	host := parseDSNHost("postgresql://admin:pass@192.168.1.100:5432/db")
	if host != "192.168.1.100" {
		t.Errorf("Expected '192.168.1.100', got '%s'", host)
	}
}

// --- generatePassword additional tests ---

func TestGeneratePassword_ZeroLength(t *testing.T) {
	pass, err := generatePassword(0)
	if err != nil {
		t.Fatalf("generatePassword(0) failed: %v", err)
	}
	if len(pass) != 0 {
		t.Errorf("Expected empty password for 0 length, got '%s'", pass)
	}
}

func TestGeneratePassword_SmallLength(t *testing.T) {
	pass, err := generatePassword(1)
	if err != nil {
		t.Fatalf("generatePassword(1) failed: %v", err)
	}
	if len(pass) != 2 { // 1 byte = 2 hex chars
		t.Errorf("Expected 2-char password, got %d chars", len(pass))
	}
}

// --- sanitizeIdentifier additional tests ---

func TestSanitizeIdentifier_MixedCase(t *testing.T) {
	result := sanitizeIdentifier("MyDb_123_Test")
	if result != "mydb_123_test" {
		t.Errorf("Expected 'mydb_123_test', got '%s'", result)
	}
}

func TestSanitizeIdentifier_OnlyUnderscores(t *testing.T) {
	result := sanitizeIdentifier("___")
	if result != "___" {
		t.Errorf("Expected '___', got '%s'", result)
	}
}

func TestSanitizeIdentifier_UnicodeChars(t *testing.T) {
	result := sanitizeIdentifier("café_db")
	if result != "caf_db" {
		t.Errorf("Expected 'caf_db', got '%s'", result)
	}
}
