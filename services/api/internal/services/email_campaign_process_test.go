package services

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// --- processTemplate integration test ---

func TestProcessTemplate_Welcome_SendsEmail(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
	emailSender := &mockCampaignEmailSender{}
	svc.SetEmailSender(emailSender)

	ctx := context.Background()

	// Create a user
	user, _ := userRepo.Create(ctx, "campaign@example.com", "Test1234", "Campaign User", entities.RoleCustomer)

	// Record a signup event 30 minutes ago (within welcome window)
	eventRepo.Track(ctx, &entities.UserEvent{
		UserID:    user.ID,
		EventType: entities.EventSignup,
		CreatedAt: time.Now().Add(-30 * time.Minute),
	})

	// Process the welcome template
	svc.processTemplate(ctx, entities.EmailWelcome, svc.shouldSendWelcome)

	// Verify email was sent
	if emailSender.genericCount == 0 {
		t.Error("Expected welcome email to be sent")
	}

	// Verify it's recorded as sent
	sent, _ := emailSendRepo.HasSent(ctx, user.ID, entities.EmailWelcome)
	if !sent {
		t.Error("Expected welcome email to be recorded as sent")
	}
}

func TestProcessTemplate_Welcome_SkipAlreadySent(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
	emailSender := &mockCampaignEmailSender{}
	svc.SetEmailSender(emailSender)

	ctx := context.Background()

	user, _ := userRepo.Create(ctx, "already-sent@example.com", "Test1234", "Already Sent", entities.RoleCustomer)

	eventRepo.Track(ctx, &entities.UserEvent{
		UserID:    user.ID,
		EventType: entities.EventSignup,
		CreatedAt: time.Now().Add(-30 * time.Minute),
	})

	// Mark as already sent
	emailSendRepo.Record(ctx, &entities.EmailSend{
		UserID:      user.ID,
		TemplateKey: entities.EmailWelcome,
	})

	// Process — should skip
	svc.processTemplate(ctx, entities.EmailWelcome, svc.shouldSendWelcome)

	if emailSender.genericCount != 0 {
		t.Error("Expected no email to be sent (already sent)")
	}
}

func TestProcessTemplate_Welcome_SkipOldSignup(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
	emailSender := &mockCampaignEmailSender{}
	svc.SetEmailSender(emailSender)

	ctx := context.Background()

	user, _ := userRepo.Create(ctx, "old-signup@example.com", "Test1234", "Old Signup", entities.RoleCustomer)

	// Signup event from 3 hours ago (outside welcome window)
	eventRepo.Track(ctx, &entities.UserEvent{
		UserID:    user.ID,
		EventType: entities.EventSignup,
		CreatedAt: time.Now().Add(-3 * time.Hour),
	})

	svc.processTemplate(ctx, entities.EmailWelcome, svc.shouldSendWelcome)

	if emailSender.genericCount != 0 {
		t.Error("Expected no email for old signup")
	}
}

func TestProcessTemplate_Day3Nudge_NoApp(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
	emailSender := &mockCampaignEmailSender{}
	svc.SetEmailSender(emailSender)

	ctx := context.Background()

	user, _ := userRepo.Create(ctx, "nudge@example.com", "Test1234", "Nudge User", entities.RoleCustomer)

	// Signup 72 hours ago (in 60h-96h window)
	eventRepo.Track(ctx, &entities.UserEvent{
		UserID:    user.ID,
		EventType: entities.EventSignup,
		CreatedAt: time.Now().Add(-72 * time.Hour),
	})

	svc.processTemplate(ctx, entities.EmailDay3Nudge, svc.shouldSendDay3Nudge)

	if emailSender.genericCount == 0 {
		t.Error("Expected day3 nudge email to be sent")
	}
}

// --- processAll integration test ---

func TestProcessAll_WithSender_Processes(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
	emailSender := &mockCampaignEmailSender{}
	svc.SetEmailSender(emailSender)

	ctx := context.Background()

	user, _ := userRepo.Create(ctx, "processall@example.com", "Test1234", "ProcessAll User", entities.RoleCustomer)

	// Recent signup
	eventRepo.Track(ctx, &entities.UserEvent{
		UserID:    user.ID,
		EventType: entities.EventSignup,
		CreatedAt: time.Now().Add(-30 * time.Minute),
	})

	// processAll should process all templates
	svc.processAll()

	// At minimum the welcome email should be sent
	if emailSender.genericCount == 0 {
		t.Error("Expected at least one email to be sent during processAll")
	}
}

// --- shouldSendDay3Engage with app ---

func TestShouldSendDay3Engage_WithApp(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")

	ctx := context.Background()

	user, _ := userRepo.Create(ctx, "engage@example.com", "Test1234", "Engage User", entities.RoleCustomer)

	// Create an app event
	eventRepo.Track(ctx, &entities.UserEvent{
		UserID:    user.ID,
		EventType: entities.EventAppCreate,
		CreatedAt: time.Now().Add(-48 * time.Hour),
	})

	signupTime := time.Now().Add(-72 * time.Hour)
	result := svc.shouldSendDay3Engage(ctx, user.ID, signupTime)
	if !result {
		t.Error("Expected shouldSendDay3Engage=true for user with app in window")
	}
}

func TestShouldSendDay3Engage_WithoutApp(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	ctx := context.Background()

	// No app created
	signupTime := time.Now().Add(-72 * time.Hour)
	result := svc.shouldSendDay3Engage(ctx, "user-no-app", signupTime)
	if result {
		t.Error("Expected shouldSendDay3Engage=false for user without app")
	}
}

// --- shouldSendDay7Trial with activity ---

func TestShouldSendDay7Trial_InWindow_WithActivity(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, "trial@example.com", "Test1234", "Trial User", entities.RoleCustomer)

	signupTime := time.Now().Add(-7 * 24 * time.Hour)

	// Add 3+ activities
	for i := 0; i < 4; i++ {
		eventRepo.Track(ctx, &entities.UserEvent{
			UserID:    user.ID,
			EventType: "page.view",
			CreatedAt: time.Now().Add(-time.Duration(i) * 24 * time.Hour),
		})
	}

	result := svc.shouldSendDay7Trial(ctx, user.ID, signupTime)
	if !result {
		t.Error("Expected shouldSendDay7Trial=true for active free user in window")
	}
}

type mockCampaignEmailSender struct {
	genericCount int
}

func (m *mockCampaignEmailSender) SendVerificationEmail(_ context.Context, to, name, url string) error {
	return nil
}
func (m *mockCampaignEmailSender) SendTeamInviteEmail(_ context.Context, to, inviterName, teamName, inviteURL string) error {
	return nil
}
func (m *mockCampaignEmailSender) SendSupportTicketNotification(_ context.Context, to, ticketSubject, ticketURL string) error {
	return nil
}
func (m *mockCampaignEmailSender) SendSupportReplyNotification(_ context.Context, to, userName, ticketSubject, ticketURL string) error {
	return nil
}
func (m *mockCampaignEmailSender) SendGenericEmail(_ context.Context, to, subject, htmlBody string) error {
	m.genericCount++
	return nil
}
