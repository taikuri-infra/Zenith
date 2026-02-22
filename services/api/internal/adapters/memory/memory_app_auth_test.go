package memory

import (
	"context"
	"testing"
)

func TestEnableAuth(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	cfg, err := repo.EnableAuth(ctx, "app-1", 1000)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !cfg.Enabled {
		t.Error("Expected auth to be enabled")
	}
	if cfg.MaxUsers != 1000 {
		t.Errorf("Expected max users 1000, got %d", cfg.MaxUsers)
	}
	if cfg.JWTSecret == "" {
		t.Error("Expected JWT secret to be set")
	}
}

func TestDisableAuth(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1000)
	err := repo.DisableAuth(ctx, "app-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	cfg, _ := repo.GetAuthConfig(ctx, "app-1")
	if cfg.Enabled {
		t.Error("Expected auth to be disabled")
	}
}

func TestDisableAuthNotConfigured(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	err := repo.DisableAuth(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for unconfigured app")
	}
}

func TestCreateAppUser(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1000)

	user, err := repo.CreateAppUser(ctx, "app-1", "test@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if user.ID == "" {
		t.Error("Expected user ID to be set")
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", user.Email)
	}
}

func TestCreateAppUserDuplicateEmail(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1000)
	repo.CreateAppUser(ctx, "app-1", "test@example.com", "password123", "Test")

	_, err := repo.CreateAppUser(ctx, "app-1", "test@example.com", "password456", "Test 2")
	if err == nil {
		t.Error("Expected duplicate email error")
	}
}

func TestCreateAppUserLimitReached(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1) // limit of 1 user

	repo.CreateAppUser(ctx, "app-1", "user1@test.com", "password123", "User 1")

	_, err := repo.CreateAppUser(ctx, "app-1", "user2@test.com", "password123", "User 2")
	if err == nil {
		t.Error("Expected user limit error")
	}
}

func TestCreateAppUserAuthNotEnabled(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	_, err := repo.CreateAppUser(ctx, "app-1", "test@test.com", "password123", "Test")
	if err == nil {
		t.Error("Expected error when auth not enabled")
	}
}

func TestGetAppUserByEmail(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1000)
	created, _ := repo.CreateAppUser(ctx, "app-1", "test@test.com", "password123", "Test")

	user, hash, err := repo.GetAppUserByEmail(ctx, "app-1", "test@test.com")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if user.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, user.ID)
	}
	if hash == "" {
		t.Error("Expected password hash to be returned")
	}
}

func TestGetAppUserByEmailNotFound(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1000)

	_, _, err := repo.GetAppUserByEmail(ctx, "app-1", "nonexistent@test.com")
	if err == nil {
		t.Error("Expected not found error")
	}
}

func TestCountAppUsers(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1000)
	repo.CreateAppUser(ctx, "app-1", "u1@test.com", "pw123456", "U1")
	repo.CreateAppUser(ctx, "app-1", "u2@test.com", "pw123456", "U2")

	count, err := repo.CountAppUsers(ctx, "app-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestListAppUsers(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1000)
	repo.CreateAppUser(ctx, "app-1", "u1@test.com", "pw123456", "U1")
	repo.CreateAppUser(ctx, "app-1", "u2@test.com", "pw123456", "U2")

	users, err := repo.ListAppUsers(ctx, "app-1", 10, 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestListAppUsersWithOffset(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1000)
	repo.CreateAppUser(ctx, "app-1", "u1@test.com", "pw123456", "U1")
	repo.CreateAppUser(ctx, "app-1", "u2@test.com", "pw123456", "U2")

	users, _ := repo.ListAppUsers(ctx, "app-1", 10, 1)
	if len(users) != 1 {
		t.Errorf("Expected 1 user with offset 1, got %d", len(users))
	}
}

func TestDeleteAppUser(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1000)
	user, _ := repo.CreateAppUser(ctx, "app-1", "test@test.com", "pw123456", "Test")

	err := repo.DeleteAppUser(ctx, "app-1", user.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	count, _ := repo.CountAppUsers(ctx, "app-1")
	if count != 0 {
		t.Errorf("Expected 0 users after delete, got %d", count)
	}

	// Email should be freed for reuse
	_, err = repo.CreateAppUser(ctx, "app-1", "test@test.com", "pw123456", "Test 2")
	if err != nil {
		t.Errorf("Expected email to be reusable after delete, got %v", err)
	}
}

func TestDeleteAppUserNotFound(t *testing.T) {
	repo := NewMemoryAppAuthRepository()
	ctx := context.Background()

	repo.EnableAuth(ctx, "app-1", 1000)

	err := repo.DeleteAppUser(ctx, "app-1", "nonexistent")
	if err == nil {
		t.Error("Expected not found error")
	}
}
