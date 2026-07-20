package main

import (
	"net"
	"net/http/httptest"
	"testing"
)

func TestIsPublicIP(t *testing.T) {
	public := []string{"1.2.3.4", "203.0.113.9", "8.8.8.8", "2606:4700:4700::1111"}
	private := []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "172.16.0.1", "169.254.0.1", "0.0.0.0", "::1", "224.0.0.1"}
	for _, s := range public {
		if !isPublicIP(net.ParseIP(s)) {
			t.Errorf("%s should be public", s)
		}
	}
	for _, s := range private {
		if isPublicIP(net.ParseIP(s)) {
			t.Errorf("%s should be rejected as non-public", s)
		}
	}
}

func TestValidHostname(t *testing.T) {
	ok := []string{"swift-otter-3e0b.apps.freezenith.com", "a-b-c.apps.freezenith.com"}
	bad := []string{"a b.apps.freezenith.com", "x/../y", "name&injected=1", "up?per.com", "UPPER.com", "semi;colon"}
	for _, h := range ok {
		if !validHostname(h) {
			t.Errorf("%q should be valid", h)
		}
	}
	for _, h := range bad {
		if validHostname(h) {
			t.Errorf("%q should be rejected (injection-safe)", h)
		}
	}
}

func TestAuthorized_FailsClosed(t *testing.T) {
	// No token configured => everything is rejected.
	open := &server{installToken: ""}
	req := httptest.NewRequest("POST", "/register", nil)
	req.Header.Set("Authorization", "Bearer anything")
	if open.authorized(req) {
		t.Error("must fail closed when INSTALL_TOKEN is unset")
	}

	s := &server{installToken: "secret"}
	good := httptest.NewRequest("POST", "/register", nil)
	good.Header.Set("Authorization", "Bearer secret")
	if !s.authorized(good) {
		t.Error("correct token should be authorized")
	}
	bad := httptest.NewRequest("POST", "/register", nil)
	bad.Header.Set("Authorization", "Bearer wrong")
	if s.authorized(bad) {
		t.Error("wrong token must be rejected")
	}
}
