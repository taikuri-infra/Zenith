package redisclient

import (
	"testing"
	"time"
)

// TestNewRedisClient_InvalidURL verifies that a malformed URL causes an error
// at construction time — before any network dial is attempted.
func TestNewRedisClient_InvalidURL(t *testing.T) {
	_, err := New("not-a-valid-redis-url://???")
	if err == nil {
		t.Fatal("expected error for invalid Redis URL, got nil")
	}
}

// TestNewRedisClient_UnreachableAddr verifies that a valid URL pointing to an
// unreachable address returns an error from the ping step.
func TestNewRedisClient_UnreachableAddr(t *testing.T) {
	_, err := New("redis://127.0.0.1:19999") // port unlikely to be in use
	if err == nil {
		t.Fatal("expected connection error for unreachable Redis address, got nil")
	}
}

// TestRateLimiterStorage_NilSafe verifies that calling methods on a
// RateLimiterStorage constructed from a nil-rdb does not panic.
// In practice the nil check is a safety guard for unit tests that skip Redis.
func TestRateLimiterStorage_GetSetDelete_Integration(t *testing.T) {
	// Skip if we cannot reach a real Redis — this is an integration test.
	c, err := New("redis://127.0.0.1:6379")
	if err != nil {
		t.Skipf("no Redis available at 127.0.0.1:6379, skipping integration test: %v", err)
	}
	defer c.Close()

	storage := c.NewRateLimiterStorage("test:")

	// Set a value
	if err := storage.Set("key1", []byte("val1"), 10*time.Second); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get the value back
	val, err := storage.Get("key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(val) != "val1" {
		t.Errorf("expected val1, got %s", string(val))
	}

	// Delete the key
	if err := storage.Delete("key1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// After deletion, Get should return nil
	val, err = storage.Get("key1")
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil after delete, got %s", string(val))
	}
}

// TestRateLimiterStorage_Reset_Integration verifies Reset clears all prefixed keys.
func TestRateLimiterStorage_Reset_Integration(t *testing.T) {
	c, err := New("redis://127.0.0.1:6379")
	if err != nil {
		t.Skipf("no Redis available, skipping integration test: %v", err)
	}
	defer c.Close()

	storage := c.NewRateLimiterStorage("testreset:")

	storage.Set("a", []byte("1"), 60*time.Second)
	storage.Set("b", []byte("2"), 60*time.Second)

	if err := storage.Reset(); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	val, _ := storage.Get("a")
	if val != nil {
		t.Error("expected key 'a' to be gone after Reset")
	}
}

// TestRateLimiterStorage_Close_NoOp verifies that Close on the storage is safe.
func TestRateLimiterStorage_Close_NoOp(t *testing.T) {
	c, err := New("redis://127.0.0.1:6379")
	if err != nil {
		t.Skipf("no Redis available, skipping integration test: %v", err)
	}
	defer c.Close()

	storage := c.NewRateLimiterStorage("test:")
	if err := storage.Close(); err != nil {
		t.Errorf("Close should be a no-op, got error: %v", err)
	}
}

// TestTokenBlacklist_RevokeAndCheck_Integration verifies the full revoke/check cycle.
func TestTokenBlacklist_RevokeAndCheck_Integration(t *testing.T) {
	c, err := New("redis://127.0.0.1:6379")
	if err != nil {
		t.Skipf("no Redis available, skipping integration test: %v", err)
	}
	defer c.Close()

	bl := c.NewTokenBlacklist()

	const hash = "testtoken_abc123"

	// Token should not be revoked initially.
	if bl.IsRevoked(hash) {
		t.Fatal("expected token to not be revoked before Revoke call")
	}

	// Revoke with a future expiry.
	bl.Revoke(hash, time.Now().Add(60*time.Second))

	if !bl.IsRevoked(hash) {
		t.Error("expected token to be revoked after Revoke call")
	}
}

// TestTokenBlacklist_RevokeAlreadyExpired_NoOp verifies that revoking an already-expired
// token is a safe no-op (the token's TTL would be zero or negative).
func TestTokenBlacklist_RevokeAlreadyExpired_NoOp(t *testing.T) {
	c, err := New("redis://127.0.0.1:6379")
	if err != nil {
		t.Skipf("no Redis available, skipping integration test: %v", err)
	}
	defer c.Close()

	bl := c.NewTokenBlacklist()

	const hash = "expired_token_xyz"
	// Should not write to Redis (TTL <= 0), should not panic.
	bl.Revoke(hash, time.Now().Add(-1*time.Hour))

	if bl.IsRevoked(hash) {
		t.Error("already-expired token should not appear as revoked")
	}
}
