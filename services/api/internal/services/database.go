package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/postgres"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/jackc/pgx/v5"
)

// DatabaseService handles real database provisioning via CNPG.
type DatabaseService struct {
	dbRepo    ports.DatabaseRepository
	appRepo   ports.AppRepository
	planRepo  ports.UserPlanRepository
	k8sClient k8sclient.Client
	adminDSN  string // CNPG admin connection string (shared cluster)
	cnpgHost  string // parsed host from adminDSN
	namespace string // K8s namespace for secrets/CRDs
}

// NewDatabaseService creates a new DatabaseService.
func NewDatabaseService(
	dbRepo ports.DatabaseRepository,
	appRepo ports.AppRepository,
	planRepo ports.UserPlanRepository,
	k8sClient k8sclient.Client,
	adminDSN string,
	namespace string,
) *DatabaseService {
	host := parseDSNHost(adminDSN)
	return &DatabaseService{
		dbRepo:    dbRepo,
		appRepo:   appRepo,
		planRepo:  planRepo,
		k8sClient: k8sClient,
		adminDSN:  adminDSN,
		cnpgHost:  host,
		namespace: namespace,
	}
}

// parseDSNHost extracts the host from a PostgreSQL DSN.
func parseDSNHost(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// generatePassword creates a cryptographically random hex password.
func generatePassword(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sanitizeIdentifier makes a string safe for use as a SQL identifier.
func sanitizeIdentifier(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// ProvisionDatabase creates a real database, either on the shared CNPG cluster
// (Free/Pro) or as a dedicated CNPG Cluster CRD (Team/Enterprise).
func (s *DatabaseService) ProvisionDatabase(ctx context.Context, appID, userID string, input *dto.CreateDatabaseInput) (*entities.UserDatabase, error) {
	// 1. Create metadata record (status: provisioning)
	db, err := s.dbRepo.CreateDatabase(ctx, appID, userID, input)
	if err != nil {
		return nil, err
	}

	// 2. Determine tier
	plan, err := s.planRepo.GetUserPlan(ctx, userID)
	if err != nil {
		log.Printf("[db] Warning: could not get user plan for %s, defaulting to shared: %v", userID, err)
		plan = &entities.UserPlan{Tier: entities.PlanFree}
	}

	var password string
	var host string
	var provisioner entities.DatabaseProvisioner

	switch plan.Tier {
	case entities.PlanTeam, entities.PlanEnterprise:
		host, password, err = s.provisionDedicated(ctx, db, plan.Tier)
		provisioner = entities.DBProvisionerDedicated
	default:
		host, password, err = s.provisionShared(ctx, db)
		provisioner = entities.DBProvisionerShared
	}

	if err != nil {
		// Best-effort cleanup
		s.dbRepo.DeleteDatabase(ctx, db.ID)
		return nil, fmt.Errorf("provision database: %w", err)
	}

	// 3. Update metadata with host and provisioner
	if pgRepo, ok := s.dbRepo.(*postgres.PostgresDatabaseRepository); ok {
		pgRepo.UpdateDatabaseHost(ctx, db.ID, host, provisioner)
	} else {
		s.dbRepo.UpdateDatabaseStatus(ctx, db.ID, entities.DatabaseStatusReady)
	}
	db.Host = host
	db.Status = entities.DatabaseStatusReady
	db.Provisioner = provisioner

	// 4. Store credentials in K8s Secret
	secretName := "db-" + db.ID[:8] + "-credentials"
	connStr := db.ConnectionString(password)
	secretData := map[string][]byte{
		"DATABASE_URL": []byte(connStr),
		"DB_PASSWORD":  []byte(password),
		"DB_HOST":      []byte(host),
		"DB_NAME":      []byte(db.DBName),
		"DB_USER":      []byte(db.DBUser),
		"DB_PORT":      []byte("5432"),
	}
	if err := s.k8sClient.CreateSecret(ctx, s.namespace, secretName, secretData, map[string]string{
		"zenith.io/database": db.ID,
		"zenith.io/user":     userID,
	}); err != nil {
		log.Printf("[db] Warning: failed to create K8s secret %s: %v", secretName, err)
	}

	// 5. Auto-inject DATABASE_URL env var on the app
	if appID != "" {
		envKey := envKeyForDBEngine(db.Engine)
		s.appRepo.SetEnvVars(ctx, appID, map[string]string{envKey: connStr})
	}

	return db, nil
}

// provisionShared creates a database and user on the shared CNPG cluster.
func (s *DatabaseService) provisionShared(ctx context.Context, db *entities.UserDatabase) (host, password string, err error) {
	password, err = generatePassword(16)
	if err != nil {
		return "", "", fmt.Errorf("generate password: %w", err)
	}

	conn, err := pgx.Connect(ctx, s.adminDSN)
	if err != nil {
		return "", "", fmt.Errorf("connect to CNPG admin: %w", err)
	}
	defer conn.Close(ctx)

	dbUser := sanitizeIdentifier(db.DBUser)
	dbName := sanitizeIdentifier(db.DBName)

	// CREATE USER (use format string — pgx doesn't support parameterized DDL)
	_, err = conn.Exec(ctx, fmt.Sprintf(`CREATE USER %q WITH PASSWORD '%s'`, dbUser, password))
	if err != nil {
		return "", "", fmt.Errorf("create user %s: %w", dbUser, err)
	}

	// CREATE DATABASE
	_, err = conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %q OWNER %q`, dbName, dbUser))
	if err != nil {
		// Cleanup user on failure
		conn.Exec(ctx, fmt.Sprintf(`DROP USER IF EXISTS %q`, dbUser))
		return "", "", fmt.Errorf("create database %s: %w", dbName, err)
	}

	return s.cnpgHost, password, nil
}

// provisionDedicated creates a CNPG Cluster CRD for Team/Enterprise tiers.
func (s *DatabaseService) provisionDedicated(ctx context.Context, db *entities.UserDatabase, tier entities.PlanTier) (host, password string, err error) {
	clusterName := "db-" + db.ID[:8]

	instances := 1
	if tier == entities.PlanEnterprise {
		instances = 3
	}

	storageSizeMB := db.MaxSizeMB
	if storageSizeMB <= 0 {
		storageSizeMB = 1024 // 1Gi default
	}

	spec := map[string]interface{}{
		"instances": instances,
		"storage": map[string]interface{}{
			"size": fmt.Sprintf("%dMi", storageSizeMB),
		},
		"postgresql": map[string]interface{}{
			"parameters": map[string]interface{}{
				"max_connections": "50",
			},
		},
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return "", "", fmt.Errorf("marshal CNPG spec: %w", err)
	}

	crd := &k8sclient.CRDObject{
		APIVersion: "postgresql.cnpg.io/v1",
		Kind:       "Cluster",
		Metadata: k8sclient.ObjectMeta{
			Name:      clusterName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"zenith.io/database": db.ID,
				"zenith.io/user":     db.UserID,
			},
		},
		Spec: specJSON,
	}

	if err := s.k8sClient.CreateCRD(ctx, crd); err != nil {
		return "", "", fmt.Errorf("create CNPG cluster CRD: %w", err)
	}

	// CNPG operator auto-creates: {clusterName}-superuser secret with username/password
	// For now, set status to provisioning — the password will be read from the secret once ready
	host = clusterName + "-rw." + s.namespace + ".svc.cluster.local"

	// Try to read CNPG-generated password (may not exist yet if operator is still provisioning)
	secretName := clusterName + "-superuser"
	secretData, err := s.k8sClient.GetSecret(ctx, s.namespace, secretName)
	if err == nil && len(secretData["password"]) > 0 {
		password = string(secretData["password"])
	} else {
		// Generate a temporary password; operator will set the real one
		password, _ = generatePassword(16)
		log.Printf("[db] CNPG secret %s not ready yet, using generated password", secretName)
	}

	return host, password, nil
}

// DeleteDatabase deprovisions and removes a database.
func (s *DatabaseService) DeleteDatabase(ctx context.Context, id string) error {
	db, err := s.dbRepo.GetDatabase(ctx, id)
	if err != nil {
		return err
	}

	// Mark as deleting
	s.dbRepo.UpdateDatabaseStatus(ctx, id, entities.DatabaseStatusDeleting)

	switch db.Provisioner {
	case entities.DBProvisionerDedicated:
		// Delete CNPG Cluster CRD
		clusterName := "db-" + db.ID[:8]
		if err := s.k8sClient.DeleteCRDWithVersion(ctx, "postgresql.cnpg.io/v1", "Cluster", s.namespace, clusterName); err != nil {
			log.Printf("[db] Warning: failed to delete CNPG cluster %s: %v", clusterName, err)
		}
	default:
		// Drop database and user on shared cluster
		s.dropSharedDatabase(ctx, db)
	}

	// Delete K8s Secret
	secretName := "db-" + db.ID[:8] + "-credentials"
	if err := s.k8sClient.DeleteSecret(ctx, s.namespace, secretName); err != nil {
		log.Printf("[db] Warning: failed to delete K8s secret %s: %v", secretName, err)
	}

	// Remove auto-injected env var
	if db.AppID != "" {
		envKey := envKeyForDBEngine(db.Engine)
		s.appRepo.DeleteEnvVar(ctx, db.AppID, envKey)
	}

	// Delete metadata
	return s.dbRepo.DeleteDatabase(ctx, id)
}

// dropSharedDatabase removes the database and user from the shared CNPG cluster.
func (s *DatabaseService) dropSharedDatabase(ctx context.Context, db *entities.UserDatabase) {
	conn, err := pgx.Connect(ctx, s.adminDSN)
	if err != nil {
		log.Printf("[db] Warning: could not connect to CNPG for cleanup: %v", err)
		return
	}
	defer conn.Close(ctx)

	dbName := sanitizeIdentifier(db.DBName)
	dbUser := sanitizeIdentifier(db.DBUser)

	// Terminate active connections before dropping
	conn.Exec(ctx, fmt.Sprintf(
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()`,
		dbName,
	))

	if _, err := conn.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %q`, dbName)); err != nil {
		log.Printf("[db] Warning: DROP DATABASE %s failed: %v", dbName, err)
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf(`DROP USER IF EXISTS %q`, dbUser)); err != nil {
		log.Printf("[db] Warning: DROP USER %s failed: %v", dbUser, err)
	}
}

// GetDatabasePassword reads the password from the K8s Secret for a database.
func (s *DatabaseService) GetDatabasePassword(ctx context.Context, id string) (string, error) {
	secretName := "db-" + id[:8] + "-credentials"
	data, err := s.k8sClient.GetSecret(ctx, s.namespace, secretName)
	if err != nil {
		return "", fmt.Errorf("database credentials not found")
	}
	pw, ok := data["DB_PASSWORD"]
	if !ok {
		return "", fmt.Errorf("password not found in secret")
	}
	return string(pw), nil
}

func envKeyForDBEngine(engine entities.DatabaseEngine) string {
	switch engine {
	case entities.DatabaseEngineRedis:
		return "REDIS_URL"
	case entities.DatabaseEngineMySQL:
		return "MYSQL_URL"
	default:
		return "DATABASE_URL"
	}
}
