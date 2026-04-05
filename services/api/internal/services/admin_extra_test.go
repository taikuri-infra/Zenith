package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
)

// --- GetTenant tests ---

func TestAdminGetTenant_Success(t *testing.T) {
	svc, k8s, _ := newTestAdminService()
	ctx := context.Background()

	// Create a Project CRD first
	spec, _ := json.Marshal(map[string]interface{}{
		"displayName": "My Tenant",
		"plan":        "pro",
	})
	k8s.CreateCRD(ctx, &k8sclient.CRDObject{
		Kind: "Project",
		Metadata: k8sclient.ObjectMeta{
			Name: "tenant-1",
		},
		Spec: spec,
	})

	tenant, err := svc.GetTenant(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("GetTenant failed: %v", err)
	}
	if tenant.Name != "My Tenant" {
		t.Errorf("Expected name 'My Tenant', got '%s'", tenant.Name)
	}
	if tenant.Plan != "pro" {
		t.Errorf("Expected plan 'pro', got '%s'", tenant.Plan)
	}
}

func TestAdminGetTenant_NotFound(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	_, err := svc.GetTenant(ctx, "nonexistent-tenant")
	if err == nil {
		t.Error("Expected error for nonexistent tenant")
	}
}

// --- SuspendTenant tests ---

func TestAdminSuspendTenant_Success(t *testing.T) {
	svc, k8s, _ := newTestAdminService()
	ctx := context.Background()

	spec, _ := json.Marshal(map[string]interface{}{
		"displayName": "Suspend Me",
		"plan":        "free",
	})
	k8s.CreateCRD(ctx, &k8sclient.CRDObject{
		Kind: "Project",
		Metadata: k8sclient.ObjectMeta{
			Name: "to-suspend",
		},
		Spec: spec,
	})

	err := svc.SuspendTenant(ctx, "to-suspend", "admin")
	if err != nil {
		t.Fatalf("SuspendTenant failed: %v", err)
	}

	// Verify the annotation was set
	tenant, err := svc.GetTenant(ctx, "to-suspend")
	if err != nil {
		t.Fatalf("GetTenant after suspend failed: %v", err)
	}
	if tenant.Status != "suspended" {
		t.Errorf("Expected status 'suspended', got '%s'", tenant.Status)
	}
}

func TestAdminSuspendTenant_NotFound(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	err := svc.SuspendTenant(ctx, "nonexistent", "admin")
	if err == nil {
		t.Error("Expected error for nonexistent tenant")
	}
}

func TestAdminSuspendTenant_AlreadySuspended(t *testing.T) {
	svc, k8s, _ := newTestAdminService()
	ctx := context.Background()

	spec, _ := json.Marshal(map[string]interface{}{})
	k8s.CreateCRD(ctx, &k8sclient.CRDObject{
		Kind: "Project",
		Metadata: k8sclient.ObjectMeta{
			Name: "already-suspended",
			Annotations: map[string]string{
				"zenith.dev/suspended": "true",
			},
		},
		Spec: spec,
	})

	// Should not error — just updates annotations again
	err := svc.SuspendTenant(ctx, "already-suspended", "admin")
	if err != nil {
		t.Fatalf("SuspendTenant on already-suspended failed: %v", err)
	}
}

// --- ListTenants with CRDs ---

func TestAdminListTenants_WithProjects(t *testing.T) {
	svc, k8s, _ := newTestAdminService()
	ctx := context.Background()

	for _, name := range []string{"proj-a", "proj-b", "proj-c"} {
		spec, _ := json.Marshal(map[string]interface{}{
			"displayName": name,
			"plan":        "free",
		})
		k8s.CreateCRD(ctx, &k8sclient.CRDObject{
			Kind: "Project",
			Metadata: k8sclient.ObjectMeta{
				Name:      name,
				Namespace: "",
			},
			Spec: spec,
		})
	}

	tenants, err := svc.ListTenants(ctx)
	if err != nil {
		t.Fatalf("ListTenants failed: %v", err)
	}
	if len(tenants) != 3 {
		t.Errorf("Expected 3 tenants, got %d", len(tenants))
	}
}
