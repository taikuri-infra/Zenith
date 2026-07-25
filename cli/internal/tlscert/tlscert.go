// Package tlscert generates a small local Certificate Authority and leaf
// certificates for compose installs that cannot use Let's Encrypt — internal or
// offline domains (e.g. zenith.lan) where no public CA will ever issue.
//
// The operator imports the CA cert (zenith-ca.crt) once into their trust store;
// from then on every browser on that machine trusts the domain with no warnings,
// even after the leaf is re-issued. Leaf certs carry a wildcard SAN so deployed
// app subdomains (<slug>.<domain>) are covered by the same one-time import.
package tlscert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// validity is deliberately long (~10y): these certs are trusted manually via a
// one-time CA import, so silent auto-renewal (as with ACME) isn't available and
// a short lifetime would just strand the operator with an expired internal site.
const validity = 10 * 365 * 24 * time.Hour

// notBefore is backdated slightly to tolerate clock skew between the box that
// generates the cert and the clients that validate it.
func certWindow() (time.Time, time.Time) {
	now := time.Now()
	return now.Add(-1 * time.Hour), now.Add(validity)
}

func serial() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

func encodeKey(key *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), nil
}

// GenerateCA returns a PEM-encoded self-signed CA certificate and its private key.
func GenerateCA() (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate CA key: %w", err)
	}
	sn, err := serial()
	if err != nil {
		return nil, nil, err
	}
	notBefore, notAfter := certWindow()
	tmpl := &x509.Certificate{
		SerialNumber:          sn,
		Subject:               pkix.Name{CommonName: "Zenith Local CA", Organization: []string{"Zenith"}},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("create CA cert: %w", err)
	}
	keyPEM, err = encodeKey(key)
	if err != nil {
		return nil, nil, err
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return certPEM, keyPEM, nil
}

// GenerateLeaf signs a leaf certificate for domain (plus the wildcard *.domain)
// with the given CA. The returned certPEM is the leaf followed by the CA, so
// Traefik serves a complete chain.
func GenerateLeaf(caCertPEM, caKeyPEM []byte, domain string) (certPEM, keyPEM []byte, err error) {
	caCert, caKey, err := parseCA(caCertPEM, caKeyPEM)
	if err != nil {
		return nil, nil, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate leaf key: %w", err)
	}
	sn, err := serial()
	if err != nil {
		return nil, nil, err
	}
	notBefore, notAfter := certWindow()
	tmpl := &x509.Certificate{
		SerialNumber: sn,
		Subject:      pkix.Name{CommonName: domain},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{domain, "*." + domain},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("sign leaf cert: %w", err)
	}
	keyPEM, err = encodeKey(key)
	if err != nil {
		return nil, nil, err
	}
	leaf := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	certPEM = append(leaf, caCertPEM...)
	return certPEM, keyPEM, nil
}

func parseCA(caCertPEM, caKeyPEM []byte) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	cb, _ := pem.Decode(caCertPEM)
	if cb == nil {
		return nil, nil, fmt.Errorf("invalid CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CA certificate: %w", err)
	}
	kb, _ := pem.Decode(caKeyPEM)
	if kb == nil {
		return nil, nil, fmt.Errorf("invalid CA key PEM")
	}
	caKey, err := x509.ParseECPrivateKey(kb.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CA key: %w", err)
	}
	return caCert, caKey, nil
}
