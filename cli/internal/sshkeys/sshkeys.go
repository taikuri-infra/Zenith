package sshkeys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// KeyPair holds a generated RSA key pair.
type KeyPair struct {
	PrivateKeyPEM []byte // PEM-encoded PKCS#1 RSA private key
	PublicKeySSH  string // OpenSSH authorized_keys format
}

// Generate creates a new 4096-bit RSA key pair.
func Generate() (*KeyPair, error) {
	return GenerateWithBits(4096)
}

// GenerateWithBits creates an RSA key pair of the given bit size.
func GenerateWithBits(bits int) (*KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("generate RSA key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	pubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("create SSH public key: %w", err)
	}

	return &KeyPair{
		PrivateKeyPEM: privPEM,
		PublicKeySSH:  string(ssh.MarshalAuthorizedKey(pubKey)),
	}, nil
}

// ParsePrivateKey parses a PEM-encoded RSA private key into an ssh.Signer.
func ParsePrivateKey(pemData []byte) (ssh.Signer, error) {
	signer, err := ssh.ParsePrivateKey(pemData)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return signer, nil
}
