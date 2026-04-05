package services

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// --- CheckoutCompleted with email sender ---

func TestNotificationService_CheckoutCompleted_WithEmail(t *testing.T) {
	eventBus := memory.NewMemoryEventBus()
	userRepo := memory.NewMemoryUserRepository()
	notifRepo := memory.NewMemoryNotificationRepository()
	emailSender := &mockEmailSenderNotif{}

	svc := NewNotificationService(eventBus, emailSender, userRepo, "https://app.example.com")
	svc.SetNotificationRepo(notifRepo)
	svc.Start()

	ctx := context.Background()

	// Create a user first so the email lookup works
	user, _ := userRepo.Create(ctx, "checkout@example.com", "Test1234", "Checkout User", entities.RoleCustomer)

	eventBus.Publish(ctx, &entities.PlatformEvent{
		Subject:   entities.EventBillingCheckoutCompleted,
		UserID:    user.ID,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"tier": "pro"},
	})

	time.Sleep(100 * time.Millisecond)

	notifs, _ := notifRepo.ListByUser(ctx, user.ID, 10)
	if len(notifs) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifs))
	}
	if notifs[0].Type != entities.NotifBilling {
		t.Errorf("Expected type billing, got %s", notifs[0].Type)
	}

	// Verify email was sent
	if emailSender.verifyCount == 0 {
		t.Error("Expected email to be sent for checkout completed event")
	}
}

// --- CheckoutCompleted without user in repo ---

func TestNotificationService_CheckoutCompleted_UserNotFound(t *testing.T) {
	eventBus := memory.NewMemoryEventBus()
	userRepo := memory.NewMemoryUserRepository()
	notifRepo := memory.NewMemoryNotificationRepository()
	emailSender := &mockEmailSenderNotif{}

	svc := NewNotificationService(eventBus, emailSender, userRepo, "https://app.example.com")
	svc.SetNotificationRepo(notifRepo)
	svc.Start()

	ctx := context.Background()

	// Dispatch event for nonexistent user - should not crash
	eventBus.Publish(ctx, &entities.PlatformEvent{
		Subject:   entities.EventBillingCheckoutCompleted,
		UserID:    "nonexistent-user",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"tier": "pro"},
	})

	time.Sleep(100 * time.Millisecond)

	// Notification should still be created (createNotification doesn't need user repo)
	notifs, _ := notifRepo.ListByUser(ctx, "nonexistent-user", 10)
	if len(notifs) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifs))
	}

	// But email should NOT be sent (user not found)
	if emailSender.verifyCount != 0 {
		t.Error("Expected no email when user not found")
	}
}

// --- All notification event types covered ---

func TestNotificationService_AllEventTypes(t *testing.T) {
	eventBus := memory.NewMemoryEventBus()
	userRepo := memory.NewMemoryUserRepository()
	notifRepo := memory.NewMemoryNotificationRepository()

	svc := NewNotificationService(eventBus, nil, userRepo, "https://app.example.com")
	svc.SetNotificationRepo(notifRepo)
	svc.Start()

	ctx := context.Background()
	userID := "user-all-events"

	events := []struct {
		subject entities.EventSubject
		data    map[string]interface{}
	}{
		{entities.EventDeployStarted, map[string]interface{}{"app_name": "app1"}},
		{entities.EventDeployCompleted, map[string]interface{}{"app_name": "app1", "image": "img:v1"}},
		{entities.EventDeployFailed, map[string]interface{}{"app_name": "app1", "error": "OOM"}},
		{entities.EventBillingCheckoutCompleted, map[string]interface{}{"tier": "pro"}},
		{entities.EventBillingPaymentFailed, map[string]interface{}{}},
		{entities.EventBillingSubscriptionCanceled, map[string]interface{}{"previous_tier": "pro"}},
		{entities.EventBillingSubscriptionUpdated, map[string]interface{}{"new_tier": "team"}},
	}

	for _, e := range events {
		eventBus.Publish(ctx, &entities.PlatformEvent{
			Subject:   e.subject,
			UserID:    userID,
			Timestamp: time.Now(),
			Data:      e.data,
		})
	}

	time.Sleep(200 * time.Millisecond)

	notifs, _ := notifRepo.ListByUser(ctx, userID, 20)
	if len(notifs) != 7 {
		t.Errorf("Expected 7 notifications (one per event type), got %d", len(notifs))
	}
}

// --- SetNotificationRepo tests ---

func TestSetNotificationRepo(t *testing.T) {
	svc := NewNotificationService(nil, nil, memory.NewMemoryUserRepository(), "https://app.example.com")
	notifRepo := memory.NewMemoryNotificationRepository()
	svc.SetNotificationRepo(notifRepo)
	// No panic means success
}

// --- NewNotificationService tests ---

func TestNewNotificationService_AllNil(t *testing.T) {
	svc := NewNotificationService(nil, nil, nil, "")
	if svc == nil {
		t.Fatal("Expected non-nil NotificationService")
	}
}

// mockEmailSenderNotif implements ports.EmailSender for notification tests.
type mockEmailSenderNotif struct {
	verifyCount int
}

func (m *mockEmailSenderNotif) SendVerificationEmail(_ context.Context, to, name, url string) error {
	m.verifyCount++
	return nil
}

func (m *mockEmailSenderNotif) SendTeamInviteEmail(_ context.Context, to, inviterName, teamName, inviteURL string) error {
	return nil
}

func (m *mockEmailSenderNotif) SendSupportTicketNotification(_ context.Context, to, ticketSubject, ticketURL string) error {
	return nil
}

func (m *mockEmailSenderNotif) SendSupportReplyNotification(_ context.Context, to, userName, ticketSubject, ticketURL string) error {
	return nil
}

func (m *mockEmailSenderNotif) SendGenericEmail(_ context.Context, to, subject, htmlBody string) error {
	return nil
}
