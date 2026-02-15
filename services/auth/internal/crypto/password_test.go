package crypto

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	if hash == "" {
		t.Fatal("Hash is empty")
	}
	if hash == "mypassword123" {
		t.Error("Hash should not be plaintext")
	}

	if !CheckPassword("mypassword123", hash) {
		t.Error("Password should match hash")
	}
	if CheckPassword("wrongpassword", hash) {
		t.Error("Wrong password should not match hash")
	}
}

func TestGenerateSecret(t *testing.T) {
	secret, err := GenerateSecret(32)
	if err != nil {
		t.Fatalf("Failed to generate secret: %v", err)
	}
	if len(secret) != 32 {
		t.Errorf("Expected secret length 32, got %d", len(secret))
	}

	secret2, _ := GenerateSecret(32)
	if secret == secret2 {
		t.Error("Two generated secrets should be different")
	}
}

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if id == "" {
		t.Error("Generated ID is empty")
	}
	if len(id) != 22 {
		t.Errorf("Expected ID length 22, got %d", len(id))
	}

	id2 := GenerateID()
	if id == id2 {
		t.Error("Two generated IDs should be different")
	}
}
