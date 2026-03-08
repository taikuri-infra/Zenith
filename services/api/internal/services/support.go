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

// SupportService handles support ticket operations with plan gating.
type SupportService struct {
	repo        ports.SupportRepository
	planRepo    ports.UserPlanRepository
	userRepo    ports.UserRepository
	emailSender ports.EmailSender
	appURL      string
	adminEmail  string
}

// NewSupportService creates a new SupportService.
func NewSupportService(repo ports.SupportRepository, planRepo ports.UserPlanRepository, userRepo ports.UserRepository) *SupportService {
	return &SupportService{
		repo:     repo,
		planRepo: planRepo,
		userRepo: userRepo,
	}
}

// SetEmailSender configures email notifications.
func (s *SupportService) SetEmailSender(sender ports.EmailSender, appURL, adminEmail string) {
	s.emailSender = sender
	s.appURL = appURL
	s.adminEmail = adminEmail
}

// requireProPlan checks that the user is on Pro plan or higher.
func (s *SupportService) requireProPlan(ctx context.Context, userID string) error {
	plan, err := s.planRepo.GetUserPlan(ctx, userID)
	if err != nil {
		return fmt.Errorf("could not determine plan: %w", err)
	}
	if plan.Tier == entities.PlanFree {
		return fmt.Errorf("support requires Pro plan or higher")
	}
	return nil
}

// CreateTicket creates a new support ticket with an initial message.
func (s *SupportService) CreateTicket(ctx context.Context, userID, subject, category, priority, body string) (*entities.SupportTicket, error) {
	if err := s.requireProPlan(ctx, userID); err != nil {
		return nil, err
	}

	if category == "" {
		category = string(entities.TicketCategoryGeneral)
	}
	if priority == "" {
		priority = string(entities.TicketPriorityNormal)
	}

	now := time.Now()
	ticketID := uuid.New().String()

	ticket := &entities.SupportTicket{
		ID:       ticketID,
		UserID:   userID,
		Subject:  subject,
		Category: entities.TicketCategory(category),
		Priority: entities.TicketPriority(priority),
		Status:   entities.TicketStatusOpen,
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	msg := &entities.SupportMessage{
		ID:         uuid.New().String(),
		TicketID:   ticketID,
		SenderID:   userID,
		SenderRole: entities.SenderRoleUser,
		Body:       body,
		CreatedAt:  now,
	}

	if err := s.repo.CreateTicket(ctx, ticket, msg); err != nil {
		return nil, err
	}

	// Notify admin via email (best effort)
	if s.emailSender != nil && s.adminEmail != "" {
		ticketURL := fmt.Sprintf("%s/support/%s", s.appURL, ticketID)
		if err := s.emailSender.SendSupportTicketNotification(ctx, s.adminEmail, subject, ticketURL); err != nil {
			slog.Error("failed to send admin notification for support ticket", "error", err)
		}
	}

	return ticket, nil
}

// ListMyTickets returns the user's tickets.
func (s *SupportService) ListMyTickets(ctx context.Context, userID string) ([]entities.SupportTicket, error) {
	if err := s.requireProPlan(ctx, userID); err != nil {
		return nil, err
	}
	return s.repo.ListTicketsByUser(ctx, userID)
}

// GetTicket returns a ticket with its messages (user must own it).
func (s *SupportService) GetTicket(ctx context.Context, userID, ticketID string) (*entities.SupportTicket, []entities.SupportMessage, error) {
	if err := s.requireProPlan(ctx, userID); err != nil {
		return nil, nil, err
	}

	ticket, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, nil, err
	}
	if ticket.UserID != userID {
		return nil, nil, fmt.Errorf("not your ticket")
	}

	messages, err := s.repo.ListMessages(ctx, ticketID)
	if err != nil {
		return nil, nil, err
	}
	return ticket, messages, nil
}

// AddUserMessage adds a user message to a ticket they own.
func (s *SupportService) AddUserMessage(ctx context.Context, userID, ticketID, body string) (*entities.SupportMessage, error) {
	if err := s.requireProPlan(ctx, userID); err != nil {
		return nil, err
	}

	ticket, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket.UserID != userID {
		return nil, fmt.Errorf("not your ticket")
	}
	if ticket.Status == entities.TicketStatusClosed {
		return nil, fmt.Errorf("ticket is closed")
	}

	msg := &entities.SupportMessage{
		ID:         uuid.New().String(),
		TicketID:   ticketID,
		SenderID:   userID,
		SenderRole: entities.SenderRoleUser,
		Body:       body,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.AddMessage(ctx, msg); err != nil {
		return nil, err
	}

	// If ticket was waiting-on-customer, move back to open
	if ticket.Status == entities.TicketStatusWaitingOnCustomer {
		s.repo.UpdateTicketStatus(ctx, ticketID, entities.TicketStatusOpen)
	}

	return msg, nil
}

// AdminListTickets returns paginated tickets for admin view.
func (s *SupportService) AdminListTickets(ctx context.Context, status string, limit, offset int) ([]entities.SupportTicket, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListAllTickets(ctx, status, limit, offset)
}

// AdminGetTicket returns a ticket with its messages (admin view).
func (s *SupportService) AdminGetTicket(ctx context.Context, ticketID string) (*entities.SupportTicket, []entities.SupportMessage, error) {
	ticket, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, nil, err
	}
	messages, err := s.repo.ListMessages(ctx, ticketID)
	if err != nil {
		return nil, nil, err
	}
	return ticket, messages, nil
}

// AdminReply adds an admin message and sets status to waiting-on-customer.
func (s *SupportService) AdminReply(ctx context.Context, adminUserID, ticketID, body string) (*entities.SupportMessage, error) {
	ticket, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	msg := &entities.SupportMessage{
		ID:         uuid.New().String(),
		TicketID:   ticketID,
		SenderID:   adminUserID,
		SenderRole: entities.SenderRoleAdmin,
		Body:       body,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.AddMessage(ctx, msg); err != nil {
		return nil, err
	}

	// Set status to waiting-on-customer
	s.repo.UpdateTicketStatus(ctx, ticketID, entities.TicketStatusWaitingOnCustomer)

	// Notify user via email (best effort)
	if s.emailSender != nil {
		user, err := s.userRepo.GetByID(ctx, ticket.UserID)
		if err == nil && user != nil {
			ticketURL := fmt.Sprintf("%s/support/%s", s.appURL, ticketID)
			if err := s.emailSender.SendSupportReplyNotification(ctx, user.Email, user.Name, ticket.Subject, ticketURL); err != nil {
				slog.Error("failed to send reply notification for support ticket", "error", err)
			}
		}
	}

	return msg, nil
}

// AdminUpdateStatus changes a ticket's status.
func (s *SupportService) AdminUpdateStatus(ctx context.Context, ticketID string, status entities.TicketStatus) error {
	return s.repo.UpdateTicketStatus(ctx, ticketID, status)
}

// AdminAssignTicket assigns a ticket to an admin user.
func (s *SupportService) AdminAssignTicket(ctx context.Context, ticketID, adminUserID string) error {
	return s.repo.UpdateTicketAssignee(ctx, ticketID, adminUserID)
}
