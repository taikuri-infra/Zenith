package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/auth/internal/models"
)

// Store defines the interface for auth data storage
type Store interface {
	// Realms
	CreateRealm(realm *models.Realm) error
	GetRealm(id string) (*models.Realm, error)
	ListRealms() ([]*models.Realm, error)
	UpdateRealm(realm *models.Realm) error
	DeleteRealm(id string) error

	// Users
	CreateUser(user *models.User) error
	GetUser(realmID, id string) (*models.User, error)
	GetUserByEmail(realmID, email string) (*models.User, error)
	ListUsers(realmID string) ([]*models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(realmID, id string) error

	// Clients
	CreateClient(client *models.Client) error
	GetClient(realmID, id string) (*models.Client, error)
	ListClients(realmID string) ([]*models.Client, error)
	DeleteClient(realmID, id string) error

	// Sessions
	CreateSession(session *models.Session) error
	GetSession(id string) (*models.Session, error)
	DeleteSession(id string) error
	DeleteUserSessions(realmID, userID string) error
}

// MemoryStore implements Store with in-memory maps
type MemoryStore struct {
	mu       sync.RWMutex
	realms   map[string]*models.Realm
	users    map[string]*models.User   // key: realmID/userID
	clients  map[string]*models.Client // key: realmID/clientID
	sessions map[string]*models.Session
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		realms:   make(map[string]*models.Realm),
		users:    make(map[string]*models.User),
		clients:  make(map[string]*models.Client),
		sessions: make(map[string]*models.Session),
	}
}

// Realm operations

func (s *MemoryStore) CreateRealm(realm *models.Realm) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.realms[realm.ID]; exists {
		return fmt.Errorf("realm %s already exists", realm.ID)
	}
	realm.CreatedAt = time.Now()
	realm.UpdatedAt = time.Now()
	s.realms[realm.ID] = realm
	return nil
}

func (s *MemoryStore) GetRealm(id string) (*models.Realm, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	realm, exists := s.realms[id]
	if !exists {
		return nil, fmt.Errorf("realm %s not found", id)
	}
	return realm, nil
}

func (s *MemoryStore) ListRealms() ([]*models.Realm, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Realm
	for _, r := range s.realms {
		result = append(result, r)
	}
	return result, nil
}

func (s *MemoryStore) UpdateRealm(realm *models.Realm) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.realms[realm.ID]; !exists {
		return fmt.Errorf("realm %s not found", realm.ID)
	}
	realm.UpdatedAt = time.Now()
	s.realms[realm.ID] = realm
	return nil
}

func (s *MemoryStore) DeleteRealm(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.realms[id]; !exists {
		return fmt.Errorf("realm %s not found", id)
	}
	delete(s.realms, id)
	return nil
}

// User operations

func userKey(realmID, userID string) string {
	return realmID + "/" + userID
}

func (s *MemoryStore) CreateUser(user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userKey(user.RealmID, user.ID)
	if _, exists := s.users[key]; exists {
		return fmt.Errorf("user %s already exists", key)
	}

	// Check email uniqueness
	for _, u := range s.users {
		if u.RealmID == user.RealmID && u.Email == user.Email {
			return fmt.Errorf("user with email %s already exists", user.Email)
		}
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	s.users[key] = user
	return nil
}

func (s *MemoryStore) GetUser(realmID, id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userKey(realmID, id)]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (s *MemoryStore) GetUserByEmail(realmID, email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.RealmID == realmID && u.Email == email {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (s *MemoryStore) ListUsers(realmID string) ([]*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.User
	for _, u := range s.users {
		if u.RealmID == realmID {
			result = append(result, u)
		}
	}
	return result, nil
}

func (s *MemoryStore) UpdateUser(user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userKey(user.RealmID, user.ID)
	if _, exists := s.users[key]; !exists {
		return fmt.Errorf("user not found")
	}
	user.UpdatedAt = time.Now()
	s.users[key] = user
	return nil
}

func (s *MemoryStore) DeleteUser(realmID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userKey(realmID, id)
	if _, exists := s.users[key]; !exists {
		return fmt.Errorf("user not found")
	}
	delete(s.users, key)
	return nil
}

// Client operations

func clientKey(realmID, clientID string) string {
	return realmID + "/" + clientID
}

func (s *MemoryStore) CreateClient(client *models.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := clientKey(client.RealmID, client.ID)
	if _, exists := s.clients[key]; exists {
		return fmt.Errorf("client %s already exists", key)
	}
	client.CreatedAt = time.Now()
	s.clients[key] = client
	return nil
}

func (s *MemoryStore) GetClient(realmID, id string) (*models.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[clientKey(realmID, id)]
	if !exists {
		return nil, fmt.Errorf("client not found")
	}
	return client, nil
}

func (s *MemoryStore) ListClients(realmID string) ([]*models.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Client
	for _, c := range s.clients {
		if c.RealmID == realmID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *MemoryStore) DeleteClient(realmID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := clientKey(realmID, id)
	if _, exists := s.clients[key]; !exists {
		return fmt.Errorf("client not found")
	}
	delete(s.clients, key)
	return nil
}

// Session operations

func (s *MemoryStore) CreateSession(session *models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session.CreatedAt = time.Now()
	s.sessions[session.ID] = session
	return nil
}

func (s *MemoryStore) GetSession(id string) (*models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (s *MemoryStore) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	return nil
}

func (s *MemoryStore) DeleteUserSessions(realmID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, session := range s.sessions {
		if session.RealmID == realmID && session.UserID == userID {
			delete(s.sessions, id)
		}
	}
	return nil
}
