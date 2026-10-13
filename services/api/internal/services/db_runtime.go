// services/api/internal/services/db_runtime.go
package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pkgCrypto "github.com/dotechhq/zenith/services/api/internal/crypto"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// DBBackend runs database instances (Docker containers in standalone mode).
type DBBackend interface {
	Provision(ctx context.Context, db *entities.UserDatabase, password string) (host string, port int, err error)
	WaitReady(ctx context.Context, db *entities.UserDatabase) error
	Delete(ctx context.Context, db *entities.UserDatabase, purge bool) error
}

// HostPorter is an optional DBBackend capability: the host-side port for SSH access.
type HostPorter interface {
	HostPort(ctx context.Context, db *entities.UserDatabase) (int, error)
}

// DBRuntimeService creates database records and provisions them on a DBBackend.
type DBRuntimeService struct {
	repo    ports.DatabaseRepository
	backend DBBackend
	crypto  *pkgCrypto.EnvCrypto
	wg      sync.Map // id -> *sync.WaitGroup

	provisionTimeout time.Duration
}

func NewDBRuntimeService(repo ports.DatabaseRepository, backend DBBackend, crypto *pkgCrypto.EnvCrypto) *DBRuntimeService {
	return &DBRuntimeService{repo: repo, backend: backend, crypto: crypto, provisionTimeout: 5 * time.Minute}
}

// SupportedEngine reports whether the Docker runtime can run engine.
func SupportedEngine(e entities.DatabaseEngine) bool {
	switch e {
	case entities.DatabaseEnginePostgres, entities.DatabaseEngineMySQL, entities.DatabaseEngineRedis:
		return true
	}
	return false
}

func newDBPassword() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil // 36 hex chars, URL-safe
}

// Create inserts the record and provisions it in the background. The returned
// database has status provisioning; it turns ready or error when done.
func (s *DBRuntimeService) Create(ctx context.Context, appID, userID string, in *dto.CreateDatabaseInput) (*entities.UserDatabase, error) {
	if in.Engine == "" {
		in.Engine = entities.DatabaseEnginePostgres
	}
	if !SupportedEngine(in.Engine) {
		return nil, fmt.Errorf("engine %s is not available on this install", in.Engine)
	}
	db, err := s.repo.CreateDatabase(ctx, appID, userID, in)
	if err != nil {
		return nil, err
	}
	pw, err := newDBPassword()
	if err != nil {
		return nil, fmt.Errorf("generate password: %w", err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	s.wg.Store(db.ID, wg)
	go func() {
		defer wg.Done()
		bg, cancel := context.WithTimeout(context.Background(), s.provisionTimeout)
		defer cancel()
		if err := s.provision(bg, db, pw); err != nil {
			slog.Error("database provisioning failed", "db", db.Name, "engine", db.Engine, "error", err)
			// bg may already be expired; record the failure on a fresh context.
			upd, ucancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer ucancel()
			if uerr := s.repo.UpdateDatabaseStatus(upd, db.ID, entities.DatabaseStatusError); uerr != nil {
				slog.Error("could not mark database as failed", "db", db.Name, "error", uerr)
			}
		}
	}()
	return db, nil
}

func (s *DBRuntimeService) provision(ctx context.Context, db *entities.UserDatabase, pw string) error {
	host, port, err := s.backend.Provision(ctx, db, pw)
	if err != nil {
		return err
	}
	if err := s.backend.WaitReady(ctx, db); err != nil {
		return err
	}
	secret := pw
	if s.crypto != nil {
		if secret, err = s.crypto.Encrypt(db.UserID, pw); err != nil {
			return fmt.Errorf("encrypt password: %w", err)
		}
	}
	if err := s.repo.SetDatabaseProvisioned(ctx, db.ID, host, port, secret); err != nil {
		return err
	}
	if hp, ok := s.backend.(HostPorter); ok {
		if port, err := hp.HostPort(ctx, db); err != nil {
			slog.Warn("database host port unknown", "db", db.Name, "error", err)
		} else if err := s.repo.SetDatabaseHostPort(ctx, db.ID, port); err != nil {
			slog.Warn("store database host port failed", "db", db.Name, "error", err)
		}
	}
	return nil
}

// Reprovision recreates a provisioned database's instance with its stored
// password. The data volume is kept, so this is safe for a container that
// vanished or a restore onto a new host.
func (s *DBRuntimeService) Reprovision(ctx context.Context, db *entities.UserDatabase) error {
	pw, err := s.Password(ctx, db)
	if err != nil {
		return err
	}
	return s.provision(ctx, db, pw)
}

// Wait blocks until the background provisioning of id has finished (tests).
func (s *DBRuntimeService) Wait(id string) {
	if v, ok := s.wg.Load(id); ok {
		v.(*sync.WaitGroup).Wait()
		s.wg.Delete(id)
	}
}

// Password returns the plaintext password of a provisioned database.
func (s *DBRuntimeService) Password(ctx context.Context, db *entities.UserDatabase) (string, error) {
	secret, err := s.repo.GetDatabaseSecret(ctx, db.ID)
	if err != nil || secret == "" {
		return "", fmt.Errorf("no credentials yet")
	}
	if pkgCrypto.IsEncrypted(secret) {
		if s.crypto == nil {
			return "", fmt.Errorf("database credentials are encrypted but no encryption key is configured")
		}
		return s.crypto.Decrypt(db.UserID, secret)
	}
	return secret, nil
}

// Delete removes the instance and its record. purge also drops the data volume.
func (s *DBRuntimeService) Delete(ctx context.Context, db *entities.UserDatabase, purge bool) error {
	if err := s.backend.Delete(ctx, db, purge); err != nil {
		return err
	}
	return s.repo.DeleteDatabase(ctx, db.ID)
}
