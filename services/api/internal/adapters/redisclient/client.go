package redisclient

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a Redis connection with helper methods for rate limiting and
// token blacklisting. If the connection fails at startup the caller should
// fall back to in-memory implementations.
type Client struct {
	rdb *redis.Client
}

// New connects to Redis and pings to verify. Returns an error if unreachable.
func New(redisURL string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, err
	}
	slog.Info("redis connected", "addr", opts.Addr)
	return &Client{rdb: rdb}, nil
}

// Close shuts down the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// --- Rate Limiter Storage (Fiber-compatible) ---

// RateLimiterStorage implements github.com/gofiber/fiber/v2/middleware/limiter.Storage
// using a sliding-window counter in Redis. Each key is auto-expired.
type RateLimiterStorage struct {
	rdb    *redis.Client
	prefix string
}

// NewRateLimiterStorage creates a Fiber-compatible rate limiter storage backed by Redis.
func (c *Client) NewRateLimiterStorage(prefix string) *RateLimiterStorage {
	return &RateLimiterStorage{rdb: c.rdb, prefix: prefix}
}

// Get returns the stored value for a key. Returns nil if the key doesn't exist.
func (s *RateLimiterStorage) Get(key string) ([]byte, error) {
	ctx := context.Background()
	val, err := s.rdb.Get(ctx, s.prefix+key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

// Set stores a value with an expiration.
func (s *RateLimiterStorage) Set(key string, val []byte, exp time.Duration) error {
	ctx := context.Background()
	return s.rdb.Set(ctx, s.prefix+key, val, exp).Err()
}

// Delete removes a key.
func (s *RateLimiterStorage) Delete(key string) error {
	ctx := context.Background()
	return s.rdb.Del(ctx, s.prefix+key).Err()
}

// Reset clears all keys with this storage's prefix.
func (s *RateLimiterStorage) Reset() error {
	ctx := context.Background()
	iter := s.rdb.Scan(ctx, 0, s.prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		s.rdb.Del(ctx, iter.Val())
	}
	return iter.Err()
}

// Close is a no-op — the underlying Redis connection is managed by the Client.
func (s *RateLimiterStorage) Close() error {
	return nil
}

// --- Token Blacklist ---

// TokenBlacklist stores revoked JWT tokens in Redis with auto-expiry.
// This replaces the in-memory TokenBlacklist when Redis is available.
type TokenBlacklist struct {
	rdb    *redis.Client
	prefix string
}

// NewTokenBlacklist creates a Redis-backed token blacklist.
func (c *Client) NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{rdb: c.rdb, prefix: "zenith:blacklist:"}
}

// Revoke adds a token hash to the blacklist. It auto-expires at the token's expiry time.
func (tb *TokenBlacklist) Revoke(tokenHash string, expiresAt time.Time) {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return // already expired, no need to blacklist
	}
	ctx := context.Background()
	if err := tb.rdb.Set(ctx, tb.prefix+tokenHash, "1", ttl).Err(); err != nil {
		slog.Warn("redis: failed to revoke token", "error", err)
	}
}

// IsRevoked checks if a token hash is in the blacklist.
func (tb *TokenBlacklist) IsRevoked(tokenHash string) bool {
	ctx := context.Background()
	exists, err := tb.rdb.Exists(ctx, tb.prefix+tokenHash).Result()
	if err != nil {
		slog.Warn("redis: failed to check token blacklist", "error", err)
		return false // fail open — the in-memory blacklist is the fallback
	}
	return exists > 0
}
