package services

import (
	"fmt"
	"strings"
	"testing"
)

// --- buildRedisStatefulSet tests ---

func TestBuildRedisStatefulSet_Basic(t *testing.T) {
	sts := buildRedisStatefulSet("ms-redis", "zenith-apps", "7", "ms-redis-auth", 5)

	if sts["apiVersion"] != "apps/v1" {
		t.Error("Expected apiVersion apps/v1")
	}
	if sts["kind"] != "StatefulSet" {
		t.Error("Expected kind StatefulSet")
	}

	metadata, ok := sts["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected metadata map")
	}
	if metadata["name"] != "ms-redis" {
		t.Errorf("Expected name 'ms-redis', got '%s'", metadata["name"])
	}
	if metadata["namespace"] != "zenith-apps" {
		t.Errorf("Expected namespace 'zenith-apps', got '%s'", metadata["namespace"])
	}
}

func TestBuildRedisStatefulSet_VersionSuffix(t *testing.T) {
	// Version without dot gets "-alpine" appended
	sts := buildRedisStatefulSet("ms-redis", "ns", "7", "secret", 5)
	spec := sts["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	image := containers[0]["image"].(string)
	if !strings.Contains(image, "7-alpine") {
		t.Errorf("Expected image with 7-alpine, got '%s'", image)
	}
}

func TestBuildRedisStatefulSet_VersionWithDot(t *testing.T) {
	// Version with dot should be used as-is
	sts := buildRedisStatefulSet("ms-redis", "ns", "7.2.4", "secret", 5)
	spec := sts["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	image := containers[0]["image"].(string)
	if image != "redis:7.2.4" {
		t.Errorf("Expected 'redis:7.2.4', got '%s'", image)
	}
}

func TestBuildRedisStatefulSet_Storage(t *testing.T) {
	sts := buildRedisStatefulSet("ms-redis", "ns", "7", "secret", 20)
	spec := sts["spec"].(map[string]interface{})
	pvcs := spec["volumeClaimTemplates"].([]map[string]interface{})
	pvcSpec := pvcs[0]["spec"].(map[string]interface{})
	resources := pvcSpec["resources"].(map[string]interface{})
	requests := resources["requests"].(map[string]interface{})
	if requests["storage"] != "20Gi" {
		t.Errorf("Expected storage '20Gi', got '%s'", requests["storage"])
	}
}

// --- buildGenericService tests ---

func TestBuildGenericService(t *testing.T) {
	svc := buildGenericService("ms-redis", "zenith-apps", 6379, 6379, "redis")

	if svc["apiVersion"] != "v1" {
		t.Error("Expected apiVersion v1")
	}
	if svc["kind"] != "Service" {
		t.Error("Expected kind Service")
	}

	spec := svc["spec"].(map[string]interface{})
	if spec["clusterIP"] != "None" {
		t.Error("Expected headless service (clusterIP: None)")
	}

	ports := spec["ports"].([]map[string]interface{})
	if len(ports) != 1 {
		t.Fatalf("Expected 1 port, got %d", len(ports))
	}
	if ports[0]["port"] != 6379 {
		t.Errorf("Expected port 6379, got %v", ports[0]["port"])
	}
	if ports[0]["name"] != "redis" {
		t.Errorf("Expected port name 'redis', got '%s'", ports[0]["name"])
	}
}

// --- buildRedisService tests ---

func TestBuildRedisService(t *testing.T) {
	svc := buildRedisService("ms-redis", "zenith-apps", 6379)

	spec := svc["spec"].(map[string]interface{})
	ports := spec["ports"].([]map[string]interface{})
	if ports[0]["targetPort"] != 6379 {
		t.Errorf("Expected targetPort 6379, got %v", ports[0]["targetPort"])
	}
}

// --- buildMySQLStatefulSet tests ---

func TestBuildMySQLStatefulSet_Basic(t *testing.T) {
	sts := buildMySQLStatefulSet("ms-mysql", "zenith-apps", "8", "ms-mysql-auth", 10)

	if sts["kind"] != "StatefulSet" {
		t.Error("Expected kind StatefulSet")
	}

	metadata := sts["metadata"].(map[string]interface{})
	if metadata["name"] != "ms-mysql" {
		t.Errorf("Expected name 'ms-mysql', got '%s'", metadata["name"])
	}
}

func TestBuildMySQLStatefulSet_VersionSuffix(t *testing.T) {
	sts := buildMySQLStatefulSet("ms-mysql", "ns", "8", "secret", 5)
	spec := sts["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	image := containers[0]["image"].(string)
	if !strings.Contains(image, "8-debian") {
		t.Errorf("Expected image with 8-debian, got '%s'", image)
	}
}

func TestBuildMySQLStatefulSet_VersionWithDot(t *testing.T) {
	sts := buildMySQLStatefulSet("ms-mysql", "ns", "8.0.35", "secret", 5)
	spec := sts["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	image := containers[0]["image"].(string)
	if image != "mysql:8.0.35" {
		t.Errorf("Expected 'mysql:8.0.35', got '%s'", image)
	}
}

// --- buildMongoDBStatefulSet tests ---

func TestBuildMongoDBStatefulSet_Basic(t *testing.T) {
	sts := buildMongoDBStatefulSet("ms-mongo", "zenith-apps", "7", "ms-mongo-auth", "testdb", 10)

	if sts["kind"] != "StatefulSet" {
		t.Error("Expected kind StatefulSet")
	}

	metadata := sts["metadata"].(map[string]interface{})
	if metadata["name"] != "ms-mongo" {
		t.Errorf("Expected name 'ms-mongo', got '%s'", metadata["name"])
	}
}

func TestBuildMongoDBStatefulSet_VersionSuffix(t *testing.T) {
	sts := buildMongoDBStatefulSet("ms-mongo", "ns", "7", "secret", "db", 5)
	spec := sts["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	image := containers[0]["image"].(string)
	if !strings.Contains(image, "7.0") {
		t.Errorf("Expected image with 7.0, got '%s'", image)
	}
}

func TestBuildMongoDBStatefulSet_VersionWithDot(t *testing.T) {
	sts := buildMongoDBStatefulSet("ms-mongo", "ns", "7.0.12", "secret", "db", 5)
	spec := sts["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	image := containers[0]["image"].(string)
	if image != "mongo:7.0.12" {
		t.Errorf("Expected 'mongo:7.0.12', got '%s'", image)
	}
}

func TestBuildMongoDBStatefulSet_DbName(t *testing.T) {
	sts := buildMongoDBStatefulSet("ms-mongo", "ns", "7", "secret", "my-database", 5)
	spec := sts["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	envVars := containers[0]["env"].([]map[string]interface{})

	found := false
	for _, env := range envVars {
		if env["name"] == "MONGO_INITDB_DATABASE" {
			if env["value"] != "my-database" {
				t.Errorf("Expected MONGO_INITDB_DATABASE='my-database', got '%s'", env["value"])
			}
			found = true
		}
	}
	if !found {
		t.Error("Expected MONGO_INITDB_DATABASE env var")
	}
}

// --- buildRabbitMQStatefulSet tests ---

func TestBuildRabbitMQStatefulSet_Basic(t *testing.T) {
	sts := buildRabbitMQStatefulSet("ms-rabbit", "zenith-apps", "3", "ms-rabbit-auth", 5)

	if sts["kind"] != "StatefulSet" {
		t.Error("Expected kind StatefulSet")
	}

	metadata := sts["metadata"].(map[string]interface{})
	if metadata["name"] != "ms-rabbit" {
		t.Errorf("Expected name 'ms-rabbit', got '%s'", metadata["name"])
	}
}

func TestBuildRabbitMQStatefulSet_VersionSuffix(t *testing.T) {
	sts := buildRabbitMQStatefulSet("ms-rabbit", "ns", "3", "secret", 5)
	spec := sts["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	image := containers[0]["image"].(string)
	if !strings.Contains(image, "3-management-alpine") {
		t.Errorf("Expected image with 3-management-alpine, got '%s'", image)
	}
}

func TestBuildRabbitMQStatefulSet_VersionWithDash(t *testing.T) {
	// Version with dash should be used as-is
	sts := buildRabbitMQStatefulSet("ms-rabbit", "ns", "3.12-management", "secret", 5)
	spec := sts["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	image := containers[0]["image"].(string)
	if image != "rabbitmq:3.12-management" {
		t.Errorf("Expected 'rabbitmq:3.12-management', got '%s'", image)
	}
}

func TestBuildRabbitMQStatefulSet_Ports(t *testing.T) {
	sts := buildRabbitMQStatefulSet("ms-rabbit", "ns", "3", "secret", 5)
	spec := sts["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	ports := containers[0]["ports"].([]map[string]interface{})
	if len(ports) != 2 {
		t.Fatalf("Expected 2 ports (amqp + management), got %d", len(ports))
	}

	portNames := map[string]bool{}
	for _, p := range ports {
		portNames[fmt.Sprintf("%s", p["name"])] = true
	}
	if !portNames["amqp"] {
		t.Error("Expected amqp port")
	}
	if !portNames["management"] {
		t.Error("Expected management port")
	}
}

// --- buildCNPGCluster additional tests ---

func TestBuildCNPGCluster_DefaultPatch(t *testing.T) {
	// Version without a dot should get ".6" appended
	cluster := buildCNPGCluster("test", "ns", "16", "u", "p", "db", 5)
	spec := cluster["spec"].(map[string]interface{})
	imageName := spec["imageName"].(string)
	if !strings.Contains(imageName, "16.6") {
		t.Errorf("Expected image tag with 16.6, got '%s'", imageName)
	}
}

func TestBuildCNPGCluster_MultipleDistroSuffixes(t *testing.T) {
	suffixes := []string{"-alpine", "-bullseye", "-bookworm", "-slim", "-debian"}
	for _, suffix := range suffixes {
		cluster := buildCNPGCluster("test", "ns", "16"+suffix, "u", "p", "db", 5)
		spec := cluster["spec"].(map[string]interface{})
		imageName := spec["imageName"].(string)
		if strings.Contains(imageName, suffix) {
			t.Errorf("Expected suffix '%s' to be stripped from image, got '%s'", suffix, imageName)
		}
	}
}

func TestBuildCNPGCluster_Bootstrap(t *testing.T) {
	cluster := buildCNPGCluster("test-cluster", "ns", "16", "myuser", "mypass", "mydb", 10)
	spec := cluster["spec"].(map[string]interface{})
	bootstrap := spec["bootstrap"].(map[string]interface{})
	initdb := bootstrap["initdb"].(map[string]interface{})
	if initdb["database"] != "mydb" {
		t.Errorf("Expected database 'mydb', got '%s'", initdb["database"])
	}
	if initdb["owner"] != "myuser" {
		t.Errorf("Expected owner 'myuser', got '%s'", initdb["owner"])
	}
}

func TestBuildCNPGCluster_Storage(t *testing.T) {
	cluster := buildCNPGCluster("test", "ns", "16", "u", "p", "db", 50)
	spec := cluster["spec"].(map[string]interface{})
	storage := spec["storage"].(map[string]interface{})
	if storage["size"] != "50Gi" {
		t.Errorf("Expected storage '50Gi', got '%s'", storage["size"])
	}
}
