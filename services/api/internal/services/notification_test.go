package services

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func newTestNotificationService() (*NotificationService, *memory.MemoryNotificationRepository) {
	eventBus := memory.NewMemoryEventBus()
	userRepo := memory.NewMemoryUserRepository()
	notifRepo := memory.NewMemoryNotificationRepository()

	svc := NewNotificationService(eventBus, nil, userRepo, "https://app.example.com")
	svc.SetNotificationRepo(notifRepo)

	return svc, notifRepo
}

func TestNotificationService_Start_NoEventBus(t *testing.T) {
	svc := NewNotificationService(nil, nil, memory.NewMemoryUserRepository(), "https://app.example.com")

	err := svc.Start()
	if err != nil {
		t.Errorf("Expected nil error when eventBus is nil, got: %v", err)
	}
}

func TestNotificationService_Start_WithEventBus(t *testing.T) {
	eventBus := memory.NewMemoryEventBus()
	svc := NewNotificationService(eventBus, nil, memory.NewMemoryUserRepository(), "https://app.example.com")

	err := svc.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}
}

func TestNotificationService_DeployStarted(t *testing.T) {
	eventBus := memory.NewMemoryEventBus()
	userRepo := memory.NewMemoryUserRepository()
	notifRepo := memory.NewMemoryNotificationRepository()
	svc := NewNotificationService(eventBus, nil, userRepo, "https://app.example.com")
	svc.SetNotificationRepo(notifRepo)
	svc.Start()

	ctx := context.Background()
	userID := "user-deploy"

	eventBus.Publish(ctx, &entities.PlatformEvent{
		Subject:   entities.EventDeployStarted,
		UserID:    userID,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"app_name": "my-app"},
	})

	// Give async handler time to run
	time.Sleep(50 * time.Millisecond)

	notifs, err := notifRepo.ListByUser(ctx, userID, 10)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(notifs) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifs))
	}
	if notifs[0].Type != entities.NotifDeploy {
		t.Errorf("Expected type deploy, got %s", notifs[0].Type)
	}
	if notifs[0].Title != "Deployment started" {
		t.Errorf("Expected title 'Deployment started', got '%s'", notifs[0].Title)
	}
}

func TestNotificationService_DeployCompleted(t *testing.T) {
	eventBus := memory.NewMemoryEventBus()
	userRepo := memory.NewMemoryUserRepository()
	notifRepo := memory.NewMemoryNotificationRepository()
	svc := NewNotificationService(eventBus, nil, userRepo, "https://app.example.com")
	svc.SetNotificationRepo(notifRepo)
	svc.Start()

	ctx := context.Background()
	userID := "user-deploy-done"

	eventBus.Publish(ctx, &entities.PlatformEvent{
		Subject:   entities.EventDeployCompleted,
		UserID:    userID,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"app_name": "my-app", "image": "img:v1"},
	})

	time.Sleep(50 * time.Millisecond)

	notifs, _ := notifRepo.ListByUser(ctx, userID, 10)
	if len(notifs) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifs))
	}
	if notifs[0].Title != "Deployment successful" {
		t.Errorf("Expected title 'Deployment successful', got '%s'", notifs[0].Title)
	}
}

func TestNotificationService_DeployFailed(t *testing.T) {
	eventBus := memory.NewMemoryEventBus()
	userRepo := memory.NewMemoryUserRepository()
	notifRepo := memory.NewMemoryNotificationRepository()
	svc := NewNotificationService(eventBus, nil, userRepo, "https://app.example.com")
	svc.SetNotificationRepo(notifRepo)
	svc.Start()

	ctx := context.Background()
	userID := "user-deploy-fail"

	eventBus.Publish(ctx, &entities.PlatformEvent{
		Subject:   entities.EventDeployFailed,
		UserID:    userID,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"app_name": "my-app", "error": "OOM killed"},
	})

	time.Sleep(50 * time.Millisecond)

	notifs, _ := notifRepo.ListByUser(ctx, userID, 10)
	if len(notifs) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifs))
	}
	if notifs[0].Type != entities.NotifAlert {
		t.Errorf("Expected type alert, got %s", notifs[0].Type)
	}
	if notifs[0].Title != "Deployment failed" {
		t.Errorf("Expected title 'Deployment failed', got '%s'", notifs[0].Title)
	}
}

func TestNotificationService_BillingEvents(t *testing.T) {
	eventBus := memory.NewMemoryEventBus()
	userRepo := memory.NewMemoryUserRepository()
	notifRepo := memory.NewMemoryNotificationRepository()
	svc := NewNotificationService(eventBus, nil, userRepo, "https://app.example.com")
	svc.SetNotificationRepo(notifRepo)
	svc.Start()

	ctx := context.Background()
	userID := "user-billing"

	tests := []struct {
		subject  entities.EventSubject
		data     map[string]interface{}
		title    string
		notifType entities.NotificationType
	}{
		{
			entities.EventBillingPaymentFailed,
			map[string]interface{}{},
			"Payment failed",
			entities.NotifAlert,
		},
		{
			entities.EventBillingSubscriptionCanceled,
			map[string]interface{}{"previous_tier": "pro"},
			"Subscription canceled",
			entities.NotifBilling,
		},
		{
			entities.EventBillingSubscriptionUpdated,
			map[string]interface{}{"new_tier": "team"},
			"Plan updated",
			entities.NotifBilling,
		},
	}

	for _, tt := range tests {
		eventBus.Publish(ctx, &entities.PlatformEvent{
			Subject:   tt.subject,
			UserID:    userID,
			Timestamp: time.Now(),
			Data:      tt.data,
		})
	}

	time.Sleep(100 * time.Millisecond)

	notifs, _ := notifRepo.ListByUser(ctx, userID, 20)
	if len(notifs) != 3 {
		t.Fatalf("Expected 3 billing notifications, got %d", len(notifs))
	}
}

func TestNotificationService_CheckoutCompleted(t *testing.T) {
	eventBus := memory.NewMemoryEventBus()
	userRepo := memory.NewMemoryUserRepository()
	notifRepo := memory.NewMemoryNotificationRepository()
	svc := NewNotificationService(eventBus, nil, userRepo, "https://app.example.com")
	svc.SetNotificationRepo(notifRepo)
	svc.Start()

	ctx := context.Background()
	userID := "user-checkout"

	eventBus.Publish(ctx, &entities.PlatformEvent{
		Subject:   entities.EventBillingCheckoutCompleted,
		UserID:    userID,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"tier": "pro"},
	})

	time.Sleep(50 * time.Millisecond)

	notifs, _ := notifRepo.ListByUser(ctx, userID, 10)
	if len(notifs) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifs))
	}
	if notifs[0].Type != entities.NotifBilling {
		t.Errorf("Expected type billing, got %s", notifs[0].Type)
	}
}

func TestNotificationService_NoRepoSkipsCreation(t *testing.T) {
	eventBus := memory.NewMemoryEventBus()
	userRepo := memory.NewMemoryUserRepository()
	// Do NOT set notification repo
	svc := NewNotificationService(eventBus, nil, userRepo, "https://app.example.com")
	svc.Start()

	ctx := context.Background()

	// This should not panic even without notifRepo
	eventBus.Publish(ctx, &entities.PlatformEvent{
		Subject:   entities.EventDeployStarted,
		UserID:    "user-no-repo",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"app_name": "my-app"},
	})

	time.Sleep(50 * time.Millisecond)
	// If we got here without panic, the test passes
}
