package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        int
	Environment string
	Mode        string // ZENITH_MODE: "standalone" (default) or "saas"
	CORSOrigins string
	LogLevel    string

	// Kubernetes
	K8sMode    string // K8S_MODE: "memory" (default) or "real"
	KubeConfig string
	InCluster  bool

	// Auth
	JWTSecret          string
	JWTIssuer          string
	AdminEmail         string
	AdminPassword      string
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string

	// Email (Resend)
	ResendAPIKey string
	AppURL       string
	EmailFrom    string

	// Internal
	InternalSecret string

	// Deploy
	BaseDomain    string
	GatewayDomain string // subdomain for API gateways (e.g. "gw.stage.freezenith.com")
	Registry      string // container registry for user images

	// Harbor Registry API
	HarborURL      string // e.g. "https://hub.stage.freezenith.com"
	HarborUser     string // robot account username
	HarborPassword string // robot account token

	// Deploy concurrency
	MaxConcurrentDeploys int

	// Secrets encryption
	SecretsKey string // SECRETS_ENCRYPTION_KEY: 64-char hex (32 bytes)

	// OpenTelemetry (opt-in: only active when OTELEndpoint is set)
	OTELEndpoint   string  // OTEL_EXPORTER_OTLP_ENDPOINT
	OTELInsecure   bool    // OTEL_INSECURE (default: true for in-cluster)
	OTELSampleRate float64 // OTEL_SAMPLE_RATE (0.0–1.0, default: 1.0)

	// Database
	DatabaseURL string

	// Stripe Billing (Phase 6)
	StripeBillingEnabled bool
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripeProPriceID      string
	StripeTeamPriceID     string
	StripeBusinessPriceID string

	// Temporal (customer provisioning workflows)
	TemporalEnabled   bool
	TemporalHost      string
	TemporalNamespace string

	// Keycloak Admin (realm provisioning)
	KeycloakURL           string
	KeycloakAdminUser     string
	KeycloakAdminPassword string

	// S3 / Hetzner Object Storage (tenant bucket provisioning)
	S3Endpoint       string
	S3AccessKey      string
	S3SecretKey      string
	S3Region         string
	S3PlatformBucket string // shared bucket for all user storage (prefix-isolated)

	// CNPG Admin DSN (for CREATE DATABASE in shared cluster)
	CNPGAdminDSN string

	// Monitoring (Prometheus + Loki + Grafana + Tempo)
	PrometheusURL string
	LokiURL       string
	GrafanaURL    string
	TempoURL      string

	// Redis (rate limiting + token blacklist)
	RedisURL string // REDIS_URL: "redis://host:6379/0" (empty = in-memory fallback)

	// NATS JetStream (event bus)
	NATSEnabled    bool
	NATSServers    string // comma-separated NATS URLs
	NATSStreamName string // JetStream stream name

	// Hetzner Autoscaler (Phase 5)
	HetznerToken        string
	AutoscalerEnabled   bool
	AutoscalerMinNodes  int
	AutoscalerMaxNodes  int
	AutoscalerInterval  int // seconds
	HetznerServerType   string
	HetznerLocation     string
	K3sToken            string
}

func Load() *Config {
	mode := getEnv("ZENITH_MODE", "standalone")
	// Never default to wildcard CORS — require explicit origin allowlist.
	// In standalone mode use localhost; in saas mode require CORS_ORIGINS env var.
	corsDefault := "http://localhost:3000"

	return &Config{
		Port:        getEnvInt("PORT", 8080),
		Environment: getEnv("ENVIRONMENT", "development"),
		Mode:        mode,
		CORSOrigins: getEnv("CORS_ORIGINS", corsDefault),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		KubeConfig:  getEnv("KUBECONFIG", ""),
		K8sMode:    getEnv("K8S_MODE", "memory"),
		InCluster:   getEnvBool("IN_CLUSTER", false),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTIssuer:     getEnv("JWT_ISSUER", "zenith"),
		AdminEmail:     getEnv("ADMIN_EMAIL", ""),
		AdminPassword:  getEnv("ADMIN_PASSWORD", ""),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		ResendAPIKey:   getEnv("RESEND_API_KEY", ""),
		AppURL:         getEnv("APP_URL", ""),
		EmailFrom:      getEnv("EMAIL_FROM", "Zenith <noreply@freezenith.com>"),
		InternalSecret: getEnv("INTERNAL_SECRET", ""),
		BaseDomain:    getEnv("BASE_DOMAIN", "freezenith.com"),
		GatewayDomain: getEnv("GATEWAY_DOMAIN", "gw."+getEnv("BASE_DOMAIN", "freezenith.com")),
		Registry:       getEnv("REGISTRY", "registry.freezenith.com"),
		HarborURL:      getEnv("HARBOR_URL", ""),
		HarborUser:     getEnv("HARBOR_USER", ""),
		HarborPassword: getEnv("HARBOR_PASSWORD", ""),
		MaxConcurrentDeploys: getEnvInt("MAX_CONCURRENT_DEPLOYS", 5),
		SecretsKey:          getEnv("SECRETS_ENCRYPTION_KEY", ""),
		OTELEndpoint:        getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		OTELInsecure:        getEnvBool("OTEL_INSECURE", true),
		OTELSampleRate:      getEnvFloat("OTEL_SAMPLE_RATE", 1.0),
		DatabaseURL:         buildDatabaseURL(),
		StripeBillingEnabled: getEnvBool("STRIPE_BILLING_ENABLED", false),
		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripeProPriceID:      getEnv("STRIPE_PRO_PRICE_ID", ""),
		StripeTeamPriceID:    getEnv("STRIPE_TEAM_PRICE_ID", ""),
		StripeBusinessPriceID: getEnv("STRIPE_BUSINESS_PRICE_ID", ""),
		TemporalEnabled:       getEnvBool("TEMPORAL_ENABLED", false),
		TemporalHost:          getEnv("TEMPORAL_HOST", "temporal.temporal.svc.cluster.local:7233"),
		TemporalNamespace:     getEnv("TEMPORAL_NAMESPACE", "default"),
		KeycloakURL:           getEnv("KEYCLOAK_URL", ""),
		KeycloakAdminUser:     getEnv("KEYCLOAK_ADMIN_USER", "admin"),
		KeycloakAdminPassword: getEnv("KEYCLOAK_ADMIN_PASSWORD", ""),
		S3Endpoint:            getEnv("S3_ENDPOINT", ""),
		S3AccessKey:           getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:           getEnv("S3_SECRET_KEY", ""),
		S3Region:              getEnv("S3_REGION", "fsn1"),
		S3PlatformBucket:      getEnv("S3_PLATFORM_BUCKET", "zenith-platform-storage"),
		CNPGAdminDSN:          getEnv("CNPG_ADMIN_DSN", ""),
		PrometheusURL:         getEnv("PROMETHEUS_URL", "http://kube-prometheus-stack-prometheus.monitoring.svc.cluster.local:9090"),
		LokiURL:              getEnv("LOKI_URL", "http://loki.monitoring.svc.cluster.local:3100"),
		GrafanaURL:           getEnv("GRAFANA_URL", "http://kube-prometheus-stack-grafana.monitoring.svc.cluster.local:80"),
		TempoURL:             getEnv("TEMPO_URL", "http://tempo.monitoring.svc.cluster.local:3100"),
		RedisURL:              getEnv("REDIS_URL", ""),
		NATSEnabled:           getEnvBool("NATS_ENABLED", false),
		NATSServers:           getEnv("NATS_SERVERS", "nats://nats.nats.svc.cluster.local:4222"),
		NATSStreamName:        getEnv("NATS_STREAM_NAME", "zenith_events"),
		HetznerToken:          getEnv("HCLOUD_TOKEN", ""),
		AutoscalerEnabled:   getEnvBool("AUTOSCALER_ENABLED", false),
		AutoscalerMinNodes:  getEnvInt("AUTOSCALER_MIN_NODES", 2),
		AutoscalerMaxNodes:  getEnvInt("AUTOSCALER_MAX_NODES", 10),
		AutoscalerInterval:  getEnvInt("AUTOSCALER_INTERVAL", 60),
		HetznerServerType:   getEnv("HETZNER_SERVER_TYPE", "cpx31"),
		HetznerLocation:     getEnv("HETZNER_LOCATION", "fsn1"),
		K3sToken:            getEnv("K3S_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

// buildDatabaseURL returns DATABASE_URL if set, otherwise assembles one from
// individual DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME env vars.
// Returns "" when no database is configured (falls back to in-memory stores).
func buildDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	host := os.Getenv("DB_HOST")
	if host == "" {
		return ""
	}
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "zenith")
	pass := os.Getenv("DB_PASSWORD")
	name := getEnv("DB_NAME", "zenith")
	sslmode := getEnv("DB_SSLMODE", "require")
	return "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + name + "?sslmode=" + sslmode
}

func getEnvBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return fallback
}
