package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// --- CreateGroup tests ---

func TestCreateGroup_Success(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "group-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Group GW")

	group := &entities.GatewayGroup{
		Name:  "api-group",
		AppID: app.ID,
	}
	created, err := svc.CreateGroup(ctx, gw.ID, group)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if created == nil {
		t.Fatal("Expected non-nil group")
	}
	if created.Name != "api-group" {
		t.Errorf("Expected name 'api-group', got '%s'", created.Name)
	}
	if created.AppSubdomain == "" {
		t.Error("Expected non-empty app subdomain")
	}
	if created.GatewayID != gw.ID {
		t.Errorf("Expected gateway ID '%s', got '%s'", gw.ID, created.GatewayID)
	}
}

func TestCreateGroup_GatewayNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	group := &entities.GatewayGroup{Name: "test", AppID: "app-1"}
	_, err := svc.CreateGroup(ctx, "nonexistent", group)
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

func TestCreateGroup_AppNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	group := &entities.GatewayGroup{Name: "test", AppID: "nonexistent-app"}
	_, err := svc.CreateGroup(ctx, gw.ID, group)
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

func TestCreateGroup_AppWrongUser(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "other-app",
		UserID:       "user-2",
		ProjectID:    "project-2",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	group := &entities.GatewayGroup{Name: "test", AppID: app.ID}
	_, err := svc.CreateGroup(ctx, gw.ID, group)
	if err == nil {
		t.Error("Expected error when app belongs to different user")
	}
}

func TestCreateGroup_InvalidPlugins(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "plugin-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	group := &entities.GatewayGroup{
		Name:  "bad-plugins",
		AppID: app.ID,
		Plugins: []entities.GatewayRoutePlugin{
			{Name: "not-allowed-plugin", Enable: true},
		},
	}
	_, err := svc.CreateGroup(ctx, gw.ID, group)
	if err == nil {
		t.Error("Expected error for invalid plugin")
	}
}

// --- UpdateGroup tests ---

func TestUpdateGroup_Success(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "upd-group-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	created, _ := svc.CreateGroup(ctx, gw.ID, &entities.GatewayGroup{
		Name:  "original",
		AppID: app.ID,
	})

	updated, err := svc.UpdateGroup(ctx, gw.ID, created.ID, &entities.GatewayGroup{
		Name: "renamed",
	})
	if err != nil {
		t.Fatalf("UpdateGroup failed: %v", err)
	}
	if updated.Name != "renamed" {
		t.Errorf("Expected name 'renamed', got '%s'", updated.Name)
	}
}

func TestUpdateGroup_GatewayNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	_, err := svc.UpdateGroup(ctx, "nonexistent", "group-1", &entities.GatewayGroup{Name: "test"})
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

func TestUpdateGroup_GroupNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	_, err := svc.UpdateGroup(ctx, gw.ID, "nonexistent-group", &entities.GatewayGroup{Name: "test"})
	if err == nil {
		t.Error("Expected error for nonexistent group")
	}
}

func TestUpdateGroup_WrongGateway(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "wg-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw1, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW1")
	gw2, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW2 Different")
	group, _ := svc.CreateGroup(ctx, gw1.ID, &entities.GatewayGroup{Name: "grp", AppID: app.ID})

	_, err := svc.UpdateGroup(ctx, gw2.ID, group.ID, &entities.GatewayGroup{Name: "hijack"})
	if err == nil {
		t.Error("Expected error when group does not belong to gateway")
	}
}

func TestUpdateGroup_ChangeApp(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app1, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "app-one",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})
	app2, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "app-two",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	group, _ := svc.CreateGroup(ctx, gw.ID, &entities.GatewayGroup{Name: "grp", AppID: app1.ID})

	updated, err := svc.UpdateGroup(ctx, gw.ID, group.ID, &entities.GatewayGroup{
		AppID: app2.ID,
	})
	if err != nil {
		t.Fatalf("UpdateGroup with new app failed: %v", err)
	}
	if updated.AppID != app2.ID {
		t.Errorf("Expected app ID '%s', got '%s'", app2.ID, updated.AppID)
	}
}

func TestUpdateGroup_InvalidPlugins(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "plg-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	group, _ := svc.CreateGroup(ctx, gw.ID, &entities.GatewayGroup{Name: "grp", AppID: app.ID})

	_, err := svc.UpdateGroup(ctx, gw.ID, group.ID, &entities.GatewayGroup{
		Plugins: []entities.GatewayRoutePlugin{
			{Name: "forbidden-plugin", Enable: true},
		},
	})
	if err == nil {
		t.Error("Expected error for invalid plugin")
	}
}

// --- DeleteGroup tests ---

func TestDeleteGroup_Success(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "del-grp-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	group, _ := svc.CreateGroup(ctx, gw.ID, &entities.GatewayGroup{Name: "to-delete", AppID: app.ID})

	err := svc.DeleteGroup(ctx, gw.ID, group.ID)
	if err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}

	// Verify group is gone
	groups, _ := svc.ListGroups(ctx, gw.ID)
	for _, g := range groups {
		if g.ID == group.ID {
			t.Error("Group should have been deleted")
		}
	}
}

func TestDeleteGroup_NotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	err := svc.DeleteGroup(ctx, gw.ID, "nonexistent-group")
	if err == nil {
		t.Error("Expected error for nonexistent group")
	}
}

func TestDeleteGroup_WrongGateway(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "wg-del-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw1, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW1")
	gw2, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW2 Diff")
	group, _ := svc.CreateGroup(ctx, gw1.ID, &entities.GatewayGroup{Name: "grp", AppID: app.ID})

	err := svc.DeleteGroup(ctx, gw2.ID, group.ID)
	if err == nil {
		t.Error("Expected error when group does not belong to gateway")
	}
}

// --- UpdateRoute tests ---

func TestUpdateRoute_Success(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "upd-route-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "UpdRoute GW")
	route, _ := svc.CreateRoute(ctx, gw.ID, &entities.GatewayRoute{Path: "/old", AppID: app.ID})

	updated, err := svc.UpdateRoute(ctx, gw.ID, route.ID, &entities.GatewayRoute{
		Path:    "/new",
		Methods: []string{"POST"},
	})
	if err != nil {
		t.Fatalf("UpdateRoute failed: %v", err)
	}
	if updated.Path != "/new" {
		t.Errorf("Expected path '/new', got '%s'", updated.Path)
	}
}

func TestUpdateRoute_GatewayNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	_, err := svc.UpdateRoute(ctx, "nonexistent", "route-1", &entities.GatewayRoute{Path: "/test"})
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

func TestUpdateRoute_RouteNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	_, err := svc.UpdateRoute(ctx, gw.ID, "nonexistent-route", &entities.GatewayRoute{Path: "/test"})
	if err == nil {
		t.Error("Expected error for nonexistent route")
	}
}

func TestUpdateRoute_WrongGateway(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "wr-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw1, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW1")
	gw2, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW2 Other")
	route, _ := svc.CreateRoute(ctx, gw1.ID, &entities.GatewayRoute{Path: "/test", AppID: app.ID})

	_, err := svc.UpdateRoute(ctx, gw2.ID, route.ID, &entities.GatewayRoute{Path: "/hijack"})
	if err == nil {
		t.Error("Expected error when route does not belong to gateway")
	}
}

func TestUpdateRoute_InvalidPlugins(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "plg-route-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	route, _ := svc.CreateRoute(ctx, gw.ID, &entities.GatewayRoute{Path: "/test", AppID: app.ID})

	_, err := svc.UpdateRoute(ctx, gw.ID, route.ID, &entities.GatewayRoute{
		Path: "/test",
		Plugins: []entities.GatewayRoutePlugin{
			{Name: "not-allowed", Enable: true},
		},
	})
	if err == nil {
		t.Error("Expected error for invalid plugin")
	}
}

func TestUpdateRoute_WithGroupAssignment(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "group-route-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	group, _ := svc.CreateGroup(ctx, gw.ID, &entities.GatewayGroup{Name: "grp", AppID: app.ID})
	route, _ := svc.CreateRoute(ctx, gw.ID, &entities.GatewayRoute{Path: "/test", AppID: app.ID})

	// Assign route to group
	updated, err := svc.UpdateRoute(ctx, gw.ID, route.ID, &entities.GatewayRoute{
		Path:    "/test",
		GroupID: group.ID,
	})
	if err != nil {
		t.Fatalf("UpdateRoute with group assignment failed: %v", err)
	}
	if updated.GroupID != group.ID {
		t.Errorf("Expected group ID '%s', got '%s'", group.ID, updated.GroupID)
	}
	// AppID should be cleared when assigned to group
	if updated.AppID != "" {
		t.Errorf("Expected empty app ID when assigned to group, got '%s'", updated.AppID)
	}
}

func TestUpdateRoute_GroupFromWrongGateway(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "multi-gw-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw1, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW1")
	gw2, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW2 Diff")
	group, _ := svc.CreateGroup(ctx, gw1.ID, &entities.GatewayGroup{Name: "grp", AppID: app.ID})
	route, _ := svc.CreateRoute(ctx, gw2.ID, &entities.GatewayRoute{Path: "/test", AppID: app.ID})

	// Try to assign route to a group from a different gateway
	_, err := svc.UpdateRoute(ctx, gw2.ID, route.ID, &entities.GatewayRoute{
		Path:    "/test",
		GroupID: group.ID,
	})
	if err == nil {
		t.Error("Expected error when group does not belong to same gateway")
	}
}

// --- AutoCreateRoute tests ---

func TestAutoCreateRoute_Public(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "public-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Auto GW")

	err := svc.AutoCreateRoute(ctx, gw, app)
	if err != nil {
		t.Fatalf("AutoCreateRoute failed: %v", err)
	}
}

func TestAutoCreateRoute_Protected(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "protected-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
		Exposure:     entities.ExposureProtected,
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Auto GW2")

	err := svc.AutoCreateRoute(ctx, gw, app)
	if err != nil {
		t.Fatalf("AutoCreateRoute (protected) failed: %v", err)
	}
}
