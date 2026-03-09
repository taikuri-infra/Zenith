package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminCRMHandler serves CRM endpoints for Mission Control.
type AdminCRMHandler struct {
	pool *pgxpool.Pool
}

// NewAdminCRMHandler creates a new AdminCRMHandler.
func NewAdminCRMHandler(pool *pgxpool.Pool) *AdminCRMHandler {
	return &AdminCRMHandler{pool: pool}
}

// GetPipeline returns the CRM pipeline with customers by lifecycle stage.
// GET /api/v1/admin/crm/pipeline
func (h *AdminCRMHandler) GetPipeline(c *fiber.Ctx) error {
	if h.pool == nil {
		return c.JSON(entities.CRMPipeline{Stages: []entities.PipelineStage{}})
	}

	stages := []entities.PipelineStage{
		{Name: "trial"},
		{Name: "active"},
		{Name: "at_risk"},
		{Name: "churned"},
	}

	// Active customers
	rows, _ := h.pool.Query(c.Context(),
		`SELECT u.id, u.email, u.name, COALESCE(s.plan_name, 'free'),
		        COALESCE(s.price_cents, 0) / 100.0
		 FROM users u
		 LEFT JOIN subscriptions s ON s.user_id = u.id AND s.status = 'active'
		 WHERE u.role = 'customer'
		 ORDER BY u.created_at DESC`,
	)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var pc entities.PipelineCustomer
			if err := rows.Scan(&pc.ID, &pc.Email, &pc.Name, &pc.Plan, &pc.MRR); err == nil {
				pc.HealthScore = 75 // default
				if pc.MRR > 0 {
					stages[1].Customers = append(stages[1].Customers, pc)
					stages[1].Count++
				} else {
					stages[0].Customers = append(stages[0].Customers, pc)
					stages[0].Count++
				}
			}
		}
	}

	return c.JSON(entities.CRMPipeline{Stages: stages})
}

// GetHealthScores returns customer health scores.
// GET /api/v1/admin/crm/health-scores
func (h *AdminCRMHandler) GetHealthScores(c *fiber.Ctx) error {
	if h.pool == nil {
		return c.JSON([]entities.HealthScore{})
	}

	rows, err := h.pool.Query(c.Context(),
		`SELECT u.id FROM users u WHERE u.role = 'customer' ORDER BY u.created_at DESC LIMIT 50`,
	)
	if err != nil {
		return c.JSON([]entities.HealthScore{})
	}
	defer rows.Close()

	var scores []entities.HealthScore
	for rows.Next() {
		var hs entities.HealthScore
		if err := rows.Scan(&hs.UserID); err == nil {
			// Compute simplified health scores
			var appCount int
			_ = h.pool.QueryRow(c.Context(), "SELECT COUNT(*) FROM apps WHERE user_id = $1", hs.UserID).Scan(&appCount)

			var ticketCount int
			_ = h.pool.QueryRow(c.Context(),
				"SELECT COUNT(*) FROM support_tickets WHERE user_id = $1 AND status IN ('open', 'in-progress')", hs.UserID,
			).Scan(&ticketCount)

			hs.UsageScore = min(100, appCount*25)
			hs.SupportScore = max(0, 100-ticketCount*20)
			hs.LoginScore = 70 // placeholder
			hs.Score = (hs.UsageScore + hs.SupportScore + hs.LoginScore) / 3

			if hs.Score >= 70 {
				hs.RiskLevel = "healthy"
			} else if hs.Score >= 40 {
				hs.RiskLevel = "at_risk"
			} else {
				hs.RiskLevel = "critical"
			}

			scores = append(scores, hs)
		}
	}

	return c.JSON(scores)
}

// SaveNote saves a CRM note for a customer.
// PUT /api/v1/admin/crm/customers/:id/notes
func (h *AdminCRMHandler) SaveNote(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return NewBadRequest("customer id is required")
	}

	var input struct {
		Note string   `json:"note"`
		Tags []string `json:"tags"`
	}
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	authorID, _ := c.Locals("user_id").(string)

	if h.pool == nil {
		return c.JSON(fiber.Map{"message": "note saved"})
	}

	var noteID string
	err := h.pool.QueryRow(c.Context(),
		`INSERT INTO customer_notes (user_id, author_id, note, tags)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		userID, authorID, input.Note, input.Tags,
	).Scan(&noteID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save note")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": noteID, "message": "note saved"})
}

// GetNotes returns CRM notes for a customer.
// GET /api/v1/admin/crm/customers/:id/notes
func (h *AdminCRMHandler) GetNotes(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return NewBadRequest("customer id is required")
	}

	if h.pool == nil {
		return c.JSON([]entities.CustomerNote{})
	}

	rows, err := h.pool.Query(c.Context(),
		`SELECT cn.id, cn.user_id, cn.author_id, u.name, cn.note, cn.tags, cn.created_at, cn.updated_at
		 FROM customer_notes cn
		 LEFT JOIN users u ON u.id = cn.author_id
		 WHERE cn.user_id = $1
		 ORDER BY cn.created_at DESC`,
		userID,
	)
	if err != nil {
		return c.JSON([]entities.CustomerNote{})
	}
	defer rows.Close()

	var notes []entities.CustomerNote
	for rows.Next() {
		var n entities.CustomerNote
		var authorName *string
		if err := rows.Scan(&n.ID, &n.UserID, &n.AuthorID, &authorName, &n.Note, &n.Tags, &n.CreatedAt, &n.UpdatedAt); err == nil {
			if authorName != nil {
				n.AuthorName = *authorName
			}
			notes = append(notes, n)
		}
	}

	return c.JSON(notes)
}

// UpdateTags updates tags for a customer in CRM.
// PUT /api/v1/admin/crm/customers/:id/tags
func (h *AdminCRMHandler) UpdateTags(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return NewBadRequest("customer id is required")
	}

	var input struct {
		Tags []string `json:"tags"`
	}
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	// Store tags as a note with special marker
	authorID, _ := c.Locals("user_id").(string)

	if h.pool != nil {
		_, _ = h.pool.Exec(c.Context(),
			`INSERT INTO customer_notes (user_id, author_id, note, tags)
			 VALUES ($1, $2, 'Tags updated', $3)`,
			userID, authorID, input.Tags,
		)
	}

	return c.JSON(fiber.Map{"message": "tags updated"})
}
