package crypto

import (
	"strings"
	"testing"
)

func newTestCrypto() *EnvCrypto {
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i + 1)
	}
	return NewEnvCrypto(masterKey)
}

func TestEncryptDecrypt(t *testing.T) {
	c := newTestCrypto()

	encrypted, err := c.Encrypt("user-1", "my-secret-value")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if !strings.HasPrefix(encrypted, "enc:v1:") {
		t.Errorf("Expected enc:v1: prefix, got %s", encrypted[:min(20, len(encrypted))])
	}

	decrypted, err := c.Decrypt("user-1", encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != "my-secret-value" {
		t.Errorf("Expected 'my-secret-value', got '%s'", decrypted)
	}
}

func TestEncryptProducesUniqueOutput(t *testing.T) {
	c := newTestCrypto()

	enc1, _ := c.Encrypt("user-1", "same-value")
	enc2, _ := c.Encrypt("user-1", "same-value")

	// Different nonces → different ciphertext every time
	if enc1 == enc2 {
		t.Error("Expected unique ciphertext for each encryption (random nonce)")
	}
}

func TestDecryptWrongUserFails(t *testing.T) {
	c := newTestCrypto()

	encrypted, _ := c.Encrypt("user-1", "secret")

	// Decrypting with wrong user ID must fail — different derived key
	_, err := c.Decrypt("user-2", encrypted)
	if err == nil {
		t.Error("Expected decryption to fail with wrong user ID")
	}
}

func TestDecryptPlaintextPassthrough(t *testing.T) {
	c := newTestCrypto()

	// Non-encrypted values must pass through unchanged (backward compat)
	plain, err := c.Decrypt("user-1", "not-encrypted")
	if err != nil {
		t.Fatalf("Expected no error for plaintext passthrough, got %v", err)
	}
	if plain != "not-encrypted" {
		t.Errorf("Expected 'not-encrypted', got '%s'", plain)
	}
}

func TestDecryptEmptyValue(t *testing.T) {
	c := newTestCrypto()

	result, err := c.Decrypt("user-1", "")
	if err != nil {
		t.Fatalf("Expected no error for empty value, got %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestEncryptDecryptEmptyString(t *testing.T) {
	c := newTestCrypto()

	encrypted, err := c.Encrypt("user-1", "")
	if err != nil {
		t.Fatalf("Encrypt failed for empty string: %v", err)
	}

	decrypted, err := c.Decrypt("user-1", encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed for empty string: %v", err)
	}
	if decrypted != "" {
		t.Errorf("Expected empty string, got '%s'", decrypted)
	}
}

func TestPerUserIsolation(t *testing.T) {
	c := newTestCrypto()

	// Each user gets a different derived key
	enc1, _ := c.Encrypt("user-alice", "my-db-password")
	enc2, _ := c.Encrypt("user-bob", "my-db-password")

	// Same plaintext → different ciphertexts (different keys)
	if enc1 == enc2 {
		t.Error("Expected different ciphertexts for different users")
	}

	// Each can only decrypt their own
	dec1, err := c.Decrypt("user-alice", enc1)
	if err != nil || dec1 != "my-db-password" {
		t.Errorf("Alice failed to decrypt her own value: %v", err)
	}

	dec2, err := c.Decrypt("user-bob", enc2)
	if err != nil || dec2 != "my-db-password" {
		t.Errorf("Bob failed to decrypt his own value: %v", err)
	}

	// Cross-user decryption must fail
	_, err = c.Decrypt("user-bob", enc1)
	if err == nil {
		t.Error("Bob should not be able to decrypt Alice's secret")
	}
	_, err = c.Decrypt("user-alice", enc2)
	if err == nil {
		t.Error("Alice should not be able to decrypt Bob's secret")
	}
}

func TestIsEncrypted(t *testing.T) {
	c := newTestCrypto()

	enc, _ := c.Encrypt("user-1", "value")
	if !IsEncrypted(enc) {
		t.Error("Expected IsEncrypted to return true for encrypted value")
	}
	if IsEncrypted("plain-value") {
		t.Error("Expected IsEncrypted to return false for plain value")
	}
	if IsEncrypted("") {
		t.Error("Expected IsEncrypted to return false for empty string")
	}
}

func TestDifferentMasterKeysProduceDifferentResults(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = 0xFF
	}

	c1 := NewEnvCrypto(key1)
	c2 := NewEnvCrypto(key2)

	enc, _ := c1.Encrypt("user-1", "secret")

	// Different master key cannot decrypt
	_, err := c2.Decrypt("user-1", enc)
	if err == nil {
		t.Error("Expected decryption to fail with different master key")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
