package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// RedisBlacklister is an optional Redis backend for the token blacklist.
// When set, Revoke writes to both in-memory and Redis; IsRevoked checks Redis first.
type RedisBlacklister interface {
	Revoke(tokenHash string, expiresAt time.Time)
	IsRevoked(tokenHash string) bool
}

// TokenBlacklist stores revoked JWT tokens until they expire.
// It always keeps an in-memory copy for fast local checks and optionally
// delegates to a Redis backend for cross-replica consistency.
type TokenBlacklist struct {
	mu      sync.RWMutex
	tokens  map[string]time.Time // token hash → expires at
	stopCh  chan struct{}
	redis   RedisBlacklister // optional Redis backend
}

// NewTokenBlacklist creates a new in-memory token blacklist with periodic cleanup.
func NewTokenBlacklist() *TokenBlacklist {
	tb := &TokenBlacklist{
		tokens: make(map[string]time.Time),
		stopCh: make(chan struct{}),
	}
	go tb.cleanup()
	return tb
}

// SetRedisBackend configures a Redis backend for cross-replica token revocation.
func (tb *TokenBlacklist) SetRedisBackend(rb RedisBlacklister) {
	tb.redis = rb
}

// Revoke adds a token to the blacklist. It will be automatically removed after expiresAt.
func (tb *TokenBlacklist) Revoke(token string, expiresAt time.Time) {
	h := hashToken(token)
	tb.mu.Lock()
	tb.tokens[h] = expiresAt
	tb.mu.Unlock()
	// Write-through to Redis if available
	if tb.redis != nil {
		tb.redis.Revoke(h, expiresAt)
	}
}

// IsRevoked checks if a token has been revoked.
func (tb *TokenBlacklist) IsRevoked(token string) bool {
	h := hashToken(token)

	// Check Redis first (cross-replica source of truth)
	if tb.redis != nil {
		if tb.redis.IsRevoked(h) {
			return true
		}
	}

	// Fall back to in-memory (always populated for local revocations)
	tb.mu.RLock()
	exp, exists := tb.tokens[h]
	tb.mu.RUnlock()
	if !exists {
		return false
	}
	if time.Now().After(exp) {
		return false
	}
	return true
}

// Stop halts the background cleanup goroutine.
func (tb *TokenBlacklist) Stop() {
	close(tb.stopCh)
}

// cleanup removes expired entries every 5 minutes.
func (tb *TokenBlacklist) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			tb.mu.Lock()
			for k, exp := range tb.tokens {
				if now.After(exp) {
					delete(tb.tokens, k)
				}
			}
			tb.mu.Unlock()
		case <-tb.stopCh:
			return
		}
	}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:]) // full 256-bit hash
}
