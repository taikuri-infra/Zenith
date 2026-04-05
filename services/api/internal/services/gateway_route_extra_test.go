package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// --- CreateRoute with group ---

func TestCreateRoute_WithGroup(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name: "group-target", UserID: "user-1", ProjectID: "proj-1", DeploySource: "image", ImageURL: "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "proj-1", "grouped-gw")

	// Create a group via the service
	group, err := svc.CreateGroup(ctx, gw.ID, &entities.GatewayGroup{
		Name:  "api-group",
		AppID: app.ID,
	})
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	// Create a route within the group
	route := &entities.GatewayRoute{
		Path:    "/api/users",
		GroupID: group.ID,
		Methods: []string{"GET"},
	}
	created, err := svc.CreateRoute(ctx, gw.ID, route)
	if err != nil {
		t.Fatalf("CreateRoute with group failed: %v", err)
	}

	// Route should have empty AppID (inherited from group)
	if created.AppID != "" {
		t.Errorf("Expected empty AppID for group route, got '%s'", created.AppID)
	}
	if created.GroupID != group.ID {
		t.Errorf("Expected GroupID '%s', got '%s'", group.ID, created.GroupID)
	}
}

func TestCreateRoute_GroupNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "proj-1", "group-nf-gw")

	route := &entities.GatewayRoute{
		Path:    "/test",
		GroupID: "nonexistent-group",
	}
	_, err := svc.CreateRoute(ctx, gw.ID, route)
	if err == nil {
		t.Error("Expected error for nonexistent group")
	}
}

func TestCreateRoute_GroupWrongGateway(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name: "gw-cross-app", UserID: "user-1", ProjectID: "proj-1", DeploySource: "image", ImageURL: "nginx:latest",
	})

	gw1, _ := svc.CreateGateway(ctx, "user-1", "proj-1", "gw1-for-group")
	gw2, _ := svc.CreateGateway(ctx, "user-1", "proj-1", "gw2-for-group")

	// Create group for gw1
	group, _ := svc.CreateGroup(ctx, gw1.ID, &entities.GatewayGroup{
		Name:  "gw1-group",
		AppID: app.ID,
	})

	// Try to create route in gw2 with gw1's group
	route := &entities.GatewayRoute{
		Path:    "/test",
		GroupID: group.ID,
	}
	_, err := svc.CreateRoute(ctx, gw2.ID, route)
	if err == nil {
		t.Error("Expected error when group doesn't belong to gateway")
	}
}

func TestCreateRoute_AppNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "proj-1", "app-nf-gw")

	route := &entities.GatewayRoute{
		Path:  "/test",
		AppID: "nonexistent-app",
	}
	_, err := svc.CreateRoute(ctx, gw.ID, route)
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

func TestCreateRoute_AppWrongUser(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	// Create app owned by user-2
	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name: "other-user-app", UserID: "user-2", ProjectID: "proj-2", DeploySource: "image", ImageURL: "nginx:latest",
	})

	// Create gateway owned by user-1
	gw, _ := svc.CreateGateway(ctx, "user-1", "proj-1", "wrong-user-gw")

	route := &entities.GatewayRoute{
		Path:  "/test",
		AppID: app.ID,
	}
	_, err := svc.CreateRoute(ctx, gw.ID, route)
	if err == nil {
		t.Error("Expected error when app belongs to different user")
	}
}

func TestCreateRoute_AuthPoolNotConfigured(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name: "auth-app", UserID: "user-1", ProjectID: "proj-1", DeploySource: "image", ImageURL: "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "proj-1", "auth-gw")

	route := &entities.GatewayRoute{
		Path:       "/test",
		AppID:      app.ID,
		AuthPoolID: "some-pool",
	}
	_, err := svc.CreateRoute(ctx, gw.ID, route)
	if err == nil {
		t.Error("Expected error when auth pool repo not configured")
	}
}

// --- GetGatewayTimeSeries not found ---

func TestGetGatewayTimeSeries_GatewayNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	_, err := svc.GetGatewayTimeSeries(ctx, "nonexistent", "requests", "1h")
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

// --- GetGatewayAnalytics not found ---

func TestGetGatewayAnalytics_GatewayNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	_, err := svc.GetGatewayAnalytics(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

// --- DeleteGatewayDomain not found ---

func TestDeleteGatewayDomain_DomainNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "proj-1", "del-domain-gw")

	err := svc.DeleteGatewayDomain(ctx, gw.ID, "nonexistent-domain")
	if err == nil {
		t.Error("Expected error for nonexistent domain")
	}
}
