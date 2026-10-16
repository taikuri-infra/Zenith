// services/api/internal/services/domain_tlsprobe_test.go
package services

import (
	"context"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTraefikTLSProber(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "https://")
	leaf := srv.Certificate() // httptest's cert is valid for 127.0.0.1 and example.com
	name := "example.com"
	if len(leaf.DNSNames) > 0 {
		name = leaf.DNSNames[0]
	}

	trusted := x509.NewCertPool()
	trusted.AddCert(leaf)
	valid, err := (&TraefikTLSProber{Roots: trusted}).Probe(context.Background(), addr, name)
	if err != nil || !valid {
		t.Fatalf("trusted roots: valid=%v err=%v", valid, err)
	}
	valid, err = (&TraefikTLSProber{Roots: x509.NewCertPool()}).Probe(context.Background(), addr, name)
	if err != nil || valid {
		t.Fatalf("untrusted roots must be (false, nil): valid=%v err=%v", valid, err)
	}
	valid, err = (&TraefikTLSProber{Roots: trusted}).Probe(context.Background(), addr, "other.example.org")
	if err != nil || valid {
		t.Fatalf("wrong name must be (false, nil): valid=%v err=%v", valid, err)
	}
	if _, err = (&TraefikTLSProber{}).Probe(context.Background(), "127.0.0.1:1", name); err == nil {
		t.Fatal("closed port must return an error")
	}
}
