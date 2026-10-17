package services

import (
	"crypto/x509"
	"fmt"
	"os"
)

// CertPoolWithFile returns the system root pool plus the PEM certificates in
// path (the install's own CA on a self-signed install). An empty path means the
// system pool alone. A missing or unparsable file still returns the system pool,
// together with the error, so callers can warn and continue.
func CertPoolWithFile(path string) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if path == "" {
		return pool, nil
	}
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return pool, fmt.Errorf("read tls ca file %s: %w", path, err)
	}
	if !pool.AppendCertsFromPEM(pemBytes) {
		return pool, fmt.Errorf("tls ca file %s: no certificates found", path)
	}
	return pool, nil
}
