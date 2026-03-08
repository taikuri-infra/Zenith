package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
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
	// 1. Determine tier and set max size from plan limits
	plan, err := s.planRepo.GetUserPlan(ctx, userID)
	if err != nil {
		slog.Warn("could not get user plan, defaulting to shared", "user_id", userID, "error", err)
		plan = &entities.UserPlan{Tier: entities.PlanFree}
	}

	limits := entities.DefaultPlanLimits(plan.Tier)
	if input.MaxSizeMB <= 0 {
		input.MaxSizeMB = limits.MaxDBSizeMB
	}

	// 2. Create metadata record (status: provisioning)
	db, err := s.dbRepo.CreateDatabase(ctx, appID, userID, input)
	if err != nil {
		return nil, err
	}

	// Engine-specific plan gating
	switch input.Engine {
	case entities.DatabaseEngineRedis:
		if limits.MaxRedisInstances <= 0 {
			s.dbRepo.DeleteDatabase(ctx, db.ID)
			return nil, fmt.Errorf("Redis is not available on the %s plan. Upgrade to Pro or higher", plan.Tier)
		}
	case entities.DatabaseEngineRabbitMQ:
		if limits.MaxRabbitMQInstances <= 0 {
			s.dbRepo.DeleteDatabase(ctx, db.ID)
			return nil, fmt.Errorf("RabbitMQ is not available on the %s plan. Upgrade to Pro or higher", plan.Tier)
		}
	case entities.DatabaseEngineMongoDB:
		if limits.MaxMongoDBInstances <= 0 {
			s.dbRepo.DeleteDatabase(ctx, db.ID)
			return nil, fmt.Errorf("MongoDB is not available on the %s plan. Upgrade to Team or higher", plan.Tier)
		}
	case entities.DatabaseEngineKafka:
		if limits.MaxKafkaInstances <= 0 {
			s.dbRepo.DeleteDatabase(ctx, db.ID)
			return nil, fmt.Errorf("Kafka is not available on the %s plan. Upgrade to Business or higher", plan.Tier)
		}
	}

	var password string
	var host string
	var provisioner entities.DatabaseProvisioner

	// Engine-specific provisioning
	switch db.Engine {
	case entities.DatabaseEngineRedis:
		host, password, err = s.provisionRedis(ctx, db, plan.Tier)
		provisioner = entities.DBProvisionerDedicated
	case entities.DatabaseEngineRabbitMQ:
		host, password, err = s.provisionRabbitMQ(ctx, db, plan.Tier)
		provisioner = entities.DBProvisionerDedicated
	case entities.DatabaseEngineMongoDB:
		host, password, err = s.provisionMongoDB(ctx, db, plan.Tier)
		provisioner = entities.DBProvisionerDedicated
	case entities.DatabaseEngineKafka:
		host, password, err = s.provisionKafka(ctx, db, plan.Tier)
		provisioner = entities.DBProvisionerDedicated
	default:
		switch plan.Tier {
		case entities.PlanTeam, entities.PlanBusiness, entities.PlanEnterprise:
			host, password, err = s.provisionDedicated(ctx, db, plan.Tier)
			provisioner = entities.DBProvisionerDedicated
		default:
			host, password, err = s.provisionShared(ctx, db)
			provisioner = entities.DBProvisionerShared
		}
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
	port := portForEngine(db.Engine)
	envKey := envKeyForDBEngine(db.Engine)
	secretData := map[string][]byte{
		envKey:        []byte(connStr),
		"DB_PASSWORD": []byte(password),
		"DB_HOST":     []byte(host),
		"DB_NAME":     []byte(db.DBName),
		"DB_USER":     []byte(db.DBUser),
		"DB_PORT":     []byte(port),
	}
	if err := s.k8sClient.CreateSecret(ctx, s.namespace, secretName, secretData, map[string]string{
		"zenith.io/database": db.ID,
		"zenith.io/user":     userID,
	}); err != nil {
		slog.Warn("failed to create K8s secret", "secret", secretName, "error", err)
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

// provisionDedicated creates a CNPG Cluster CRD for Team/Business/Enterprise tiers.
func (s *DatabaseService) provisionDedicated(ctx context.Context, db *entities.UserDatabase, tier entities.PlanTier) (host, password string, err error) {
	clusterName := "db-" + db.ID[:8]

	instances := 1
	if tier == entities.PlanBusiness || tier == entities.PlanEnterprise {
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
		slog.Info("CNPG secret not ready yet, using generated password", "secret", secretName)
	}

	return host, password, nil
}

// provisionRedis creates a RedisFailover CRD via the Spotahome Redis Operator.
// Redis always gets its own instance (no shared mode like PostgreSQL).
func (s *DatabaseService) provisionRedis(ctx context.Context, db *entities.UserDatabase, tier entities.PlanTier) (host, password string, err error) {
	password, err = generatePassword(16)
	if err != nil {
		return "", "", fmt.Errorf("generate password: %w", err)
	}

	instanceName := "redis-" + db.ID[:8]

	// Scale based on plan tier
	redisReplicas := 1
	sentinelReplicas := 0
	maxMemoryMB := 128
	switch tier {
	case entities.PlanTeam:
		redisReplicas = 2
		sentinelReplicas = 3
		maxMemoryMB = 512
	case entities.PlanBusiness, entities.PlanEnterprise:
		redisReplicas = 3
		sentinelReplicas = 3
		maxMemoryMB = 2048
	}

	if db.MaxSizeMB > 0 && db.MaxSizeMB < maxMemoryMB {
		maxMemoryMB = db.MaxSizeMB
	}

	spec := map[string]interface{}{
		"redis": map[string]interface{}{
			"replicas": redisReplicas,
			"customConfig": []string{
				fmt.Sprintf("maxmemory %dmb", maxMemoryMB),
				"maxmemory-policy allkeys-lru",
				fmt.Sprintf("requirepass %s", password),
			},
			"resources": map[string]interface{}{
				"requests": map[string]string{
					"cpu":    "50m",
					"memory": fmt.Sprintf("%dMi", maxMemoryMB+64),
				},
				"limits": map[string]string{
					"cpu":    "500m",
					"memory": fmt.Sprintf("%dMi", maxMemoryMB*2),
				},
			},
		},
	}

	// Add Sentinel for HA (Team/Enterprise)
	if sentinelReplicas > 0 {
		spec["sentinel"] = map[string]interface{}{
			"replicas": sentinelReplicas,
			"resources": map[string]interface{}{
				"requests": map[string]string{"cpu": "25m", "memory": "64Mi"},
				"limits":   map[string]string{"cpu": "100m", "memory": "128Mi"},
			},
		}
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return "", "", fmt.Errorf("marshal Redis spec: %w", err)
	}

	crd := &k8sclient.CRDObject{
		APIVersion: "databases.spotahome.com/v1",
		Kind:       "RedisFailover",
		Metadata: k8sclient.ObjectMeta{
			Name:      instanceName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"zenith.io/database": db.ID,
				"zenith.io/user":     db.UserID,
				"zenith.io/engine":   "redis",
			},
		},
		Spec: specJSON,
	}

	if err := s.k8sClient.CreateCRD(ctx, crd); err != nil {
		return "", "", fmt.Errorf("create RedisFailover CRD: %w", err)
	}

	// Redis service DNS: rfr-<name>.<namespace>.svc.cluster.local (read-write master)
	host = "rfr-" + instanceName + "." + s.namespace + ".svc.cluster.local"

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

	switch db.Engine {
	case entities.DatabaseEngineRedis:
		instanceName := "redis-" + db.ID[:8]
		if err := s.k8sClient.DeleteCRDWithVersion(ctx, "databases.spotahome.com/v1", "RedisFailover", s.namespace, instanceName); err != nil {
			slog.Warn("failed to delete RedisFailover", "name", instanceName, "error", err)
		}
	case entities.DatabaseEngineRabbitMQ:
		instanceName := "rmq-" + db.ID[:8]
		if err := s.k8sClient.DeleteCRDWithVersion(ctx, "rabbitmq.com/v1beta1", "RabbitmqCluster", s.namespace, instanceName); err != nil {
			slog.Warn("failed to delete RabbitmqCluster", "name", instanceName, "error", err)
		}
	case entities.DatabaseEngineMongoDB:
		instanceName := "mongo-" + db.ID[:8]
		if err := s.k8sClient.DeleteCRDWithVersion(ctx, "psmdb.percona.com/v1", "PerconaServerMongoDB", s.namespace, instanceName); err != nil {
			slog.Warn("failed to delete PerconaServerMongoDB", "name", instanceName, "error", err)
		}
	case entities.DatabaseEngineKafka:
		instanceName := "kafka-" + db.ID[:8]
		if err := s.k8sClient.DeleteCRDWithVersion(ctx, "kafka.strimzi.io/v1beta2", "Kafka", s.namespace, instanceName); err != nil {
			slog.Warn("failed to delete Kafka", "name", instanceName, "error", err)
		}
	default:
		switch db.Provisioner {
		case entities.DBProvisionerDedicated:
			clusterName := "db-" + db.ID[:8]
			if err := s.k8sClient.DeleteCRDWithVersion(ctx, "postgresql.cnpg.io/v1", "Cluster", s.namespace, clusterName); err != nil {
				slog.Warn("failed to delete CNPG cluster", "name", clusterName, "error", err)
			}
		default:
			s.dropSharedDatabase(ctx, db)
		}
	}

	// Delete K8s Secret
	secretName := "db-" + db.ID[:8] + "-credentials"
	if err := s.k8sClient.DeleteSecret(ctx, s.namespace, secretName); err != nil {
		slog.Warn("failed to delete K8s secret", "secret", secretName, "error", err)
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
		slog.Warn("could not connect to CNPG for cleanup", "error", err)
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
		slog.Warn("DROP DATABASE failed", "database", dbName, "error", err)
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf(`DROP USER IF EXISTS %q`, dbUser)); err != nil {
		slog.Warn("DROP USER failed", "user", dbUser, "error", err)
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

// ResetDatabasePassword generates a new password, updates it in PostgreSQL,
// recreates the K8s Secret, and updates the app env var if linked.
func (s *DatabaseService) ResetDatabasePassword(ctx context.Context, id string) (string, string, error) {
	db, err := s.dbRepo.GetDatabase(ctx, id)
	if err != nil {
		return "", "", fmt.Errorf("database not found: %w", err)
	}

	// Generate new password
	newPassword, err := generatePassword(16)
	if err != nil {
		return "", "", fmt.Errorf("generate password: %w", err)
	}

	// Engine-specific password reset
	switch db.Engine {
	case entities.DatabaseEngineRabbitMQ:
		return s.resetRabbitMQPassword(ctx, db, newPassword)
	case entities.DatabaseEngineMongoDB:
		return s.resetMongoDBPassword(ctx, db, newPassword)
	case entities.DatabaseEngineKafka:
		return s.resetKafkaPassword(ctx, db, newPassword)
	}

	// Redis: update the CRD customConfig with new requirepass
	if db.Engine == entities.DatabaseEngineRedis {
		instanceName := "redis-" + db.ID[:8]
		existing, err := s.k8sClient.GetCRD(ctx, "RedisFailover", s.namespace, instanceName)
		if err != nil {
			return "", "", fmt.Errorf("get Redis CRD: %w", err)
		}
		// Parse existing spec and update requirepass
		var spec map[string]interface{}
		if err := json.Unmarshal(existing.Spec, &spec); err != nil {
			return "", "", fmt.Errorf("parse Redis spec: %w", err)
		}
		if redis, ok := spec["redis"].(map[string]interface{}); ok {
			if configs, ok := redis["customConfig"].([]interface{}); ok {
				newConfigs := make([]interface{}, 0, len(configs))
				for _, c := range configs {
					if s, ok := c.(string); ok && !strings.HasPrefix(s, "requirepass ") {
						newConfigs = append(newConfigs, s)
					}
				}
				newConfigs = append(newConfigs, fmt.Sprintf("requirepass %s", newPassword))
				redis["customConfig"] = newConfigs
			}
		}
		specJSON, _ := json.Marshal(spec)
		existing.Spec = specJSON
		if err := s.k8sClient.UpdateCRD(ctx, existing); err != nil {
			return "", "", fmt.Errorf("update Redis CRD: %w", err)
		}

		// Update K8s Secret and env var
		secretName := "db-" + db.ID[:8] + "-credentials"
		connStr := db.ConnectionString(newPassword)
		_ = s.k8sClient.DeleteSecret(ctx, s.namespace, secretName)
		secretData := map[string][]byte{
			"REDIS_URL":    []byte(connStr),
			"DB_PASSWORD":  []byte(newPassword),
			"DB_HOST":      []byte(db.Host),
			"DB_NAME":      []byte(db.DBName),
			"DB_USER":      []byte(db.DBUser),
			"DB_PORT":      []byte("6379"),
		}
		s.k8sClient.CreateSecret(ctx, s.namespace, secretName, secretData, map[string]string{
			"zenith.io/database": db.ID,
			"zenith.io/user":     db.UserID,
		})
		if db.AppID != "" {
			s.appRepo.SetEnvVars(ctx, db.AppID, map[string]string{"REDIS_URL": connStr})
		}
		return newPassword, connStr, nil
	}

	// PostgreSQL/MySQL: ALTER USER on the appropriate cluster
	switch db.Provisioner {
	case entities.DBProvisionerDedicated:
		// Read password from CNPG superuser secret
		clusterName := "db-" + db.ID[:8]
		secretName := clusterName + "-superuser"
		secretData, err := s.k8sClient.GetSecret(ctx, s.namespace, secretName)
		if err != nil {
			return "", "", fmt.Errorf("cannot read dedicated cluster superuser secret: %w", err)
		}
		adminDSN := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable",
			string(secretData["username"]), string(secretData["password"]),
			clusterName+"-rw."+s.namespace+".svc.cluster.local", db.DBName)
		conn, err := pgx.Connect(ctx, adminDSN)
		if err != nil {
			return "", "", fmt.Errorf("connect to dedicated cluster: %w", err)
		}
		defer conn.Close(ctx)
		_, err = conn.Exec(ctx, fmt.Sprintf(`ALTER USER %q WITH PASSWORD '%s'`, sanitizeIdentifier(db.DBUser), newPassword))
		if err != nil {
			return "", "", fmt.Errorf("alter user password: %w", err)
		}
	default:
		// Shared cluster
		conn, err := pgx.Connect(ctx, s.adminDSN)
		if err != nil {
			return "", "", fmt.Errorf("connect to CNPG admin: %w", err)
		}
		defer conn.Close(ctx)
		_, err = conn.Exec(ctx, fmt.Sprintf(`ALTER USER %q WITH PASSWORD '%s'`, sanitizeIdentifier(db.DBUser), newPassword))
		if err != nil {
			return "", "", fmt.Errorf("alter user password: %w", err)
		}
	}

	// Recreate K8s Secret with new credentials
	secretName := "db-" + db.ID[:8] + "-credentials"
	connStr := db.ConnectionString(newPassword)
	resetPort := portForEngine(db.Engine)
	resetEnvKey := envKeyForDBEngine(db.Engine)
	_ = s.k8sClient.DeleteSecret(ctx, s.namespace, secretName)
	secretData := map[string][]byte{
		resetEnvKey:   []byte(connStr),
		"DB_PASSWORD": []byte(newPassword),
		"DB_HOST":     []byte(db.Host),
		"DB_NAME":     []byte(db.DBName),
		"DB_USER":     []byte(db.DBUser),
		"DB_PORT":     []byte(resetPort),
	}
	if err := s.k8sClient.CreateSecret(ctx, s.namespace, secretName, secretData, map[string]string{
		"zenith.io/database": db.ID,
		"zenith.io/user":     db.UserID,
	}); err != nil {
		slog.Warn("failed to recreate K8s secret", "secret", secretName, "error", err)
	}

	// Update app env var if linked
	if db.AppID != "" {
		s.appRepo.SetEnvVars(ctx, db.AppID, map[string]string{resetEnvKey: connStr})
	}

	return newPassword, connStr, nil
}

func envKeyForDBEngine(engine entities.DatabaseEngine) string {
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

func portForEngine(engine entities.DatabaseEngine) string {
	switch engine {
	case entities.DatabaseEngineRedis:
		return "6379"
	case entities.DatabaseEngineMySQL:
		return "3306"
	case entities.DatabaseEngineMongoDB:
		return "27017"
	case entities.DatabaseEngineRabbitMQ:
		return "5672"
	case entities.DatabaseEngineKafka:
		return "9092"
	default:
		return "5432"
	}
}

// provisionRabbitMQ creates a RabbitmqCluster CRD via the RabbitMQ Cluster Operator.
func (s *DatabaseService) provisionRabbitMQ(ctx context.Context, db *entities.UserDatabase, tier entities.PlanTier) (host, password string, err error) {
	password, err = generatePassword(16)
	if err != nil {
		return "", "", fmt.Errorf("generate password: %w", err)
	}

	instanceName := "rmq-" + db.ID[:8]

	replicas := 1
	memoryMB := 256
	switch tier {
	case entities.PlanTeam:
		replicas = 1
		memoryMB = 512
	case entities.PlanBusiness, entities.PlanEnterprise:
		replicas = 3
		memoryMB = 1024
	}

	if db.MaxSizeMB > 0 && db.MaxSizeMB < memoryMB {
		memoryMB = db.MaxSizeMB
	}

	spec := map[string]interface{}{
		"replicas": replicas,
		"resources": map[string]interface{}{
			"requests": map[string]string{
				"cpu":    "100m",
				"memory": fmt.Sprintf("%dMi", memoryMB),
			},
			"limits": map[string]string{
				"cpu":    "500m",
				"memory": fmt.Sprintf("%dMi", memoryMB*2),
			},
		},
		"rabbitmq": map[string]interface{}{
			"additionalConfig": fmt.Sprintf("default_user = %s\ndefault_pass = %s\ndefault_vhost = %s\n", db.DBUser, password, db.DBName),
		},
		"persistence": map[string]interface{}{
			"storageClassName": "local-path",
			"storage":          fmt.Sprintf("%dMi", memoryMB*4),
		},
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return "", "", fmt.Errorf("marshal RabbitMQ spec: %w", err)
	}

	crd := &k8sclient.CRDObject{
		APIVersion: "rabbitmq.com/v1beta1",
		Kind:       "RabbitmqCluster",
		Metadata: k8sclient.ObjectMeta{
			Name:      instanceName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"zenith.io/database": db.ID,
				"zenith.io/user":     db.UserID,
				"zenith.io/engine":   "rabbitmq",
			},
		},
		Spec: specJSON,
	}

	if err := s.k8sClient.CreateCRD(ctx, crd); err != nil {
		return "", "", fmt.Errorf("create RabbitmqCluster CRD: %w", err)
	}

	// RabbitMQ service DNS: <name>.<namespace>.svc.cluster.local
	host = instanceName + "." + s.namespace + ".svc.cluster.local"

	return host, password, nil
}

// provisionMongoDB creates a PerconaServerMongoDB CRD via the Percona MongoDB Operator.
func (s *DatabaseService) provisionMongoDB(ctx context.Context, db *entities.UserDatabase, tier entities.PlanTier) (host, password string, err error) {
	password, err = generatePassword(16)
	if err != nil {
		return "", "", fmt.Errorf("generate password: %w", err)
	}

	instanceName := "mongo-" + db.ID[:8]

	replicas := 1
	storageMB := 1024
	switch tier {
	case entities.PlanTeam:
		replicas = 3
		storageMB = 5120
	case entities.PlanBusiness, entities.PlanEnterprise:
		replicas = 3
		storageMB = 20480
	}

	if db.MaxSizeMB > 0 && db.MaxSizeMB < storageMB {
		storageMB = db.MaxSizeMB
	}

	// Store password in a K8s Secret that the operator will reference
	mongoSecretName := instanceName + "-users"
	secretData := map[string][]byte{
		"MONGODB_DATABASE_ADMIN_PASSWORD": []byte(password),
		"MONGODB_CLUSTER_ADMIN_PASSWORD":  []byte(password),
		"MONGODB_USER_ADMIN_PASSWORD":     []byte(password),
	}
	if err := s.k8sClient.CreateSecret(ctx, s.namespace, mongoSecretName, secretData, map[string]string{
		"zenith.io/database": db.ID,
		"zenith.io/engine":   "mongodb",
	}); err != nil {
		slog.Warn("failed to create MongoDB user secret", "error", err)
	}

	spec := map[string]interface{}{
		"crVersion": "1.16.0",
		"image":     "percona/percona-server-mongodb:7.0.8-5",
		"replsets": []map[string]interface{}{
			{
				"name": "rs0",
				"size": replicas,
				"volumeSpec": map[string]interface{}{
					"persistentVolumeClaim": map[string]interface{}{
						"storageClassName": "local-path",
						"resources": map[string]interface{}{
							"requests": map[string]string{
								"storage": fmt.Sprintf("%dMi", storageMB),
							},
						},
					},
				},
				"resources": map[string]interface{}{
					"requests": map[string]string{
						"cpu":    "100m",
						"memory": "256Mi",
					},
					"limits": map[string]string{
						"cpu":    "500m",
						"memory": "512Mi",
					},
				},
			},
		},
		"secrets": map[string]interface{}{
			"users": mongoSecretName,
		},
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return "", "", fmt.Errorf("marshal MongoDB spec: %w", err)
	}

	crd := &k8sclient.CRDObject{
		APIVersion: "psmdb.percona.com/v1",
		Kind:       "PerconaServerMongoDB",
		Metadata: k8sclient.ObjectMeta{
			Name:      instanceName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"zenith.io/database": db.ID,
				"zenith.io/user":     db.UserID,
				"zenith.io/engine":   "mongodb",
			},
		},
		Spec: specJSON,
	}

	if err := s.k8sClient.CreateCRD(ctx, crd); err != nil {
		return "", "", fmt.Errorf("create PerconaServerMongoDB CRD: %w", err)
	}

	// MongoDB service DNS: <name>-rs0.<namespace>.svc.cluster.local
	host = instanceName + "-rs0." + s.namespace + ".svc.cluster.local"

	return host, password, nil
}

// resetRabbitMQPassword resets the password by updating the RabbitmqCluster CRD config.
func (s *DatabaseService) resetRabbitMQPassword(ctx context.Context, db *entities.UserDatabase, newPassword string) (string, string, error) {
	instanceName := "rmq-" + db.ID[:8]
	existing, err := s.k8sClient.GetCRD(ctx, "RabbitmqCluster", s.namespace, instanceName)
	if err != nil {
		return "", "", fmt.Errorf("get RabbitMQ CRD: %w", err)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(existing.Spec, &spec); err != nil {
		return "", "", fmt.Errorf("parse RabbitMQ spec: %w", err)
	}

	if rmq, ok := spec["rabbitmq"].(map[string]interface{}); ok {
		rmq["additionalConfig"] = fmt.Sprintf("default_user = %s\ndefault_pass = %s\ndefault_vhost = %s\n", db.DBUser, newPassword, db.DBName)
	}

	specJSON, _ := json.Marshal(spec)
	existing.Spec = specJSON
	if err := s.k8sClient.UpdateCRD(ctx, existing); err != nil {
		return "", "", fmt.Errorf("update RabbitMQ CRD: %w", err)
	}

	secretName := "db-" + db.ID[:8] + "-credentials"
	connStr := db.ConnectionString(newPassword)
	_ = s.k8sClient.DeleteSecret(ctx, s.namespace, secretName)
	s.k8sClient.CreateSecret(ctx, s.namespace, secretName, map[string][]byte{
		"RABBITMQ_URL": []byte(connStr),
		"DB_PASSWORD":  []byte(newPassword),
		"DB_HOST":      []byte(db.Host),
		"DB_NAME":      []byte(db.DBName),
		"DB_USER":      []byte(db.DBUser),
		"DB_PORT":      []byte("5672"),
	}, map[string]string{
		"zenith.io/database": db.ID,
		"zenith.io/user":     db.UserID,
	})
	if db.AppID != "" {
		s.appRepo.SetEnvVars(ctx, db.AppID, map[string]string{"RABBITMQ_URL": connStr})
	}
	return newPassword, connStr, nil
}

// resetMongoDBPassword resets the MongoDB password by updating the operator user secret.
func (s *DatabaseService) resetMongoDBPassword(ctx context.Context, db *entities.UserDatabase, newPassword string) (string, string, error) {
	instanceName := "mongo-" + db.ID[:8]
	mongoSecretName := instanceName + "-users"

	// Update the operator's user secret
	_ = s.k8sClient.DeleteSecret(ctx, s.namespace, mongoSecretName)
	s.k8sClient.CreateSecret(ctx, s.namespace, mongoSecretName, map[string][]byte{
		"MONGODB_DATABASE_ADMIN_PASSWORD": []byte(newPassword),
		"MONGODB_CLUSTER_ADMIN_PASSWORD":  []byte(newPassword),
		"MONGODB_USER_ADMIN_PASSWORD":     []byte(newPassword),
	}, map[string]string{
		"zenith.io/database": db.ID,
		"zenith.io/engine":   "mongodb",
	})

	secretName := "db-" + db.ID[:8] + "-credentials"
	connStr := db.ConnectionString(newPassword)
	_ = s.k8sClient.DeleteSecret(ctx, s.namespace, secretName)
	s.k8sClient.CreateSecret(ctx, s.namespace, secretName, map[string][]byte{
		"MONGODB_URL": []byte(connStr),
		"DB_PASSWORD": []byte(newPassword),
		"DB_HOST":     []byte(db.Host),
		"DB_NAME":     []byte(db.DBName),
		"DB_USER":     []byte(db.DBUser),
		"DB_PORT":     []byte("27017"),
	}, map[string]string{
		"zenith.io/database": db.ID,
		"zenith.io/user":     db.UserID,
	})
	if db.AppID != "" {
		s.appRepo.SetEnvVars(ctx, db.AppID, map[string]string{"MONGODB_URL": connStr})
	}
	return newPassword, connStr, nil
}

// provisionKafka creates a Strimzi Kafka CRD with a KafkaUser for authentication.
func (s *DatabaseService) provisionKafka(ctx context.Context, db *entities.UserDatabase, tier entities.PlanTier) (host, password string, err error) {
	password, err = generatePassword(16)
	if err != nil {
		return "", "", fmt.Errorf("generate password: %w", err)
	}

	instanceName := "kafka-" + db.ID[:8]

	replicas := 1
	storageMB := 1024
	switch tier {
	case entities.PlanBusiness:
		replicas = 3
		storageMB = 10240
	case entities.PlanEnterprise:
		replicas = 3
		storageMB = 51200
	}

	// Create Kafka cluster CRD (Strimzi)
	spec := map[string]interface{}{
		"kafka": map[string]interface{}{
			"version":  "3.7.0",
			"replicas": replicas,
			"listeners": []map[string]interface{}{
				{
					"name": "plain",
					"port": 9092,
					"type": "internal",
					"tls":  false,
					"authentication": map[string]interface{}{
						"type": "scram-sha-512",
					},
				},
			},
			"config": map[string]interface{}{
				"offsets.topic.replication.factor":         1,
				"transaction.state.log.replication.factor": 1,
				"transaction.state.log.min.isr":            1,
				"default.replication.factor":               1,
				"min.insync.replicas":                      1,
			},
			"storage": map[string]interface{}{
				"type": "persistent-claim",
				"size": fmt.Sprintf("%dMi", storageMB),
				"class": "local-path",
			},
			"resources": map[string]interface{}{
				"requests": map[string]string{
					"cpu":    "200m",
					"memory": "512Mi",
				},
				"limits": map[string]string{
					"cpu":    "1000m",
					"memory": "1Gi",
				},
			},
		},
		"zookeeper": map[string]interface{}{
			"replicas": replicas,
			"storage": map[string]interface{}{
				"type": "persistent-claim",
				"size": "256Mi",
				"class": "local-path",
			},
			"resources": map[string]interface{}{
				"requests": map[string]string{
					"cpu":    "100m",
					"memory": "256Mi",
				},
				"limits": map[string]string{
					"cpu":    "500m",
					"memory": "512Mi",
				},
			},
		},
		"entityOperator": map[string]interface{}{
			"topicOperator": map[string]interface{}{},
			"userOperator":  map[string]interface{}{},
		},
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return "", "", fmt.Errorf("marshal Kafka spec: %w", err)
	}

	crd := &k8sclient.CRDObject{
		APIVersion: "kafka.strimzi.io/v1beta2",
		Kind:       "Kafka",
		Metadata: k8sclient.ObjectMeta{
			Name:      instanceName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"zenith.io/database": db.ID,
				"zenith.io/user":     db.UserID,
				"zenith.io/engine":   "kafka",
			},
		},
		Spec: specJSON,
	}

	if err := s.k8sClient.CreateCRD(ctx, crd); err != nil {
		return "", "", fmt.Errorf("create Kafka CRD: %w", err)
	}

	// Create KafkaUser CRD for SCRAM-SHA-512 authentication
	userSpec := map[string]interface{}{
		"authentication": map[string]interface{}{
			"type":     "scram-sha-512",
			"password": map[string]interface{}{
				"valueFrom": map[string]interface{}{
					"secretKeyRef": map[string]interface{}{
						"name": instanceName + "-user-password",
						"key":  "password",
					},
				},
			},
		},
		"authorization": map[string]interface{}{
			"type": "simple",
			"acls": []map[string]interface{}{
				{
					"resource": map[string]interface{}{
						"type":        "topic",
						"name":        "*",
						"patternType": "literal",
					},
					"operations": []string{"All"},
				},
				{
					"resource": map[string]interface{}{
						"type":        "group",
						"name":        "*",
						"patternType": "literal",
					},
					"operations": []string{"All"},
				},
			},
		},
	}

	// Create password secret for the KafkaUser
	s.k8sClient.CreateSecret(ctx, s.namespace, instanceName+"-user-password", map[string][]byte{
		"password": []byte(password),
	}, map[string]string{
		"zenith.io/database": db.ID,
		"zenith.io/engine":   "kafka",
	})

	userSpecJSON, err := json.Marshal(userSpec)
	if err != nil {
		return "", "", fmt.Errorf("marshal KafkaUser spec: %w", err)
	}

	userCRD := &k8sclient.CRDObject{
		APIVersion: "kafka.strimzi.io/v1beta2",
		Kind:       "KafkaUser",
		Metadata: k8sclient.ObjectMeta{
			Name:      instanceName + "-user",
			Namespace: s.namespace,
			Labels: map[string]string{
				"strimzi.io/cluster":  instanceName,
				"zenith.io/database":  db.ID,
				"zenith.io/user":      db.UserID,
				"zenith.io/engine":    "kafka",
			},
		},
		Spec: userSpecJSON,
	}

	if err := s.k8sClient.CreateCRD(ctx, userCRD); err != nil {
		slog.Warn("failed to create KafkaUser", "error", err)
	}

	// Kafka bootstrap service DNS
	host = instanceName + "-kafka-bootstrap." + s.namespace + ".svc.cluster.local"

	return host, password, nil
}

// resetKafkaPassword resets the Kafka SCRAM user password by updating the password secret.
func (s *DatabaseService) resetKafkaPassword(ctx context.Context, db *entities.UserDatabase, newPassword string) (string, string, error) {
	instanceName := "kafka-" + db.ID[:8]

	// Update the password secret
	_ = s.k8sClient.DeleteSecret(ctx, s.namespace, instanceName+"-user-password")
	s.k8sClient.CreateSecret(ctx, s.namespace, instanceName+"-user-password", map[string][]byte{
		"password": []byte(newPassword),
	}, map[string]string{
		"zenith.io/database": db.ID,
		"zenith.io/engine":   "kafka",
	})

	secretName := "db-" + db.ID[:8] + "-credentials"
	connStr := db.ConnectionString(newPassword)
	_ = s.k8sClient.DeleteSecret(ctx, s.namespace, secretName)
	s.k8sClient.CreateSecret(ctx, s.namespace, secretName, map[string][]byte{
		"KAFKA_BROKERS":    []byte(connStr),
		"KAFKA_USER":       []byte(db.DBUser),
		"KAFKA_PASSWORD":   []byte(newPassword),
	}, map[string]string{
		"zenith.io/database": db.ID,
		"zenith.io/user":     db.UserID,
	})
	if db.AppID != "" {
		s.appRepo.SetEnvVars(ctx, db.AppID, map[string]string{"KAFKA_BROKERS": connStr})
	}
	return newPassword, connStr, nil
}
