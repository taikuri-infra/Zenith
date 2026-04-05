package middleware

import (
	"testing"
	"time"
)

func TestTokenBlacklistRevokeAndIsRevoked(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	token := "test-token-abc"
	expiresAt := time.Now().Add(1 * time.Hour)

	bl.Revoke(token, expiresAt)

	if !bl.IsRevoked(token) {
		t.Error("Expected token to be revoked")
	}
}

func TestTokenBlacklistNotRevoked(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	if bl.IsRevoked("non-existent-token") {
		t.Error("Expected non-existent token to not be revoked")
	}
}

func TestTokenBlacklistExpiredToken(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	token := "expired-token"
	expiresAt := time.Now().Add(-1 * time.Hour) // Already expired

	bl.Revoke(token, expiresAt)

	if bl.IsRevoked(token) {
		t.Error("Expected expired token to not be considered revoked")
	}
}

func TestTokenBlacklistMultipleTokens(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	tokens := []string{"token-a", "token-b", "token-c"}
	expiresAt := time.Now().Add(1 * time.Hour)

	for _, tok := range tokens {
		bl.Revoke(tok, expiresAt)
	}

	for _, tok := range tokens {
		if !bl.IsRevoked(tok) {
			t.Errorf("Expected token %q to be revoked", tok)
		}
	}

	if bl.IsRevoked("token-d") {
		t.Error("Expected unrevoked token to not be revoked")
	}
}

func TestTokenBlacklistStop(t *testing.T) {
	bl := NewTokenBlacklist()
	bl.Stop() // Should not panic or block
}

func TestTokenBlacklistHashDeterminism(t *testing.T) {
	// Calling hashToken with the same input should return the same hash
	h1 := hashToken("same-token")
	h2 := hashToken("same-token")
	if h1 != h2 {
		t.Errorf("Expected same hash for same input: %s != %s", h1, h2)
	}

	// Different input should produce different hash
	h3 := hashToken("different-token")
	if h1 == h3 {
		t.Error("Expected different hashes for different inputs")
	}
}

func TestTokenBlacklistRevokeOverwrite(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	token := "overwrite-token"

	// First revoke with short expiry (already expired)
	bl.Revoke(token, time.Now().Add(-1*time.Second))
	if bl.IsRevoked(token) {
		t.Error("Expected expired revocation to not be considered revoked")
	}

	// Re-revoke with long expiry
	bl.Revoke(token, time.Now().Add(1*time.Hour))
	if !bl.IsRevoked(token) {
		t.Error("Expected re-revoked token with future expiry to be revoked")
	}
}

// mockRedisBlacklister implements RedisBlacklister for testing.
type mockRedisBlacklister struct {
	tokens map[string]time.Time
}

func newMockRedisBlacklister() *mockRedisBlacklister {
	return &mockRedisBlacklister{tokens: make(map[string]time.Time)}
}

func (m *mockRedisBlacklister) Revoke(tokenHash string, expiresAt time.Time) {
	m.tokens[tokenHash] = expiresAt
}

func (m *mockRedisBlacklister) IsRevoked(tokenHash string) bool {
	exp, exists := m.tokens[tokenHash]
	if !exists {
		return false
	}
	return time.Now().Before(exp)
}

func TestTokenBlacklistWithRedisBackend(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	redis := newMockRedisBlacklister()
	bl.SetRedisBackend(redis)

	token := "redis-token"
	expiresAt := time.Now().Add(1 * time.Hour)

	bl.Revoke(token, expiresAt)

	// Should be found via Redis
	if !bl.IsRevoked(token) {
		t.Error("Expected token to be revoked (via Redis backend)")
	}

	// Verify write-through to Redis
	h := hashToken(token)
	if _, exists := redis.tokens[h]; !exists {
		t.Error("Expected token hash to be written to Redis backend")
	}
}

func TestTokenBlacklistRedisOnlyRevocation(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	redis := newMockRedisBlacklister()
	bl.SetRedisBackend(redis)

	// Simulate a token revoked on another replica (only in Redis, not in local memory)
	token := "remote-revoked-token"
	h := hashToken(token)
	redis.tokens[h] = time.Now().Add(1 * time.Hour)

	// Should still be found via Redis check
	if !bl.IsRevoked(token) {
		t.Error("Expected token revoked only in Redis to be detected")
	}
}

func TestTokenBlacklistRedisExpiredFallbackToMemory(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	redis := newMockRedisBlacklister()
	bl.SetRedisBackend(redis)

	token := "memory-only-token"
	expiresAt := time.Now().Add(1 * time.Hour)

	// Revoke token (goes to both memory and Redis)
	bl.Revoke(token, expiresAt)

	// Remove from Redis to simulate Redis expiration
	h := hashToken(token)
	delete(redis.tokens, h)

	// Should still be found in local memory
	if !bl.IsRevoked(token) {
		t.Error("Expected token still in memory to be revoked even if Redis lost it")
	}
}
