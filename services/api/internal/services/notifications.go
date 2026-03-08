package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
)

// NotificationService subscribes to platform events and sends user notifications.
type NotificationService struct {
	eventBus  ports.EventBus
	email     ports.EmailSender
	userRepo  ports.UserRepository
	notifRepo ports.NotificationRepository
	appURL    string
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(eventBus ports.EventBus, email ports.EmailSender, userRepo ports.UserRepository, appURL string) *NotificationService {
	return &NotificationService{
		eventBus: eventBus,
		email:    email,
		userRepo: userRepo,
		appURL:   appURL,
	}
}

// SetNotificationRepo configures the notification repository for persisting notifications.
func (s *NotificationService) SetNotificationRepo(repo ports.NotificationRepository) {
	s.notifRepo = repo
}

// Start subscribes to all platform events and dispatches notifications.
func (s *NotificationService) Start() error {
	if s.eventBus == nil {
		return nil
	}

	// Subscribe to all zenith events
	return s.eventBus.Subscribe("zenith.>", s.handleEvent)
}

func (s *NotificationService) handleEvent(event *entities.PlatformEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch event.Subject {
	case entities.EventDeployStarted:
		s.createNotification(ctx, event.UserID, entities.NotifDeploy, "Deployment started",
			fmt.Sprintf("Building and deploying %s", event.Data["app_name"]))
	case entities.EventDeployCompleted:
		s.onDeployCompleted(ctx, event)
	case entities.EventDeployFailed:
		s.onDeployFailed(ctx, event)
	case entities.EventBillingCheckoutCompleted:
		s.onCheckoutCompleted(ctx, event)
	case entities.EventBillingPaymentFailed:
		s.onPaymentFailed(ctx, event)
	case entities.EventBillingSubscriptionCanceled:
		s.onSubscriptionCanceled(ctx, event)
	case entities.EventBillingSubscriptionUpdated:
		tier, _ := event.Data["new_tier"].(string)
		s.createNotification(ctx, event.UserID, entities.NotifBilling, "Plan updated",
			fmt.Sprintf("Your plan has been updated to %s", tier))
	}
}

func (s *NotificationService) createNotification(ctx context.Context, userID string, notifType entities.NotificationType, title, message string) {
	if s.notifRepo == nil {
		return
	}
	notif := &entities.Notification{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Message:   message,
		CreatedAt: time.Now(),
	}
	if err := s.notifRepo.CreateNotification(ctx, notif); err != nil {
		slog.Error("failed to create notification", "user_id", userID, "error", err)
	}
}

func (s *NotificationService) onDeployCompleted(ctx context.Context, event *entities.PlatformEvent) {
	appName, _ := event.Data["app_name"].(string)
	image, _ := event.Data["image"].(string)
	slog.Info("deploy completed", "app", appName, "image", image, "user_id", event.UserID)
	s.createNotification(ctx, event.UserID, entities.NotifDeploy, "Deployment successful",
		fmt.Sprintf("%s has been deployed successfully", appName))
}

func (s *NotificationService) onDeployFailed(ctx context.Context, event *entities.PlatformEvent) {
	appName, _ := event.Data["app_name"].(string)
	errMsg, _ := event.Data["error"].(string)
	slog.Error("deploy failed", "app", appName, "error", errMsg, "user_id", event.UserID)
	s.createNotification(ctx, event.UserID, entities.NotifAlert, "Deployment failed",
		fmt.Sprintf("Deployment of %s failed: %s", appName, errMsg))
}

func (s *NotificationService) onCheckoutCompleted(ctx context.Context, event *entities.PlatformEvent) {
	tier, _ := event.Data["tier"].(string)
	slog.Info("checkout completed", "user_id", event.UserID, "tier", tier)
	s.createNotification(ctx, event.UserID, entities.NotifBilling, "Welcome to "+tier,
		fmt.Sprintf("Your %s plan is now active. Enjoy your new features!", tier))

	if s.email == nil {
		return
	}
	user, err := s.userRepo.GetByID(ctx, event.UserID)
	if err != nil || user == nil {
		return
	}
	// Best-effort welcome email for upgrade
	_ = s.email.SendVerificationEmail(ctx, user.Email, user.Name,
		fmt.Sprintf("%s/dashboard?upgraded=%s", s.appURL, tier))
}

func (s *NotificationService) onPaymentFailed(ctx context.Context, event *entities.PlatformEvent) {
	slog.Warn("payment failed", "user_id", event.UserID)
	s.createNotification(ctx, event.UserID, entities.NotifAlert, "Payment failed",
		"We couldn't process your payment. Please update your payment method.")
}

func (s *NotificationService) onSubscriptionCanceled(ctx context.Context, event *entities.PlatformEvent) {
	tier, _ := event.Data["previous_tier"].(string)
	slog.Info("subscription canceled", "user_id", event.UserID, "previous_tier", tier)
	s.createNotification(ctx, event.UserID, entities.NotifBilling, "Subscription canceled",
		fmt.Sprintf("Your %s plan has been canceled. You will be downgraded to the Free plan at the end of the billing period.", tier))
}
