package storage

import (
	"testing"

	"github.com/dotechhq/zenith/services/auth/internal/models"
)

func TestRealmCRUD(t *testing.T) {
	store := NewMemoryStore()

	// Create
	realm := &models.Realm{ID: "test-realm", Name: "test-realm", DisplayName: "Test", Enabled: true}
	if err := store.CreateRealm(realm); err != nil {
		t.Fatalf("Failed to create realm: %v", err)
	}

	// Duplicate
	if err := store.CreateRealm(realm); err == nil {
		t.Error("Expected error for duplicate realm")
	}

	// Get
	got, err := store.GetRealm("test-realm")
	if err != nil {
		t.Fatalf("Failed to get realm: %v", err)
	}
	if got.Name != "test-realm" {
		t.Errorf("Expected name 'test-realm', got '%s'", got.Name)
	}

	// List
	realms, _ := store.ListRealms()
	if len(realms) != 1 {
		t.Errorf("Expected 1 realm, got %d", len(realms))
	}

	// Update
	realm.DisplayName = "Updated"
	if err := store.UpdateRealm(realm); err != nil {
		t.Fatalf("Failed to update realm: %v", err)
	}

	// Delete
	if err := store.DeleteRealm("test-realm"); err != nil {
		t.Fatalf("Failed to delete realm: %v", err)
	}

	// Get deleted
	if _, err := store.GetRealm("test-realm"); err == nil {
		t.Error("Expected error for deleted realm")
	}
}

func TestUserCRUD(t *testing.T) {
	store := NewMemoryStore()

	user := &models.User{ID: "user-1", RealmID: "realm-1", Email: "test@example.com", Name: "Test"}
	if err := store.CreateUser(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Duplicate email
	user2 := &models.User{ID: "user-2", RealmID: "realm-1", Email: "test@example.com"}
	if err := store.CreateUser(user2); err == nil {
		t.Error("Expected error for duplicate email")
	}

	// Get
	got, err := store.GetUser("realm-1", "user-1")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if got.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", got.Email)
	}

	// Get by email
	gotByEmail, err := store.GetUserByEmail("realm-1", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to get user by email: %v", err)
	}
	if gotByEmail.ID != "user-1" {
		t.Errorf("Expected ID 'user-1', got '%s'", gotByEmail.ID)
	}

	// List
	users, _ := store.ListUsers("realm-1")
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}

	// Delete
	if err := store.DeleteUser("realm-1", "user-1"); err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}
}

func TestClientCRUD(t *testing.T) {
	store := NewMemoryStore()

	client := &models.Client{ID: "client-1", RealmID: "realm-1", Name: "web", Type: "public"}
	if err := store.CreateClient(client); err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	got, err := store.GetClient("realm-1", "client-1")
	if err != nil {
		t.Fatalf("Failed to get client: %v", err)
	}
	if got.Name != "web" {
		t.Errorf("Expected name 'web', got '%s'", got.Name)
	}

	clients, _ := store.ListClients("realm-1")
	if len(clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(clients))
	}

	if err := store.DeleteClient("realm-1", "client-1"); err != nil {
		t.Fatalf("Failed to delete client: %v", err)
	}
}

func TestSessionCRUD(t *testing.T) {
	store := NewMemoryStore()

	session := &models.Session{ID: "sess-1", UserID: "user-1", RealmID: "realm-1"}
	if err := store.CreateSession(session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	got, err := store.GetSession("sess-1")
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if got.UserID != "user-1" {
		t.Errorf("Expected userID 'user-1', got '%s'", got.UserID)
	}

	if err := store.DeleteSession("sess-1"); err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	store.CreateSession(&models.Session{ID: "s1", UserID: "u1", RealmID: "r1"})
	store.CreateSession(&models.Session{ID: "s2", UserID: "u1", RealmID: "r1"})
	store.DeleteUserSessions("r1", "u1")

	if _, err := store.GetSession("s1"); err == nil {
		t.Error("Expected session s1 to be deleted")
	}
}
