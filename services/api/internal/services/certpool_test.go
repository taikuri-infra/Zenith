package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func selfSignedCA(t *testing.T) []byte {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Zenith Test CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestCertPool_EmptyPathIsSystemPool(t *testing.T) {
	pool, err := CertPoolWithFile("")
	if err != nil || pool == nil {
		t.Fatalf("pool=%v err=%v", pool, err)
	}
}

func TestCertPool_AddsTheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	os.WriteFile(path, selfSignedCA(t), 0o644)
	pool, err := CertPoolWithFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sys, _ := x509.SystemCertPool()
	if sys != nil && pool.Equal(sys) {
		t.Fatal("pool must contain the CA in addition to the system roots")
	}
}

func TestCertPool_MissingFileFallsBackToSystem(t *testing.T) {
	pool, err := CertPoolWithFile(filepath.Join(t.TempDir(), "nope.crt"))
	if pool == nil || err == nil {
		t.Fatalf("want system pool and an error, got pool=%v err=%v", pool, err)
	}
}

func TestCertPool_GarbageFileFallsBackToSystem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.crt")
	os.WriteFile(path, []byte("not a pem"), 0o644)
	pool, err := CertPoolWithFile(path)
	if pool == nil || err == nil {
		t.Fatalf("want system pool and an error, got pool=%v err=%v", pool, err)
	}
}

func TestCertPool_EmptyPathMatchesSystemCertPool(t *testing.T) {
	want, wantErr := x509.SystemCertPool()
	got, err := CertPoolWithFile("")
	if (err == nil) != (wantErr == nil) {
		t.Fatalf("err=%v want %v", err, wantErr)
	}
	if (got == nil) != (want == nil) || (got != nil && !got.Equal(want)) {
		t.Fatal("empty path must return exactly the system cert pool")
	}
}
