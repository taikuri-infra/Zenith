package services

import "regexp"

type piiRule struct {
	pattern     *regexp.Regexp
	replacement string
}

// Order matters: connection strings must be replaced before the email pattern,
// since "user:pass@host.com" looks like an email address.
var piiPatterns = []piiRule{
	// Connection strings (before email)
	{regexp.MustCompile(`postgresql://\w+:[^@]+@`), "postgresql://[USER]:[REDACTED]@"},
	{regexp.MustCompile(`redis://:[^@]+@`), "redis://:[REDACTED]@"},
	// Tokens and keys (before general patterns)
	{regexp.MustCompile(`Bearer\s+eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`), "Bearer [TOKEN]"},
	{regexp.MustCompile(`(sk_live_|sk_test_|pk_live_|pk_test_|api_key=|apikey=)[A-Za-z0-9_-]+`), "[API_KEY]"},
	// Email (word chars only — avoids matching :password@ in URLs)
	{regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`), "[EMAIL]"},
	// IP addresses
	{regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), "[IP]"},
	// UUIDs
	{regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), "[UUID]"},
}

// ScrubPII replaces known PII patterns in log text with safe placeholders.
func ScrubPII(log string) string {
	result := log
	for _, rule := range piiPatterns {
		result = rule.pattern.ReplaceAllString(result, rule.replacement)
	}
	return result
}
