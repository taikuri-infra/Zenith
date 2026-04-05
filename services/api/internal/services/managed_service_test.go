package services

import (
	"strings"
	"testing"
)

// --- sanitizeK8sName tests ---

func TestSanitizeK8sName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"my-service", "my-service"},
		{"My Service", "my-service"},
		{"my_service", "my-service"},
		{"MY SERVICE!", "my-service"},
		{"service---name", "service-name"},
		{"", ""},
		{"---", ""},
		{"hello world 123", "hello-world-123"},
		{"UPPER_CASE", "upper-case"},
		{"special@#$chars", "specialchars"},
		{"a__b__c", "a-b-c"},
	}

	for _, tc := range cases {
		got := sanitizeK8sName(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeK8sName(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// --- normalizeVersion tests ---

func TestNormalizeVersion(t *testing.T) {
	cases := []struct {
		version  string
		fallback string
		expected string
	}{
		{"16", "16", "16"},
		{"", "16", "16"},
		{"latest", "16", "16"},
		{"15.3", "16", "15.3"},
		{"17-alpine", "16", "17-alpine"},
	}

	for _, tc := range cases {
		got := normalizeVersion(tc.version, tc.fallback)
		if got != tc.expected {
			t.Errorf("normalizeVersion(%q, %q) = %q, want %q", tc.version, tc.fallback, got, tc.expected)
		}
	}
}

// --- generateCredentials tests ---

func TestGenerateCredentials(t *testing.T) {
	user, pass, dbName := generateCredentials("My Database")

	if !strings.HasSuffix(user, "_user") {
		t.Errorf("Expected user to end with '_user', got '%s'", user)
	}
	if !strings.HasSuffix(dbName, "_db") {
		t.Errorf("Expected dbName to end with '_db', got '%s'", dbName)
	}
	if len(pass) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Expected 32-char hex password, got %d chars", len(pass))
	}
	if strings.Contains(user, " ") {
		t.Error("User should not contain spaces")
	}
}

func TestGenerateCredentials_Uniqueness(t *testing.T) {
	_, pass1, _ := generateCredentials("test")
	_, pass2, _ := generateCredentials("test")
	if pass1 == pass2 {
		t.Error("Two generated passwords should not be identical")
	}
}

// --- buildCNPGCluster tests ---

func TestBuildCNPGCluster(t *testing.T) {
	cluster := buildCNPGCluster("ms-mydb", "zenith-apps", "16", "myuser", "mypass", "mydb", 10)

	if cluster["apiVersion"] != "postgresql.cnpg.io/v1" {
		t.Error("Expected CNPG API version")
	}
	if cluster["kind"] != "Cluster" {
		t.Error("Expected kind Cluster")
	}

	metadata, ok := cluster["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected metadata to be a map")
	}
	if metadata["name"] != "ms-mydb" {
		t.Errorf("Expected name 'ms-mydb', got '%s'", metadata["name"])
	}
	if metadata["namespace"] != "zenith-apps" {
		t.Errorf("Expected namespace 'zenith-apps', got '%s'", metadata["namespace"])
	}
}

func TestBuildCNPGCluster_VersionSuffixStripped(t *testing.T) {
	// Version with distro suffix should be cleaned
	cluster := buildCNPGCluster("ms-test", "ns", "16-alpine", "u", "p", "db", 5)
	spec, ok := cluster["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected spec to be a map")
	}
	imageName, ok := spec["imageName"].(string)
	if !ok {
		t.Fatal("Expected imageName in spec")
	}
	if strings.Contains(imageName, "alpine") {
		t.Errorf("Expected alpine suffix to be stripped from image, got '%s'", imageName)
	}
}
