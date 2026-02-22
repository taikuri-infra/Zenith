package memory

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemoryDatabaseRepository is an in-memory implementation of DatabaseRepository.
type MemoryDatabaseRepository struct {
	mu        sync.RWMutex
	databases map[string]*entities.UserDatabase
	passwords map[string]string // id -> raw password
}

// NewMemoryDatabaseRepository creates a new in-memory database repository.
func NewMemoryDatabaseRepository() *MemoryDatabaseRepository {
	return &MemoryDatabaseRepository{
		databases: make(map[string]*entities.UserDatabase),
		passwords: make(map[string]string),
	}
}

func (r *MemoryDatabaseRepository) CreateDatabase(_ context.Context, appID, userID string, input *dto.CreateDatabaseInput) (*entities.UserDatabase, error) {
	if appID == "" {
		return nil, fmt.Errorf("app_id is required")
	}
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	engine := input.Engine
	if engine == "" {
		engine = entities.DatabaseEnginePostgres
	}
	if engine != entities.DatabaseEnginePostgres && engine != entities.DatabaseEngineMySQL && engine != entities.DatabaseEngineRedis {
		return nil, fmt.Errorf("unsupported engine: %s", engine)
	}

	// Check for duplicate: one DB per engine per app
	r.mu.RLock()
	for _, db := range r.databases {
		if db.AppID == appID && db.Engine == engine {
			r.mu.RUnlock()
			return nil, fmt.Errorf("app already has a %s database", engine)
		}
	}
	r.mu.RUnlock()

	name := input.Name
	if name == "" {
		suffix := appID
		if len(suffix) > 8 {
			suffix = suffix[:8]
		}
		name = "db-" + suffix
	}

	id := uuid.New().String()
	dbName := sanitizeDBName(userID, appID)
	dbUser := "u_" + id[:8]
	password := generatePassword()

	port := 5432
	if engine == entities.DatabaseEngineMySQL {
		port = 3306
	} else if engine == entities.DatabaseEngineRedis {
		port = 6379
	}

	db := &entities.UserDatabase{
		ID:        id,
		AppID:     appID,
		UserID:    userID,
		Name:      name,
		Engine:    engine,
		DBName:    dbName,
		DBUser:    dbUser,
		Host:      "localhost",
		Port:      port,
		SizeMB:    0,
		MaxSizeMB: 500, // default free tier
		Status:    entities.DatabaseStatusReady,
		Timestamps: entities.Timestamps{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	r.mu.Lock()
	r.databases[id] = db
	r.passwords[id] = password
	r.mu.Unlock()

	return db, nil
}

func (r *MemoryDatabaseRepository) GetDatabase(_ context.Context, id string) (*entities.UserDatabase, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	db, ok := r.databases[id]
	if !ok {
		return nil, fmt.Errorf("database not found: %s", id)
	}
	return db, nil
}

func (r *MemoryDatabaseRepository) ListDatabasesByApp(_ context.Context, appID string) ([]entities.UserDatabase, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.UserDatabase
	for _, db := range r.databases {
		if db.AppID == appID {
			result = append(result, *db)
		}
	}
	return result, nil
}

func (r *MemoryDatabaseRepository) ListDatabasesByUser(_ context.Context, userID string) ([]entities.UserDatabase, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.UserDatabase
	for _, db := range r.databases {
		if db.UserID == userID {
			result = append(result, *db)
		}
	}
	return result, nil
}

func (r *MemoryDatabaseRepository) DeleteDatabase(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.databases[id]; !ok {
		return fmt.Errorf("database not found: %s", id)
	}
	delete(r.databases, id)
	delete(r.passwords, id)
	return nil
}

func (r *MemoryDatabaseRepository) UpdateDatabaseStatus(_ context.Context, id string, status entities.DatabaseStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	db, ok := r.databases[id]
	if !ok {
		return fmt.Errorf("database not found: %s", id)
	}
	db.Status = status
	db.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryDatabaseRepository) CountDatabasesByUser(_ context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, db := range r.databases {
		if db.UserID == userID {
			count++
		}
	}
	return count, nil
}

// GetPassword returns the raw password for a database (used to build connection string).
func (r *MemoryDatabaseRepository) GetPassword(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.passwords[id]
	return p, ok
}

// sanitizeDBName creates a safe database name from user and app IDs.
func sanitizeDBName(userID, appID string) string {
	name := "z_" + userID
	if len(name) > 12 {
		name = name[:12]
	}
	name += "_" + appID
	if len(name) > 24 {
		name = name[:24]
	}
	// Replace non-alphanumeric with underscore
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32 // lowercase
		}
		return '_'
	}, name)
	return safe
}

// generatePassword creates a random 24-char alphanumeric password.
func generatePassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 24)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
