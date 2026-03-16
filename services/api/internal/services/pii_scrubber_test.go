package services

import (
	"testing"
)

func TestScrubPII_Email(t *testing.T) {
	input := "user john@example.com tried to login"
	got := ScrubPII(input)
	expected := "user [EMAIL] tried to login"
	if got != expected {
		t.Errorf("email: got %q, want %q", got, expected)
	}
}

func TestScrubPII_IP(t *testing.T) {
	input := "connection from 192.168.1.100 refused"
	got := ScrubPII(input)
	expected := "connection from [IP] refused"
	if got != expected {
		t.Errorf("ip: got %q, want %q", got, expected)
	}
}

func TestScrubPII_BearerToken(t *testing.T) {
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0.abc123_def-456"
	got := ScrubPII(input)
	expected := "Authorization: Bearer [TOKEN]"
	if got != expected {
		t.Errorf("bearer: got %q, want %q", got, expected)
	}
}

func TestScrubPII_APIKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"key: sk_live_abc123", "key: [API_KEY]"},
		{"key: sk_test_xyz789", "key: [API_KEY]"},
		{"url?api_key=secret123", "url?[API_KEY]"},
	}
	for _, tt := range tests {
		got := ScrubPII(tt.input)
		if got != tt.expected {
			t.Errorf("api_key: got %q, want %q", got, tt.expected)
		}
	}
}

func TestScrubPII_PostgresURL(t *testing.T) {
	input := "dsn: postgresql://admin:s3cretPass@db.example.com:5432/mydb"
	got := ScrubPII(input)
	expected := "dsn: postgresql://[USER]:[REDACTED]@db.example.com:5432/mydb"
	if got != expected {
		t.Errorf("postgres: got %q, want %q", got, expected)
	}
}

func TestScrubPII_RedisURL(t *testing.T) {
	input := "redis url: redis://:password123@redis.local:6379"
	got := ScrubPII(input)
	expected := "redis url: redis://:[REDACTED]@redis.local:6379"
	if got != expected {
		t.Errorf("redis: got %q, want %q", got, expected)
	}
}

func TestScrubPII_UUID(t *testing.T) {
	input := "user 550e8400-e29b-41d4-a716-446655440000 deleted"
	got := ScrubPII(input)
	expected := "user [UUID] deleted"
	if got != expected {
		t.Errorf("uuid: got %q, want %q", got, expected)
	}
}

func TestScrubPII_NoMatch(t *testing.T) {
	input := "normal log line with no PII data 2024-01-15"
	got := ScrubPII(input)
	if got != input {
		t.Errorf("no-match: got %q, want %q", got, input)
	}
}

func TestScrubPII_Multiple(t *testing.T) {
	input := "user test@example.com from 10.0.0.1 with key sk_live_abc123"
	got := ScrubPII(input)
	expected := "user [EMAIL] from [IP] with key [API_KEY]"
	if got != expected {
		t.Errorf("multiple: got %q, want %q", got, expected)
	}
}
