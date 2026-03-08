package handlers

import (
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// SupportHandler handles support ticket HTTP endpoints.
type SupportHandler struct {
	svc *services.SupportService
}

// NewSupportHandler creates a new SupportHandler.
func NewSupportHandler(svc *services.SupportService) *SupportHandler {
	return &SupportHandler{svc: svc}
}

// ---------- User endpoints ----------

// CreateTicket creates a new support ticket.
// POST /api/v1/support/tickets
func (h *SupportHandler) CreateTicket(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var input dto.CreateTicketInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Subject == "" || input.Message == "" {
		return fiber.NewError(fiber.StatusBadRequest, "subject and message are required")
	}

	ticket, err := h.svc.CreateTicket(c.Context(), userID, input.Subject, input.Category, input.Priority, input.Message)
	if err != nil {
		if strings.Contains(err.Error(), "support requires Pro") {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(ticket)
}

// ListTickets lists the user's support tickets.
// GET /api/v1/support/tickets
func (h *SupportHandler) ListTickets(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	tickets, err := h.svc.ListMyTickets(c.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "support requires Pro") {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if tickets == nil {
		tickets = []entities.SupportTicket{}
	}

	return c.JSON(tickets)
}

// GetTicket returns a single ticket with messages.
// GET /api/v1/support/tickets/:ticketId
func (h *SupportHandler) GetTicket(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	ticketID := c.Params("ticketId")

	ticket, messages, err := h.svc.GetTicket(c.Context(), userID, ticketID)
	if err != nil {
		if strings.Contains(err.Error(), "support requires Pro") {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
		if strings.Contains(err.Error(), "not your ticket") {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
		if strings.Contains(err.Error(), "not found") {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if messages == nil {
		messages = []entities.SupportMessage{}
	}

	return c.JSON(fiber.Map{
		"ticket":   ticket,
		"messages": messages,
	})
}

// AddMessage adds a user message to a ticket.
// POST /api/v1/support/tickets/:ticketId/messages
func (h *SupportHandler) AddMessage(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	ticketID := c.Params("ticketId")

	var input dto.AddMessageInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Body == "" {
		return fiber.NewError(fiber.StatusBadRequest, "body is required")
	}

	msg, err := h.svc.AddUserMessage(c.Context(), userID, ticketID, input.Body)
	if err != nil {
		if strings.Contains(err.Error(), "support requires Pro") {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
		if strings.Contains(err.Error(), "not your ticket") {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
		if strings.Contains(err.Error(), "closed") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(msg)
}

// ---------- Admin endpoints ----------

// AdminListTickets lists all tickets for admin with pagination and filtering.
// GET /api/v1/admin/support/tickets
func (h *SupportHandler) AdminListTickets(c *fiber.Ctx) error {
	status := c.Query("status")
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	tickets, total, err := h.svc.AdminListTickets(c.Context(), status, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if tickets == nil {
		tickets = []entities.SupportTicket{}
	}

	return c.JSON(fiber.Map{
		"items": tickets,
		"total": total,
	})
}

// AdminGetTicket returns a ticket with messages for admin.
// GET /api/v1/admin/support/tickets/:ticketId
func (h *SupportHandler) AdminGetTicket(c *fiber.Ctx) error {
	ticketID := c.Params("ticketId")

	ticket, messages, err := h.svc.AdminGetTicket(c.Context(), ticketID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if messages == nil {
		messages = []entities.SupportMessage{}
	}

	return c.JSON(fiber.Map{
		"ticket":   ticket,
		"messages": messages,
	})
}

// AdminReply adds an admin reply to a ticket.
// POST /api/v1/admin/support/tickets/:ticketId/reply
func (h *SupportHandler) AdminReply(c *fiber.Ctx) error {
	adminUserID, _ := c.Locals("user_id").(string)
	ticketID := c.Params("ticketId")

	var input dto.AddMessageInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Body == "" {
		return fiber.NewError(fiber.StatusBadRequest, "body is required")
	}

	msg, err := h.svc.AdminReply(c.Context(), adminUserID, ticketID, input.Body)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(msg)
}

// AdminUpdateStatus changes a ticket's status.
// PUT /api/v1/admin/support/tickets/:ticketId/status
func (h *SupportHandler) AdminUpdateStatus(c *fiber.Ctx) error {
	ticketID := c.Params("ticketId")

	var input dto.UpdateTicketStatusInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Status == "" {
		return fiber.NewError(fiber.StatusBadRequest, "status is required")
	}

	if err := h.svc.AdminUpdateStatus(c.Context(), ticketID, entities.TicketStatus(input.Status)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "status updated"})
}

// AdminAssignTicket assigns a ticket to an admin user.
// PUT /api/v1/admin/support/tickets/:ticketId/assign
func (h *SupportHandler) AdminAssignTicket(c *fiber.Ctx) error {
	ticketID := c.Params("ticketId")

	var input dto.AssignTicketInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.AdminUserID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "admin_user_id is required")
	}

	if err := h.svc.AdminAssignTicket(c.Context(), ticketID, input.AdminUserID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "ticket assigned"})
}
