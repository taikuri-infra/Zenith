package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
)

const encPrefix = "enc:v1:"

// EnvCrypto provides AES-256-GCM encryption for environment variable values.
// Each user gets a unique derived key via HKDF, so one user's data cannot
// be decrypted with another user's key even if the DB is compromised.
// The master key never touches the database — it lives only in the K8s secret.
type EnvCrypto struct {
	masterKey []byte
}

// NewEnvCrypto creates a new EnvCrypto with the given 32-byte master key.
func NewEnvCrypto(masterKey []byte) *EnvCrypto {
	return &EnvCrypto{masterKey: masterKey}
}

// deriveUserKey derives a unique AES-256 key for a specific user using HKDF-SHA256.
// userID is used as the salt so each user gets a completely different key.
func (c *EnvCrypto) deriveUserKey(userID string) []byte {
	r := hkdf.New(sha256.New, c.masterKey, []byte(userID), []byte("zenith-env-v1"))
	key := make([]byte, 32)
	io.ReadFull(r, key) //nolint:errcheck — hkdf.Read never errors for key sizes ≤ hash output
	return key
}

// Encrypt encrypts a plaintext value for a specific user with AES-256-GCM.
// Returns a prefixed base64 string: "enc:v1:<base64(nonce+ciphertext)>".
func (c *EnvCrypto) Encrypt(userID, plaintext string) (string, error) {
	key := c.deriveUserKey(userID)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Seal appends ciphertext+tag after nonce
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts an encrypted value for a specific user.
// If value is not prefixed with "enc:v1:" it is returned as-is (backward compat).
func (c *EnvCrypto) Decrypt(userID, value string) (string, error) {
	if !strings.HasPrefix(value, encPrefix) {
		return value, nil // plaintext or legacy unencrypted value
	}

	data, err := base64.StdEncoding.DecodeString(value[len(encPrefix):])
	if err != nil {
		return "", fmt.Errorf("invalid encrypted value: %w", err)
	}

	key := c.deriveUserKey(userID)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: wrong key or corrupted data")
	}

	return string(plaintext), nil
}

// IsEncrypted reports whether a value was encrypted by EnvCrypto.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encPrefix)
}
