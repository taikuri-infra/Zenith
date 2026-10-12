// Package validate holds small input validators shared by handlers and services.
package validate

import (
	"errors"
	"strings"
)

// Hostname reports whether h is a lowercase RFC 1123 hostname with at least
// one dot: labels of 1-63 chars from [a-z0-9-], no leading/trailing hyphen,
// total length at most 253. Anything that passes is safe to splice into a
// Traefik Host(`...`) rule.
func Hostname(h string) bool {
	if len(h) == 0 || len(h) > 253 || !strings.Contains(h, ".") {
		return false
	}
	for _, label := range strings.Split(h, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
	}
	return true
}

// CustomDomain normalizes (trim + lowercase) and validates a user-supplied
// custom domain. reserved is the dashboard domain: it and every subdomain of
// it (where generated app hosts live) are refused. Empty reserved disables
// that check.
func CustomDomain(domain, reserved string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(domain))
	if !Hostname(d) {
		return "", errors.New("invalid domain: use a hostname like app.example.com (letters, digits, hyphens and dots only)")
	}
	r := strings.ToLower(strings.TrimSpace(reserved))
	if r != "" && (d == r || strings.HasSuffix(d, "."+r)) {
		return "", errors.New("invalid domain: " + r + " and its subdomains are reserved for Zenith")
	}
	return d, nil
}
