package store

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/models"
)

func TestMemoryAdminSettings(t *testing.T) {
	s := NewMemoryAdminRepository()
	ctx := context.Background()

	settings, err := s.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings.PlatformName != "Zenith" {
		t.Errorf("Expected default platform name 'Zenith', got '%s'", settings.PlatformName)
	}

	updated, err := s.UpdateSettings(ctx, &models.PlatformSettings{
		PlatformName: "My Platform",
		BaseDomain:   "example.com",
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}
	if updated.PlatformName != "My Platform" {
		t.Errorf("Expected updated name 'My Platform', got '%s'", updated.PlatformName)
	}
	if updated.BaseDomain != "example.com" {
		t.Errorf("Expected updated domain 'example.com', got '%s'", updated.BaseDomain)
	}
	// Provider should remain default
	if updated.Provider != "Hetzner Cloud" {
		t.Errorf("Expected provider to remain 'Hetzner Cloud', got '%s'", updated.Provider)
	}
}

func TestMemoryAdminModules(t *testing.T) {
	s := NewMemoryAdminRepository()
	ctx := context.Background()

	modules, err := s.ListModules(ctx)
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	if len(modules) == 0 {
		t.Fatal("Expected non-empty default modules")
	}

	mod, err := s.GetModule(ctx, "Zenith Operator")
	if err != nil {
		t.Fatalf("GetModule failed: %v", err)
	}
	if mod.Status != "update_available" {
		t.Errorf("Expected status 'update_available', got '%s'", mod.Status)
	}

	updated, err := s.UpdateModule(ctx, "Zenith Operator")
	if err != nil {
		t.Fatalf("UpdateModule failed: %v", err)
	}
	if updated.Status != "up_to_date" {
		t.Errorf("Expected status 'up_to_date' after update, got '%s'", updated.Status)
	}
	if updated.Installed != updated.Latest {
		t.Error("Expected installed == latest after update")
	}
}

func TestMemoryAdminGetModuleNotFound(t *testing.T) {
	s := NewMemoryAdminRepository()
	_, err := s.GetModule(context.Background(), "Nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent module")
	}
}

func TestMemoryAdminUpdateModuleNotFound(t *testing.T) {
	s := NewMemoryAdminRepository()
	_, err := s.UpdateModule(context.Background(), "Nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent module update")
	}
}

func TestMemoryAdminAuditLog(t *testing.T) {
	s := NewMemoryAdminRepository()
	ctx := context.Background()

	entries, err := s.ListAuditLog(ctx, 50, 0)
	if err != nil {
		t.Fatalf("ListAuditLog failed: %v", err)
	}
	initialCount := len(entries)

	if err := s.AddAuditEntry(ctx, models.AuditEntry{
		Time:   "10:00",
		Actor:  "test",
		Action: "test action",
	}); err != nil {
		t.Fatalf("AddAuditEntry failed: %v", err)
	}

	entries, err = s.ListAuditLog(ctx, 50, 0)
	if err != nil {
		t.Fatalf("ListAuditLog failed: %v", err)
	}
	if len(entries) != initialCount+1 {
		t.Errorf("Expected %d entries after add, got %d", initialCount+1, len(entries))
	}

	if entries[0].Action != "test action" {
		t.Errorf("Expected newest entry first, got '%s'", entries[0].Action)
	}
}

func TestMemoryAdminAuditLogLimitOffset(t *testing.T) {
	s := NewMemoryAdminRepository()
	ctx := context.Background()

	entries, _ := s.ListAuditLog(ctx, 2, 0)
	if len(entries) > 2 {
		t.Errorf("Expected at most 2 entries with limit=2, got %d", len(entries))
	}

	allEntries, _ := s.ListAuditLog(ctx, 50, 0)
	if len(allEntries) > 2 {
		entries, _ = s.ListAuditLog(ctx, 50, 2)
		expectedLen := len(allEntries) - 2
		if len(entries) != expectedLen {
			t.Errorf("Expected %d entries with offset=2, got %d", expectedLen, len(entries))
		}
	}

	entries, _ = s.ListAuditLog(ctx, 50, 9999)
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with large offset, got %d", len(entries))
	}
}

func TestMemoryAdminPlatformUpdate(t *testing.T) {
	s := NewMemoryAdminRepository()
	update, err := s.GetPlatformUpdate(context.Background())
	if err != nil {
		t.Fatalf("GetPlatformUpdate failed: %v", err)
	}

	if update.Version == "" {
		t.Error("Expected non-empty version")
	}
	if update.Current == "" {
		t.Error("Expected non-empty current")
	}
	if len(update.Features) == 0 {
		t.Error("Expected non-empty features")
	}
}

func TestMemoryAdminUpdateHistory(t *testing.T) {
	s := NewMemoryAdminRepository()
	history, err := s.ListUpdateHistory(context.Background())
	if err != nil {
		t.Fatalf("ListUpdateHistory failed: %v", err)
	}

	if len(history) == 0 {
		t.Error("Expected non-empty update history")
	}
}
