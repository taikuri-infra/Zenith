// services/api/internal/services/managed_env.go
package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// EnvKeyForEngine is the variable name an attached database injects into its app.
func EnvKeyForEngine(engine entities.DatabaseEngine) string {
	switch engine {
	case entities.DatabaseEngineRedis:
		return "REDIS_URL"
	case entities.DatabaseEngineMySQL:
		return "MYSQL_URL"
	case entities.DatabaseEngineMongoDB:
		return "MONGODB_URL"
	case entities.DatabaseEngineRabbitMQ:
		return "RABBITMQ_URL"
	case entities.DatabaseEngineKafka:
		return "KAFKA_BROKERS"
	default:
		return "DATABASE_URL"
	}
}

// Encrypter encrypts a secret for a user; *crypto.EnvCrypto satisfies it.
type Encrypter interface {
	Encrypt(userID, value string) (string, error)
}

// ManagedEnv writes the connection variables of attached databases as env-v2
// rows (source managed_service), so the Environment tab shows them and a
// detach removes exactly that database's variables.
type ManagedEnv struct {
	vars ports.EnvVarRepository
	apps ports.AppRepository
	enc  Encrypter
}

func NewManagedEnv(vars ports.EnvVarRepository, apps ports.AppRepository) *ManagedEnv {
	return &ManagedEnv{vars: vars, apps: apps}
}

// SetEncrypter makes Set store values encrypted, like secrets set from the dashboard.
func (m *ManagedEnv) SetEncrypter(e Encrypter) { m.enc = e }

// Set upserts the variable for the app's default environment, encrypting it when an encrypter is configured.
func (m *ManagedEnv) Set(ctx context.Context, appID, key, value, dbID string) error {
	stored := value
	if m.enc != nil {
		app, err := m.apps.GetApp(ctx, appID)
		if err != nil {
			return fmt.Errorf("managed env: app %s: %w", appID, err)
		}
		enc, err := m.enc.Encrypt(app.UserID, value)
		if err != nil {
			return fmt.Errorf("managed env: encrypt %s: %w", key, err)
		}
		stored = enc
	}
	return m.SetRaw(ctx, appID, key, stored, dbID)
}

// SetRaw upserts an already-stored value (used by the migration, which must not re-encrypt).
// A variable the user set themselves under the same key is never overwritten.
func (m *ManagedEnv) SetRaw(ctx context.Context, appID, key, storedValue, dbID string) error {
	_, err := m.setRaw(ctx, appID, key, storedValue, dbID)
	return err
}

// setRaw reports whether the value was written (false when a non-managed variable takes precedence).
func (m *ManagedEnv) setRaw(ctx context.Context, appID, key, storedValue, dbID string) (bool, error) {
	existing, err := m.vars.GetEnvVarsByEnvironment(ctx, appID, "")
	if err != nil {
		return false, err
	}
	for _, v := range existing {
		if v.Key == key && v.Source != entities.EnvVarSourceManagedService {
			slog.Warn(fmt.Sprintf("managed env: manual variable %s on app %s takes precedence over database %s", key, appID, dbID))
			return false, nil
		}
	}
	err = m.vars.SetEnvVar(ctx, &entities.AppEnvVar{
		AppID: appID, EnvironmentID: "", Key: key, Value: storedValue, IsSecret: true,
		Source: entities.EnvVarSourceManagedService, SourceID: dbID,
	})
	return err == nil, err
}

// Clear deletes the variables that database dbID injected into appID; other sources and databases are untouched.
func (m *ManagedEnv) Clear(ctx context.Context, appID, dbID string) error {
	vars, err := m.vars.GetEnvVarsByEnvironment(ctx, appID, "")
	if err != nil {
		return err
	}
	for _, v := range vars {
		if v.Source == entities.EnvVarSourceManagedService && v.SourceID == dbID {
			if err := m.vars.DeleteEnvVar(ctx, v.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// Has reports whether appID already holds a variable from database dbID.
func (m *ManagedEnv) Has(ctx context.Context, appID, dbID string) (bool, error) {
	vars, err := m.vars.GetEnvVarsByEnvironment(ctx, appID, "")
	if err != nil {
		return false, err
	}
	for _, v := range vars {
		if v.Source == entities.EnvVarSourceManagedService && v.SourceID == dbID {
			return true, nil
		}
	}
	return false, nil
}

// MigrateManagedEnv copies, once, the legacy-table connection variable of every
// ready, attached database into env-v2 and deletes the legacy row after the copy.
// It is idempotent: a database whose env-v2 variable exists is skipped.
func MigrateManagedEnv(ctx context.Context, dbs ports.DatabaseRepository, apps ports.AppRepository, env *ManagedEnv) (int, error) {
	ready, err := dbs.ListDatabasesByStatus(ctx, entities.DatabaseStatusReady)
	if err != nil {
		return 0, err
	}
	type group struct{ appID, key string }
	counts := map[group]int{}
	for i := range ready {
		if ready[i].AppID != "" {
			counts[group{ready[i].AppID, EnvKeyForEngine(ready[i].Engine)}]++
		}
	}
	warned := map[group]bool{}
	migrated := 0
	for i := range ready {
		db := &ready[i]
		if db.AppID == "" {
			continue
		}
		if g := (group{db.AppID, EnvKeyForEngine(db.Engine)}); counts[g] > 1 {
			if !warned[g] {
				warned[g] = true
				slog.Warn(fmt.Sprintf("managed env migration: app %s has %d ready %s databases; cannot attribute %s, skipping", g.appID, counts[g], db.Engine, g.key))
			}
			continue
		}
		has, err := env.Has(ctx, db.AppID, db.ID)
		if err != nil || has {
			continue
		}
		key := EnvKeyForEngine(db.Engine)
		legacy, err := apps.GetEnvVars(ctx, db.AppID)
		if err != nil {
			slog.Warn("managed env migration: read legacy vars", "app", db.AppID, "error", err)
			continue
		}
		for _, l := range legacy {
			if l.Key != key {
				continue
			}
			written, err := env.setRaw(ctx, db.AppID, key, l.Value, db.ID)
			if err != nil {
				slog.Warn("managed env migration: copy failed; legacy row kept", "app", db.AppID, "key", key, "error", err)
				break
			}
			if !written {
				break
			}
			if err := apps.DeleteEnvVar(ctx, db.AppID, key); err != nil {
				slog.Warn("managed env migration: legacy row not deleted", "app", db.AppID, "key", key, "error", err)
			}
			slog.Info("managed env migration: moved variable to env-v2", "app", db.AppID, "database", db.Name, "key", key)
			migrated++
			break
		}
	}
	return migrated, nil
}
