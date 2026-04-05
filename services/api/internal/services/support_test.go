package services

import (
	"context"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func newTestSupportService() (*SupportService, *memory.MemoryUserPlanRepository) {
	repo := memory.NewMemorySupportRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	userRepo := memory.NewMemoryUserRepository()
	svc := NewSupportService(repo, planRepo, userRepo)
	return svc, planRepo
}

// --- CreateTicket tests ---

func TestCreateTicket_FreePlanDenied(t *testing.T) {
	svc, _ := newTestSupportService()
	ctx := context.Background()

	// Default plan is free — should be denied
	_, err := svc.CreateTicket(ctx, "user-free", "Help me", "", "", "I need help")
	if err == nil {
		t.Error("Expected error for free plan user creating ticket")
	}
	if !strings.Contains(err.Error(), "Pro plan") {
		t.Errorf("Expected Pro plan error, got: %v", err)
	}
}

func TestCreateTicket_ProPlanAllowed(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-pro"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	ticket, err := svc.CreateTicket(ctx, userID, "Help me", "technical", "high", "I need help")
	if err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}
	if ticket.Subject != "Help me" {
		t.Errorf("Expected subject 'Help me', got '%s'", ticket.Subject)
	}
	if ticket.Category != entities.TicketCategoryTechnical {
		t.Errorf("Expected category technical, got %s", ticket.Category)
	}
	if ticket.Priority != entities.TicketPriorityHigh {
		t.Errorf("Expected priority high, got %s", ticket.Priority)
	}
	if ticket.Status != entities.TicketStatusOpen {
		t.Errorf("Expected status open, got %s", ticket.Status)
	}
}

func TestCreateTicket_DefaultCategoryAndPriority(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-defaults"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	ticket, err := svc.CreateTicket(ctx, userID, "Default test", "", "", "Testing defaults")
	if err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}
	if ticket.Category != entities.TicketCategoryGeneral {
		t.Errorf("Expected default category 'general', got '%s'", ticket.Category)
	}
	if ticket.Priority != entities.TicketPriorityNormal {
		t.Errorf("Expected default priority 'normal', got '%s'", ticket.Priority)
	}
}

// --- ListMyTickets tests ---

func TestListMyTickets_FreePlanDenied(t *testing.T) {
	svc, _ := newTestSupportService()
	ctx := context.Background()

	_, err := svc.ListMyTickets(ctx, "user-free")
	if err == nil {
		t.Error("Expected error for free plan user listing tickets")
	}
}

func TestListMyTickets_ReturnsUserTickets(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-list"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	svc.CreateTicket(ctx, userID, "Ticket 1", "", "", "Body 1")
	svc.CreateTicket(ctx, userID, "Ticket 2", "", "", "Body 2")

	tickets, err := svc.ListMyTickets(ctx, userID)
	if err != nil {
		t.Fatalf("ListMyTickets failed: %v", err)
	}
	if len(tickets) != 2 {
		t.Errorf("Expected 2 tickets, got %d", len(tickets))
	}
}

// --- GetTicket tests ---

func TestGetTicket_OwnerCanView(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-view"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	ticket, _ := svc.CreateTicket(ctx, userID, "View test", "", "", "Body")

	got, messages, err := svc.GetTicket(ctx, userID, ticket.ID)
	if err != nil {
		t.Fatalf("GetTicket failed: %v", err)
	}
	if got.ID != ticket.ID {
		t.Errorf("Expected ticket ID %s, got %s", ticket.ID, got.ID)
	}
	if len(messages) != 1 {
		t.Errorf("Expected 1 initial message, got %d", len(messages))
	}
}

func TestGetTicket_OtherUserDenied(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	ownerID := "user-owner"
	otherID := "user-other"
	planRepo.SetUserPlan(ctx, ownerID, entities.PlanPro)
	planRepo.SetUserPlan(ctx, otherID, entities.PlanPro)

	ticket, _ := svc.CreateTicket(ctx, ownerID, "Private", "", "", "Body")

	_, _, err := svc.GetTicket(ctx, otherID, ticket.ID)
	if err == nil {
		t.Error("Expected error when another user views ticket")
	}
}

// --- AddUserMessage tests ---

func TestAddUserMessage_Success(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-msg"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	ticket, _ := svc.CreateTicket(ctx, userID, "Msg test", "", "", "Initial")
	msg, err := svc.AddUserMessage(ctx, userID, ticket.ID, "Follow up")
	if err != nil {
		t.Fatalf("AddUserMessage failed: %v", err)
	}
	if msg.Body != "Follow up" {
		t.Errorf("Expected body 'Follow up', got '%s'", msg.Body)
	}
	if msg.SenderRole != entities.SenderRoleUser {
		t.Errorf("Expected sender role user, got %s", msg.SenderRole)
	}
}

func TestAddUserMessage_ClosedTicketDenied(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-closed"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	ticket, _ := svc.CreateTicket(ctx, userID, "Close test", "", "", "Initial")

	// Close the ticket via admin
	svc.AdminUpdateStatus(ctx, ticket.ID, entities.TicketStatusClosed)

	_, err := svc.AddUserMessage(ctx, userID, ticket.ID, "After close")
	if err == nil {
		t.Error("Expected error when adding message to closed ticket")
	}
}

func TestAddUserMessage_OtherUserDenied(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	ownerID := "user-owner"
	otherID := "user-other"
	planRepo.SetUserPlan(ctx, ownerID, entities.PlanPro)
	planRepo.SetUserPlan(ctx, otherID, entities.PlanPro)

	ticket, _ := svc.CreateTicket(ctx, ownerID, "Private", "", "", "Body")

	_, err := svc.AddUserMessage(ctx, otherID, ticket.ID, "Intruder message")
	if err == nil {
		t.Error("Expected error when another user adds message to ticket")
	}
}

// --- Admin tests ---

func TestAdminListTickets(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-admin-list"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	svc.CreateTicket(ctx, userID, "T1", "", "", "B1")
	svc.CreateTicket(ctx, userID, "T2", "", "", "B2")

	tickets, total, err := svc.AdminListTickets(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("AdminListTickets failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(tickets) != 2 {
		t.Errorf("Expected 2 tickets, got %d", len(tickets))
	}
}

func TestAdminListTickets_DefaultLimit(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-limit"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	svc.CreateTicket(ctx, userID, "T1", "", "", "B1")

	// Limit 0 should default to 20
	tickets, _, err := svc.AdminListTickets(ctx, "", 0, 0)
	if err != nil {
		t.Fatalf("AdminListTickets failed: %v", err)
	}
	if len(tickets) != 1 {
		t.Errorf("Expected 1 ticket, got %d", len(tickets))
	}
}

func TestAdminReply_SetsWaitingOnCustomer(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-reply"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	ticket, _ := svc.CreateTicket(ctx, userID, "Reply test", "", "", "Help")

	msg, err := svc.AdminReply(ctx, "admin-1", ticket.ID, "Working on it")
	if err != nil {
		t.Fatalf("AdminReply failed: %v", err)
	}
	if msg.SenderRole != entities.SenderRoleAdmin {
		t.Errorf("Expected sender role admin, got %s", msg.SenderRole)
	}

	// Check ticket status changed to waiting-on-customer
	got, _, _ := svc.AdminGetTicket(ctx, ticket.ID)
	if got.Status != entities.TicketStatusWaitingOnCustomer {
		t.Errorf("Expected status waiting-on-customer, got %s", got.Status)
	}
}

func TestAdminAssignTicket(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-assign"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	ticket, _ := svc.CreateTicket(ctx, userID, "Assign test", "", "", "Help")

	err := svc.AdminAssignTicket(ctx, ticket.ID, "admin-2")
	if err != nil {
		t.Fatalf("AdminAssignTicket failed: %v", err)
	}

	got, _, _ := svc.AdminGetTicket(ctx, ticket.ID)
	if got.AssignedTo != "admin-2" {
		t.Errorf("Expected assigned to admin-2, got %s", got.AssignedTo)
	}
}

func TestAddUserMessage_WaitingToOpen(t *testing.T) {
	svc, planRepo := newTestSupportService()
	ctx := context.Background()

	userID := "user-reopen"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	ticket, _ := svc.CreateTicket(ctx, userID, "Reopen test", "", "", "Help")

	// Admin replies (sets to waiting-on-customer)
	svc.AdminReply(ctx, "admin-1", ticket.ID, "Working on it")

	// User replies — should move back to open
	svc.AddUserMessage(ctx, userID, ticket.ID, "Still broken")

	got, _, _ := svc.GetTicket(ctx, userID, ticket.ID)
	if got.Status != entities.TicketStatusOpen {
		t.Errorf("Expected status open after user reply, got %s", got.Status)
	}
}
