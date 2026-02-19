package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        int
	Environment string
	CORSOrigins string
	LogLevel    string

	// Kubernetes
	KubeConfig  string
	InCluster   bool

	// Auth
	JWTSecret     string
	JWTIssuer     string
	AdminEmail    string
	AdminPassword string

	// Database
	DatabaseURL string
}

func Load() *Config {
	return &Config{
		Port:        getEnvInt("PORT", 8080),
		Environment: getEnv("ENVIRONMENT", "development"),
		CORSOrigins: getEnv("CORS_ORIGINS", "*"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		KubeConfig:  getEnv("KUBECONFIG", ""),
		InCluster:   getEnvBool("IN_CLUSTER", false),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTIssuer:     getEnv("JWT_ISSUER", "zenith"),
		AdminEmail:    getEnv("ADMIN_EMAIL", ""),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
		DatabaseURL:   buildDatabaseURL(),
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
	sslmode := getEnv("DB_SSLMODE", "disable")
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
