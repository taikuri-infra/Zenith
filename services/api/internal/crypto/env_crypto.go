package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/hkdf"
)

// EnvCrypto provides AES-256-GCM encryption for environment variable values.
// Each user gets a unique derived key via HKDF, so one user's data cannot
// be decrypted with another user's key even if the DB is compromised.
// The master key never touches the database — it lives only in the K8s secret.
//
// Supports key rotation: encrypted values are tagged with a version prefix
// ("enc:v1:", "enc:v2:", etc.) so the correct master key is used for decryption.
type EnvCrypto struct {
	currentKey     []byte
	currentVersion int
	oldKeys        map[int][]byte // version → master key (for decrypting old data)
}

// NewEnvCrypto creates a new EnvCrypto with the given 32-byte master key.
// Encryption always uses the current key (version 1 by default).
func NewEnvCrypto(masterKey []byte) *EnvCrypto {
	return &EnvCrypto{
		currentKey:     masterKey,
		currentVersion: 1,
		oldKeys:        make(map[int][]byte),
	}
}

// AddOldKey registers a previous master key for decrypting values encrypted under that version.
// Call this for each old key when rotating: the old key decrypts existing data,
// while new encryptions always use the current key.
func (c *EnvCrypto) AddOldKey(version int, key []byte) {
	c.oldKeys[version] = key
}

// SetCurrentVersion upgrades the current encryption version.
// New encryptions will use this version number in their prefix.
func (c *EnvCrypto) SetCurrentVersion(version int) {
	c.currentVersion = version
}

// deriveUserKey derives a unique AES-256 key for a specific user using HKDF-SHA256.
// userID is used as the salt so each user gets a completely different key.
func deriveUserKey(masterKey []byte, userID string) []byte {
	r := hkdf.New(sha256.New, masterKey, []byte(userID), []byte("zenith-env-v1"))
	key := make([]byte, 32)
	io.ReadFull(r, key) //nolint:errcheck — hkdf.Read never errors for key sizes ≤ hash output
	return key
}

// Encrypt encrypts a plaintext value for a specific user with AES-256-GCM.
// Returns a versioned prefixed base64 string: "enc:vN:<base64(nonce+ciphertext)>".
func (c *EnvCrypto) Encrypt(userID, plaintext string) (string, error) {
	key := deriveUserKey(c.currentKey, userID)

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
	return fmt.Sprintf("enc:v%d:%s", c.currentVersion, base64.StdEncoding.EncodeToString(sealed)), nil
}

// Decrypt decrypts an encrypted value for a specific user.
// Supports multiple key versions via the "enc:vN:" prefix.
// If value is not prefixed with "enc:" it is returned as-is (backward compat).
func (c *EnvCrypto) Decrypt(userID, value string) (string, error) {
	if !IsEncrypted(value) {
		return value, nil // plaintext or legacy unencrypted value
	}

	// Parse version from prefix: "enc:v1:..." → version=1, data=...
	parts := strings.SplitN(value, ":", 3)
	if len(parts) != 3 || !strings.HasPrefix(parts[1], "v") {
		return "", fmt.Errorf("invalid encrypted format")
	}
	version, err := strconv.Atoi(parts[1][1:])
	if err != nil {
		return "", fmt.Errorf("invalid key version: %w", err)
	}

	// Select the correct master key for this version
	var masterKey []byte
	if version == c.currentVersion {
		masterKey = c.currentKey
	} else if k, ok := c.oldKeys[version]; ok {
		masterKey = k
	} else {
		return "", fmt.Errorf("unknown key version %d", version)
	}

	data, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid encrypted value: %w", err)
	}

	key := deriveUserKey(masterKey, userID)
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

// ReEncrypt decrypts a value with its original key version and re-encrypts with the current key.
// Returns the original value unchanged if it's already at the current version.
func (c *EnvCrypto) ReEncrypt(userID, value string) (string, bool, error) {
	if !IsEncrypted(value) {
		return value, false, nil
	}

	prefix := fmt.Sprintf("enc:v%d:", c.currentVersion)
	if strings.HasPrefix(value, prefix) {
		return value, false, nil // already current version
	}

	plaintext, err := c.Decrypt(userID, value)
	if err != nil {
		return "", false, fmt.Errorf("decrypt for re-encryption: %w", err)
	}
	encrypted, err := c.Encrypt(userID, plaintext)
	if err != nil {
		return "", false, fmt.Errorf("re-encrypt: %w", err)
	}
	return encrypted, true, nil
}

// IsEncrypted reports whether a value was encrypted by EnvCrypto.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, "enc:")
}
