package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// EmailCampaignService processes drip campaign emails on a schedule.
type EmailCampaignService struct {
	emailSendRepo ports.EmailSendRepository
	eventRepo     ports.UserEventRepository
	userRepo      ports.UserRepository
	planRepo      ports.UserPlanRepository
	emailSender   ports.EmailSender
	appURL        string
	stopCh        chan struct{}
}

func NewEmailCampaignService(
	emailSendRepo ports.EmailSendRepository,
	eventRepo ports.UserEventRepository,
	userRepo ports.UserRepository,
	planRepo ports.UserPlanRepository,
	appURL string,
) *EmailCampaignService {
	return &EmailCampaignService{
		emailSendRepo: emailSendRepo,
		eventRepo:     eventRepo,
		userRepo:      userRepo,
		planRepo:      planRepo,
		appURL:        appURL,
		stopCh:        make(chan struct{}),
	}
}

func (s *EmailCampaignService) SetEmailSender(sender ports.EmailSender) {
	s.emailSender = sender
}

// Start begins the campaign processing loop (every 1 hour).
func (s *EmailCampaignService) Start() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Run once immediately
		s.processAll()

		for {
			select {
			case <-ticker.C:
				s.processAll()
			case <-s.stopCh:
				slog.Info("email campaign service stopped")
				return
			}
		}
	}()
	slog.Info("email campaign service started", "interval", "1h")
}

func (s *EmailCampaignService) Stop() {
	close(s.stopCh)
}

func (s *EmailCampaignService) processAll() {
	if s.emailSender == nil {
		return
	}
	ctx := context.Background()

	// Process each template's trigger logic
	s.processTemplate(ctx, entities.EmailWelcome, s.shouldSendWelcome)
	s.processTemplate(ctx, entities.EmailDay1Deploy, s.shouldSendDay1)
	s.processTemplate(ctx, entities.EmailDay3Engage, s.shouldSendDay3Engage)
	s.processTemplate(ctx, entities.EmailDay3Nudge, s.shouldSendDay3Nudge)
	s.processTemplate(ctx, entities.EmailDay7Trial, s.shouldSendDay7Trial)
	s.processTemplate(ctx, entities.EmailDay14Value, s.shouldSendDay14)
}

type triggerCheck func(ctx context.Context, userID string, signupTime time.Time) bool

func (s *EmailCampaignService) processTemplate(ctx context.Context, templateKey string, check triggerCheck) {
	// Get recent signups (last 30 days)
	since := time.Now().AddDate(0, -1, 0)
	events, err := s.eventRepo.ListByType(ctx, entities.EventSignup, 1000, 0)
	if err != nil {
		slog.Error("email campaign: failed to list signups", "template", templateKey, "error", err)
		return
	}

	for _, event := range events {
		if event.CreatedAt.Before(since) {
			continue
		}

		// Check if already sent
		sent, _ := s.emailSendRepo.HasSent(ctx, event.UserID, templateKey)
		if sent {
			continue
		}

		// Check trigger conditions
		if !check(ctx, event.UserID, event.CreatedAt) {
			continue
		}

		// Send email
		user, err := s.userRepo.GetByID(ctx, event.UserID)
		if err != nil || user == nil {
			continue
		}

		subject, body := getEmailContent(templateKey, user.Name, s.appURL)
		if err := s.emailSender.SendGenericEmail(ctx, user.Email, subject, body); err != nil {
			slog.Error("email campaign: failed to send", "template", templateKey, "user", event.UserID, "error", err)
			continue
		}

		// Record send
		_ = s.emailSendRepo.Record(ctx, &entities.EmailSend{
			UserID:      event.UserID,
			TemplateKey: templateKey,
		})
		slog.Info("email campaign: sent", "template", templateKey, "user", event.UserID)
	}
}

func (s *EmailCampaignService) shouldSendWelcome(_ context.Context, _ string, signupTime time.Time) bool {
	// Send immediately after signup (within 1 hour window)
	return time.Since(signupTime) < 2*time.Hour
}

func (s *EmailCampaignService) shouldSendDay1(ctx context.Context, userID string, signupTime time.Time) bool {
	if time.Since(signupTime) < 20*time.Hour || time.Since(signupTime) > 48*time.Hour {
		return false
	}
	// Only if no app created
	events, _ := s.eventRepo.GetUserActivity(ctx, userID, signupTime)
	for _, e := range events {
		if e.EventType == entities.EventAppCreate {
			return false
		}
	}
	return true
}

func (s *EmailCampaignService) shouldSendDay3Engage(ctx context.Context, userID string, signupTime time.Time) bool {
	if time.Since(signupTime) < 60*time.Hour || time.Since(signupTime) > 96*time.Hour {
		return false
	}
	// Only if has app
	events, _ := s.eventRepo.GetUserActivity(ctx, userID, signupTime)
	hasApp := false
	for _, e := range events {
		if e.EventType == entities.EventAppCreate {
			hasApp = true
			break
		}
	}
	return hasApp
}

func (s *EmailCampaignService) shouldSendDay3Nudge(ctx context.Context, userID string, signupTime time.Time) bool {
	if time.Since(signupTime) < 60*time.Hour || time.Since(signupTime) > 96*time.Hour {
		return false
	}
	// Only if no app
	events, _ := s.eventRepo.GetUserActivity(ctx, userID, signupTime)
	for _, e := range events {
		if e.EventType == entities.EventAppCreate {
			return false
		}
	}
	return true
}

func (s *EmailCampaignService) shouldSendDay7Trial(ctx context.Context, userID string, signupTime time.Time) bool {
	if time.Since(signupTime) < 6*24*time.Hour || time.Since(signupTime) > 8*24*time.Hour {
		return false
	}
	// Only for free users who are active
	plan, _ := s.planRepo.GetUserPlan(ctx, userID)
	if plan != nil && plan.Tier != entities.PlanFree {
		return false
	}
	events, _ := s.eventRepo.GetUserActivity(ctx, userID, signupTime)
	return len(events) >= 3 // At least somewhat active
}

func (s *EmailCampaignService) shouldSendDay14(ctx context.Context, userID string, signupTime time.Time) bool {
	if time.Since(signupTime) < 13*24*time.Hour || time.Since(signupTime) > 15*24*time.Hour {
		return false
	}
	plan, _ := s.planRepo.GetUserPlan(ctx, userID)
	return plan == nil || plan.Tier == entities.PlanFree
}

func getEmailContent(templateKey, userName, appURL string) (subject string, body string) {
	switch templateKey {
	case entities.EmailWelcome:
		return "Welcome to Zenith!",
			fmt.Sprintf(`<h2>Welcome to Zenith, %s!</h2>
			<p>You're all set to deploy your first cloud-native app.</p>
			<p><a href="%s">Go to Dashboard →</a></p>
			<p>Quick start: Deploy a Next.js app in under 2 minutes.</p>`, userName, appURL)

	case entities.EmailDay1Deploy:
		return "Deploy your first app in 2 minutes",
			fmt.Sprintf(`<h2>Hey %s, ready to deploy?</h2>
			<p>You signed up yesterday but haven't deployed yet. It only takes 2 minutes!</p>
			<p><a href="%s">Deploy Now →</a></p>`, userName, appURL)

	case entities.EmailDay3Engage:
		return "Level up: Add a database to your app",
			fmt.Sprintf(`<h2>Nice work, %s!</h2>
			<p>Your app is running. Ready for the next step? Add a PostgreSQL database with one click.</p>
			<p><a href="%s">Add Database →</a></p>`, userName, appURL)

	case entities.EmailDay3Nudge:
		return "Need help getting started?",
			fmt.Sprintf(`<h2>Hey %s, need a hand?</h2>
			<p>We noticed you haven't deployed yet. Need help? Reply to this email or check our docs.</p>
			<p><a href="%s">Deploy Your First App →</a></p>`, userName, appURL)

	case entities.EmailDay7Trial:
		return "Try Pro free for 7 days",
			fmt.Sprintf(`<h2>Ready for more, %s?</h2>
			<p>You've been building on Zenith for a week. Unlock custom domains, more apps, and always-on deployments with a free Pro trial.</p>
			<p><a href="%s/billing">Start Free Trial →</a></p>`, userName, appURL)

	case entities.EmailDay14Value:
		return "See what Pro users are building",
			fmt.Sprintf(`<h2>2 weeks on Zenith, %s!</h2>
			<p>Developers are shipping production apps with custom domains, databases, and team collaboration.</p>
			<p>Upgrade to Pro for just €29/mo.</p>
			<p><a href="%s/billing">Upgrade Now →</a></p>`, userName, appURL)

	default:
		return "Update from Zenith",
			fmt.Sprintf(`<p>Hi %s, check out what's new on Zenith.</p>
			<p><a href="%s">Visit Dashboard →</a></p>`, userName, appURL)
	}
}
