package sshkeys

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	kp, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(kp.PrivateKeyPEM) == 0 {
		t.Error("expected non-empty PrivateKeyPEM")
	}
	if len(kp.PublicKeySSH) == 0 {
		t.Error("expected non-empty PublicKeySSH")
	}
	if !strings.HasPrefix(string(kp.PrivateKeyPEM), "-----BEGIN RSA PRIVATE KEY-----") {
		t.Error("PrivateKeyPEM should be PEM-encoded RSA key")
	}
	if !strings.HasPrefix(kp.PublicKeySSH, "ssh-rsa ") {
		t.Error("PublicKeySSH should start with 'ssh-rsa '")
	}
}

func TestGenerateWithBits(t *testing.T) {
	kp, err := GenerateWithBits(2048)
	if err != nil {
		t.Fatalf("GenerateWithBits(2048) error: %v", err)
	}
	if len(kp.PrivateKeyPEM) == 0 {
		t.Error("expected non-empty PrivateKeyPEM")
	}
}

func TestGenerateUnique(t *testing.T) {
	kp1, _ := GenerateWithBits(2048)
	kp2, _ := GenerateWithBits(2048)
	if string(kp1.PrivateKeyPEM) == string(kp2.PrivateKeyPEM) {
		t.Error("two key pairs should not be identical")
	}
	if kp1.PublicKeySSH == kp2.PublicKeySSH {
		t.Error("two public keys should not be identical")
	}
}

func TestParsePrivateKey(t *testing.T) {
	kp, err := GenerateWithBits(2048)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	signer, err := ParsePrivateKey(kp.PrivateKeyPEM)
	if err != nil {
		t.Fatalf("ParsePrivateKey error: %v", err)
	}
	if signer == nil {
		t.Error("expected non-nil signer")
	}
}

func TestParsePrivateKey_Invalid(t *testing.T) {
	_, err := ParsePrivateKey([]byte("not a valid PEM key"))
	if err == nil {
		t.Error("expected error for invalid key")
	}
}
