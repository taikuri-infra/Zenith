package services

import (
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// --- envKeyForDBEngine tests ---

func TestEnvKeyForDBEngine_Postgres(t *testing.T) {
	key := envKeyForDBEngine(entities.DatabaseEnginePostgres)
	if key != "DATABASE_URL" {
		t.Errorf("Expected DATABASE_URL, got '%s'", key)
	}
}

func TestEnvKeyForDBEngine_Redis(t *testing.T) {
	key := envKeyForDBEngine(entities.DatabaseEngineRedis)
	if key != "REDIS_URL" {
		t.Errorf("Expected REDIS_URL, got '%s'", key)
	}
}

func TestEnvKeyForDBEngine_MySQL(t *testing.T) {
	key := envKeyForDBEngine(entities.DatabaseEngineMySQL)
	if key != "MYSQL_URL" {
		t.Errorf("Expected MYSQL_URL, got '%s'", key)
	}
}

func TestEnvKeyForDBEngine_MongoDB(t *testing.T) {
	key := envKeyForDBEngine(entities.DatabaseEngineMongoDB)
	if key != "MONGODB_URL" {
		t.Errorf("Expected MONGODB_URL, got '%s'", key)
	}
}

func TestEnvKeyForDBEngine_RabbitMQ(t *testing.T) {
	key := envKeyForDBEngine(entities.DatabaseEngineRabbitMQ)
	if key != "RABBITMQ_URL" {
		t.Errorf("Expected RABBITMQ_URL, got '%s'", key)
	}
}

func TestEnvKeyForDBEngine_Kafka(t *testing.T) {
	key := envKeyForDBEngine(entities.DatabaseEngineKafka)
	if key != "KAFKA_BROKERS" {
		t.Errorf("Expected KAFKA_BROKERS, got '%s'", key)
	}
}

func TestEnvKeyForDBEngine_Unknown(t *testing.T) {
	key := envKeyForDBEngine("unknown")
	if key != "DATABASE_URL" {
		t.Errorf("Expected DATABASE_URL for unknown engine, got '%s'", key)
	}
}

// --- portForEngine tests ---

func TestPortForEngine_Postgres(t *testing.T) {
	port := portForEngine(entities.DatabaseEnginePostgres)
	if port != "5432" {
		t.Errorf("Expected 5432, got '%s'", port)
	}
}

func TestPortForEngine_Redis(t *testing.T) {
	port := portForEngine(entities.DatabaseEngineRedis)
	if port != "6379" {
		t.Errorf("Expected 6379, got '%s'", port)
	}
}

func TestPortForEngine_MySQL(t *testing.T) {
	port := portForEngine(entities.DatabaseEngineMySQL)
	if port != "3306" {
		t.Errorf("Expected 3306, got '%s'", port)
	}
}

func TestPortForEngine_MongoDB(t *testing.T) {
	port := portForEngine(entities.DatabaseEngineMongoDB)
	if port != "27017" {
		t.Errorf("Expected 27017, got '%s'", port)
	}
}

func TestPortForEngine_RabbitMQ(t *testing.T) {
	port := portForEngine(entities.DatabaseEngineRabbitMQ)
	if port != "5672" {
		t.Errorf("Expected 5672, got '%s'", port)
	}
}

func TestPortForEngine_Kafka(t *testing.T) {
	port := portForEngine(entities.DatabaseEngineKafka)
	if port != "9092" {
		t.Errorf("Expected 9092, got '%s'", port)
	}
}

func TestPortForEngine_Unknown(t *testing.T) {
	port := portForEngine("unknown")
	if port != "5432" {
		t.Errorf("Expected 5432 for unknown engine, got '%s'", port)
	}
}
