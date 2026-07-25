package tlscert

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"
)

func parseCert(t *testing.T, certPEM []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("no PEM block in cert")
	}
	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	return c
}

func TestGenerateCA(t *testing.T) {
	certPEM, keyPEM, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	if len(keyPEM) == 0 {
		t.Fatal("empty CA key")
	}
	ca := parseCert(t, certPEM)
	if !ca.IsCA {
		t.Error("CA cert is not marked IsCA")
	}
	if ca.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("CA cert missing KeyUsageCertSign")
	}
	if got := ca.NotAfter.Sub(ca.NotBefore); got < 9*365*24*time.Hour {
		t.Errorf("CA validity too short: %v", got)
	}
}

func TestGenerateLeafVerifiesAgainstCA(t *testing.T) {
	caCert, caKey, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	leafPEM, leafKeyPEM, err := GenerateLeaf(caCert, caKey, "zenith.lan")
	if err != nil {
		t.Fatalf("GenerateLeaf: %v", err)
	}
	if len(leafKeyPEM) == 0 {
		t.Fatal("empty leaf key")
	}
	leaf := parseCert(t, leafPEM)

	// The leaf must chain to the CA.
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caCert) {
		t.Fatal("could not add CA to pool")
	}
	if _, err := leaf.Verify(x509.VerifyOptions{
		DNSName: "zenith.lan",
		Roots:   roots,
	}); err != nil {
		t.Fatalf("leaf does not verify for apex: %v", err)
	}

	// The wildcard SAN must cover deployed-app subdomains.
	if _, err := leaf.Verify(x509.VerifyOptions{
		DNSName: "app-42.zenith.lan",
		Roots:   roots,
	}); err != nil {
		t.Fatalf("leaf does not verify for subdomain: %v", err)
	}
}

func TestGenerateLeafSANs(t *testing.T) {
	caCert, caKey, _ := GenerateCA()
	leafPEM, _, err := GenerateLeaf(caCert, caKey, "example.internal")
	if err != nil {
		t.Fatalf("GenerateLeaf: %v", err)
	}
	leaf := parseCert(t, leafPEM)
	want := map[string]bool{"example.internal": false, "*.example.internal": false}
	for _, n := range leaf.DNSNames {
		if _, ok := want[n]; ok {
			want[n] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("leaf missing SAN %q (got %v)", name, leaf.DNSNames)
		}
	}
	if leaf.IsCA {
		t.Error("leaf must not be a CA")
	}
}

func TestGenerateLeafRejectsBadCAKey(t *testing.T) {
	caCert, _, _ := GenerateCA()
	if _, _, err := GenerateLeaf(caCert, []byte("not a key"), "zenith.lan"); err == nil {
		t.Error("expected error with invalid CA key, got nil")
	}
}
