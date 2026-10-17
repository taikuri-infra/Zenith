package services

import (
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// ErrSystemRootsUnavailable marks a CertPoolWithFile error caused by the host
// system root pool being unavailable (as opposed to a bad CA file).
var ErrSystemRootsUnavailable = errors.New("system root pool unavailable")

// CertPoolWithFile returns the system root pool plus the PEM certificates in
// path (the install's own CA on a self-signed install). An empty path returns
// exactly what x509.SystemCertPool returns; a nil pool means system roots to
// TraefikTLSProber. A missing or unparsable file returns the system pool
// together with the error, so callers can warn and continue. If the system
// pool is unavailable and a file is given, the pool holds the install CA only
// and the error wraps ErrSystemRootsUnavailable.
func CertPoolWithFile(path string) (*x509.CertPool, error) {
	sys, serr := x509.SystemCertPool()
	if path == "" {
		return sys, serr
	}
	pool := sys
	sysDown := serr != nil || pool == nil
	if sysDown {
		pool = x509.NewCertPool()
	}
	pemBytes, err := os.ReadFile(path)
	if err == nil && !pool.AppendCertsFromPEM(pemBytes) {
		err = fmt.Errorf("tls ca file %s: no certificates found", path)
	} else if err != nil {
		err = fmt.Errorf("read tls ca file %s: %w", path, err)
	}
	if err != nil {
		if sysDown {
			return nil, fmt.Errorf("%w: %w", ErrSystemRootsUnavailable, err)
		}
		return sys, err
	}
	if sysDown {
		return pool, fmt.Errorf("%w (trusting install CA only): %v", ErrSystemRootsUnavailable, serr)
	}
	return pool, nil
}
