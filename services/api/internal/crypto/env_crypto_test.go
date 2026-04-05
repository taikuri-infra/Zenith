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

// --- Key rotation tests ---

func TestAddOldKeyAndDecryptOldVersion(t *testing.T) {
	// Encrypt with key1 as current (v1)
	key1 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i + 1)
	}
	c := NewEnvCrypto(key1)

	encrypted, err := c.Encrypt("user-1", "rotate-me")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if !strings.HasPrefix(encrypted, "enc:v1:") {
		t.Fatalf("Expected enc:v1: prefix, got %s", encrypted[:min(20, len(encrypted))])
	}

	// Rotate: key2 becomes current (v2), key1 becomes old (v1)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 100)
	}
	c2 := NewEnvCrypto(key2)
	c2.SetCurrentVersion(2)
	c2.AddOldKey(1, key1)

	// Decrypt the v1-encrypted value with the rotated crypto
	decrypted, err := c2.Decrypt("user-1", encrypted)
	if err != nil {
		t.Fatalf("Decrypt with old key failed: %v", err)
	}
	if decrypted != "rotate-me" {
		t.Errorf("Expected 'rotate-me', got '%s'", decrypted)
	}
}

func TestReEncrypt(t *testing.T) {
	// Encrypt with key1 (v1)
	key1 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i + 1)
	}
	c1 := NewEnvCrypto(key1)
	encrypted, err := c1.Encrypt("user-1", "secret-data")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Rotate: key2 is current (v2), key1 is old (v1)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 100)
	}
	c2 := NewEnvCrypto(key2)
	c2.SetCurrentVersion(2)
	c2.AddOldKey(1, key1)

	// ReEncrypt should change the ciphertext
	reEncrypted, changed, err := c2.ReEncrypt("user-1", encrypted)
	if err != nil {
		t.Fatalf("ReEncrypt failed: %v", err)
	}
	if !changed {
		t.Error("Expected changed=true after re-encryption from v1 to v2")
	}
	if reEncrypted == encrypted {
		t.Error("Expected re-encrypted value to differ from original")
	}
	if !strings.HasPrefix(reEncrypted, "enc:v2:") {
		t.Errorf("Expected enc:v2: prefix after re-encryption, got %s", reEncrypted[:min(20, len(reEncrypted))])
	}

	// The re-encrypted value should decrypt correctly with key2 as current
	decrypted, err := c2.Decrypt("user-1", reEncrypted)
	if err != nil {
		t.Fatalf("Decrypt of re-encrypted value failed: %v", err)
	}
	if decrypted != "secret-data" {
		t.Errorf("Expected 'secret-data', got '%s'", decrypted)
	}
}

func TestReEncryptNoChange(t *testing.T) {
	c := newTestCrypto()

	// Encrypt with the current key (v1)
	encrypted, err := c.Encrypt("user-1", "already-current")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// ReEncrypt should return changed=false since it's already at current version
	result, changed, err := c.ReEncrypt("user-1", encrypted)
	if err != nil {
		t.Fatalf("ReEncrypt failed: %v", err)
	}
	if changed {
		t.Error("Expected changed=false for value already at current version")
	}
	if result != encrypted {
		t.Error("Expected value to remain unchanged when already at current version")
	}
}

func TestSetCurrentVersion(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	c := NewEnvCrypto(key)
	c.SetCurrentVersion(5)

	encrypted, err := c.Encrypt("user-1", "versioned")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if !strings.HasPrefix(encrypted, "enc:v5:") {
		t.Errorf("Expected enc:v5: prefix, got %s", encrypted[:min(20, len(encrypted))])
	}

	// Decrypt should work since current version is 5
	decrypted, err := c.Decrypt("user-1", encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != "versioned" {
		t.Errorf("Expected 'versioned', got '%s'", decrypted)
	}
}

func TestDecryptUnknownVersion(t *testing.T) {
	// Encrypt with key1 (v1)
	key1 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i + 1)
	}
	c1 := NewEnvCrypto(key1)
	encrypted, err := c1.Encrypt("user-1", "will-be-lost")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Create a new crypto with only v3 key — v1 is unknown
	key3 := make([]byte, 32)
	for i := range key3 {
		key3[i] = byte(i + 200)
	}
	c3 := NewEnvCrypto(key3)
	c3.SetCurrentVersion(3)
	// Deliberately NOT adding v1 as old key

	_, err = c3.Decrypt("user-1", encrypted)
	if err == nil {
		t.Error("Expected decryption to fail for unknown key version")
	}
	if !strings.Contains(err.Error(), "unknown key version") {
		t.Errorf("Expected 'unknown key version' error, got: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
