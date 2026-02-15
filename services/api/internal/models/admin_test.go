package models

import (
	"encoding/json"
	"testing"
)

func TestDashboardStatsJSON(t *testing.T) {
	stats := DashboardStats{
		ClusterCount:     3,
		AllHealthy:       true,
		TenantCount:      5,
		ActiveToday:      4,
		MonthlyCost:      "EUR 47.60",
		CostProvider:     "Hetzner Cloud",
		UpdatesAvailable: 2,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal DashboardStats: %v", err)
	}

	var decoded DashboardStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal DashboardStats: %v", err)
	}

	if decoded.ClusterCount != 3 {
		t.Errorf("Expected clusterCount 3, got %d", decoded.ClusterCount)
	}
	if !decoded.AllHealthy {
		t.Error("Expected allHealthy true")
	}
	if decoded.MonthlyCost != "EUR 47.60" {
		t.Errorf("Expected monthlyCost 'EUR 47.60', got '%s'", decoded.MonthlyCost)
	}
}

func TestClusterJSON(t *testing.T) {
	cluster := Cluster{
		Name:       "test-cluster",
		K8sVersion: "v1.30.2",
		Nodes:      8,
		Region:     "fsn1",
		Type:       "shared",
		CPUPercent: 62,
		RAMPercent: 58,
		Pods:       ResourcePair{Used: 234, Total: 500},
		PVCs:       ResourcePair{Used: 89, Total: 200},
		Status:     "healthy",
	}

	data, err := json.Marshal(cluster)
	if err != nil {
		t.Fatalf("Failed to marshal Cluster: %v", err)
	}

	var decoded Cluster
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Cluster: %v", err)
	}

	if decoded.Name != "test-cluster" {
		t.Errorf("Expected name 'test-cluster', got '%s'", decoded.Name)
	}
	if decoded.Pods.Used != 234 {
		t.Errorf("Expected pods used 234, got %d", decoded.Pods.Used)
	}
	if decoded.Pods.Total != 500 {
		t.Errorf("Expected pods total 500, got %d", decoded.Pods.Total)
	}
}

func TestClusterJSONOmitsEmptyOptionals(t *testing.T) {
	cluster := Cluster{
		Name:       "minimal",
		K8sVersion: "v1.30.2",
		Status:     "healthy",
	}

	data, err := json.Marshal(cluster)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	// tenant and upgradeAvailable should be omitted when empty
	if _, ok := raw["tenant"]; ok {
		val, _ := raw["tenant"].(string)
		if val != "" {
			t.Error("Expected tenant to be empty or omitted")
		}
	}
	if _, ok := raw["upgradeAvailable"]; ok {
		val, _ := raw["upgradeAvailable"].(string)
		if val != "" {
			t.Error("Expected upgradeAvailable to be empty or omitted")
		}
	}
}

func TestTenantJSON(t *testing.T) {
	tenant := Tenant{
		Name:      "my-startup",
		Plan:      "starter",
		Apps:      12,
		Databases: 3,
		CPUUsed:   "2.4",
		CPULimit:  "4",
		RAMUsed:   "3.1",
		RAMLimit:  "4",
		Status:    "active",
	}

	data, err := json.Marshal(tenant)
	if err != nil {
		t.Fatalf("Failed to marshal Tenant: %v", err)
	}

	var decoded Tenant
	json.Unmarshal(data, &decoded)

	if decoded.Name != "my-startup" {
		t.Errorf("Expected name 'my-startup', got '%s'", decoded.Name)
	}
	if decoded.Plan != "starter" {
		t.Errorf("Expected plan 'starter', got '%s'", decoded.Plan)
	}
}

func TestModuleJSON(t *testing.T) {
	mod := Module{
		Name:        "CloudNativePG",
		Installed:   "v1.22.1",
		Latest:      "v1.23.0",
		Status:      "update_available",
		Description: "PostgreSQL operator",
	}

	data, _ := json.Marshal(mod)
	var decoded Module
	json.Unmarshal(data, &decoded)

	if decoded.Name != "CloudNativePG" {
		t.Errorf("Expected name 'CloudNativePG', got '%s'", decoded.Name)
	}
	if decoded.Status != "update_available" {
		t.Errorf("Expected status 'update_available', got '%s'", decoded.Status)
	}
}

func TestPlatformSettingsJSON(t *testing.T) {
	settings := PlatformSettings{
		PlatformName:  "Zenith",
		BaseDomain:    "freezenith.com",
		Provider:      "Hetzner Cloud",
		DefaultRegion: "fsn1",
		RegionLabel:   "Falkenstein",
		AutoBackups:   true,
		RetentionDays: 30,
	}

	data, _ := json.Marshal(settings)
	var decoded PlatformSettings
	json.Unmarshal(data, &decoded)

	if decoded.PlatformName != "Zenith" {
		t.Errorf("Expected platformName 'Zenith', got '%s'", decoded.PlatformName)
	}
	if !decoded.AutoBackups {
		t.Error("Expected autoBackups true")
	}
	if decoded.RetentionDays != 30 {
		t.Errorf("Expected retentionDays 30, got %d", decoded.RetentionDays)
	}
}

func TestInfraOverviewJSON(t *testing.T) {
	infra := InfraOverview{
		Servers:       10,
		Volumes:       5,
		VolumeSize:    "100 GB",
		LoadBalancers: 2,
		LBPublic:      2,
		LBInternal:    0,
		MonthlyCost:   "EUR 100.00",
		Resources: []InfraNode{
			{Name: "pool-1", Type: "CX22", Count: 4, Cluster: "shared", MonthlyCost: "EUR 20.00"},
		},
	}

	data, _ := json.Marshal(infra)
	var decoded InfraOverview
	json.Unmarshal(data, &decoded)

	if decoded.Servers != 10 {
		t.Errorf("Expected 10 servers, got %d", decoded.Servers)
	}
	if len(decoded.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(decoded.Resources))
	}
}

func TestPlatformStateJSON(t *testing.T) {
	state := PlatformState{
		PlatformVersion:       "v1.2.1",
		UpdateAvailable:       "v1.3.0",
		InstalledDate:         "2026-01-15",
		InstalledDaysAgo:      31,
		ManagementK8sVersion:  "v1.30.2",
		ManagementK8sUpToDate: true,
		Domain:                "freezenith.com",
		WildcardTLS:           true,
	}

	data, _ := json.Marshal(state)
	var decoded PlatformState
	json.Unmarshal(data, &decoded)

	if decoded.PlatformVersion != "v1.2.1" {
		t.Errorf("Expected version 'v1.2.1', got '%s'", decoded.PlatformVersion)
	}
	if !decoded.WildcardTLS {
		t.Error("Expected wildcardTls true")
	}
}

func TestPlatformUpdateJSON(t *testing.T) {
	update := PlatformUpdate{
		Version:         "v1.3.0",
		Current:         "v1.2.1",
		ReleasedAt:      "February 10, 2026",
		Features:        []string{"MongoDB support", "GitOps mode"},
		BreakingChanges: false,
	}

	data, _ := json.Marshal(update)
	var decoded PlatformUpdate
	json.Unmarshal(data, &decoded)

	if decoded.Version != "v1.3.0" {
		t.Errorf("Expected version 'v1.3.0', got '%s'", decoded.Version)
	}
	if len(decoded.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(decoded.Features))
	}
}

func TestCreateClusterInputJSON(t *testing.T) {
	input := CreateClusterInput{
		Name:       "new-cluster",
		Region:     "fsn1",
		Type:       "dedicated",
		Tenant:     "acme",
		Nodes:      4,
		K8sVersion: "v1.30.2",
	}

	data, _ := json.Marshal(input)
	var decoded CreateClusterInput
	json.Unmarshal(data, &decoded)

	if decoded.Name != "new-cluster" {
		t.Errorf("Expected name 'new-cluster', got '%s'", decoded.Name)
	}
	if decoded.Tenant != "acme" {
		t.Errorf("Expected tenant 'acme', got '%s'", decoded.Tenant)
	}
}
