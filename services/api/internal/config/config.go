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
	JWTSecret   string
	JWTIssuer   string
}

func Load() *Config {
	return &Config{
		Port:        getEnvInt("PORT", 8080),
		Environment: getEnv("ENVIRONMENT", "development"),
		CORSOrigins: getEnv("CORS_ORIGINS", "*"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		KubeConfig:  getEnv("KUBECONFIG", ""),
		InCluster:   getEnvBool("IN_CLUSTER", false),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		JWTIssuer:   getEnv("JWT_ISSUER", "zenith"),
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

func getEnvBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return fallback
}
