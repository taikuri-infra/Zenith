package entities

import "time"

// TicketCategory classifies the support request.
type TicketCategory string

const (
	TicketCategoryGeneral        TicketCategory = "general"
	TicketCategoryBilling        TicketCategory = "billing"
	TicketCategoryTechnical      TicketCategory = "technical"
	TicketCategoryBugReport      TicketCategory = "bug-report"
	TicketCategoryFeatureRequest TicketCategory = "feature-request"
)

// TicketPriority indicates urgency.
type TicketPriority string

const (
	TicketPriorityLow    TicketPriority = "low"
	TicketPriorityNormal TicketPriority = "normal"
	TicketPriorityHigh   TicketPriority = "high"
	TicketPriorityUrgent TicketPriority = "urgent"
)

// TicketStatus tracks the lifecycle of a ticket.
type TicketStatus string

const (
	TicketStatusOpen               TicketStatus = "open"
	TicketStatusInProgress         TicketStatus = "in-progress"
	TicketStatusWaitingOnCustomer  TicketStatus = "waiting-on-customer"
	TicketStatusResolved           TicketStatus = "resolved"
	TicketStatusClosed             TicketStatus = "closed"
)

// MessageSenderRole distinguishes who sent a message.
type MessageSenderRole string

const (
	SenderRoleUser  MessageSenderRole = "user"
	SenderRoleAdmin MessageSenderRole = "admin"
)

// SupportTicket represents a user's support request.
type SupportTicket struct {
	ID         string         `json:"id"`
	UserID     string         `json:"user_id"`
	Subject    string         `json:"subject"`
	Category   TicketCategory `json:"category"`
	Priority   TicketPriority `json:"priority"`
	Status     TicketStatus   `json:"status"`
	AssignedTo string         `json:"assigned_to,omitempty"`
	ClosedAt   *time.Time     `json:"closed_at,omitempty"`
	Timestamps
}

// SupportMessage represents a single message in a ticket thread.
type SupportMessage struct {
	ID         string            `json:"id"`
	TicketID   string            `json:"ticket_id"`
	SenderID   string            `json:"sender_id"`
	SenderRole MessageSenderRole `json:"sender_role"`
	Body       string            `json:"body"`
	CreatedAt  time.Time         `json:"created_at"`
}
