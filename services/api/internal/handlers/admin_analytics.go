package handlers

import (
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminAnalyticsHandler serves analytics endpoints for Mission Control.
type AdminAnalyticsHandler struct {
	pool *pgxpool.Pool
	userRepo ports.UserRepository
}

// NewAdminAnalyticsHandler creates a new AdminAnalyticsHandler.
func NewAdminAnalyticsHandler(pool *pgxpool.Pool, userRepo ports.UserRepository) *AdminAnalyticsHandler {
	return &AdminAnalyticsHandler{pool: pool, userRepo: userRepo}
}

// GetRevenue returns revenue analytics.
// GET /api/v1/admin/analytics/revenue
func (h *AdminAnalyticsHandler) GetRevenue(c *fiber.Ctx) error {
	if h.pool == nil {
		return c.JSON(entities.RevenueStats{})
	}

	var mrr float64
	_ = h.pool.QueryRow(c.Context(),
		`SELECT COALESCE(SUM(s.price_cents), 0) / 100.0
		 FROM subscriptions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.status = 'active'`,
	).Scan(&mrr)

	var totalUsers, churnedMonth int
	_ = h.pool.QueryRow(c.Context(), "SELECT COUNT(*) FROM users WHERE role = 'customer'").Scan(&totalUsers)
	_ = h.pool.QueryRow(c.Context(),
		`SELECT COUNT(*) FROM users WHERE role = 'customer' AND updated_at < now() - interval '30 days'
		 AND id NOT IN (SELECT user_id FROM subscriptions WHERE status = 'active')`,
	).Scan(&churnedMonth)

	churnRate := float64(0)
	if totalUsers > 0 {
		churnRate = float64(churnedMonth) / float64(totalUsers) * 100
	}

	// Revenue by plan
	rows, _ := h.pool.Query(c.Context(),
		`SELECT COALESCE(s.plan_name, 'free'), COALESCE(SUM(s.price_cents), 0) / 100.0, COUNT(*)
		 FROM subscriptions s WHERE s.status = 'active'
		 GROUP BY s.plan_name ORDER BY SUM(s.price_cents) DESC`,
	)
	var byPlan []entities.PlanRevenue
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var pr entities.PlanRevenue
			if err := rows.Scan(&pr.Plan, &pr.Revenue, &pr.Count); err == nil {
				byPlan = append(byPlan, pr)
			}
		}
	}

	// Monthly trend (last 12 months)
	trendRows, _ := h.pool.Query(c.Context(),
		`SELECT to_char(date_trunc('month', s.created_at), 'YYYY-MM'),
		        COALESCE(SUM(s.price_cents), 0) / 100.0
		 FROM subscriptions s
		 WHERE s.created_at > now() - interval '12 months'
		 GROUP BY date_trunc('month', s.created_at)
		 ORDER BY date_trunc('month', s.created_at)`,
	)
	var trend []entities.MonthlyRevenue
	if trendRows != nil {
		defer trendRows.Close()
		for trendRows.Next() {
			var mr entities.MonthlyRevenue
			if err := trendRows.Scan(&mr.Month, &mr.NewRevenue); err == nil {
				trend = append(trend, mr)
			}
		}
	}

	return c.JSON(entities.RevenueStats{
		MRR:           mrr,
		ARR:           mrr * 12,
		ChurnRate:     churnRate,
		RevenueByPlan: byPlan,
		MonthlyTrend:  trend,
	})
}

// GetGrowth returns growth analytics.
// GET /api/v1/admin/analytics/growth
func (h *AdminAnalyticsHandler) GetGrowth(c *fiber.Ctx) error {
	if h.pool == nil {
		return c.JSON(entities.GrowthStats{})
	}

	var total, newMonth int
	_ = h.pool.QueryRow(c.Context(), "SELECT COUNT(*) FROM users WHERE role = 'customer'").Scan(&total)
	_ = h.pool.QueryRow(c.Context(),
		"SELECT COUNT(*) FROM users WHERE role = 'customer' AND created_at > date_trunc('month', now())",
	).Scan(&newMonth)

	// Monthly growth
	rows, _ := h.pool.Query(c.Context(),
		`SELECT to_char(date_trunc('month', created_at), 'YYYY-MM'), COUNT(*)
		 FROM users WHERE role = 'customer' AND created_at > now() - interval '12 months'
		 GROUP BY date_trunc('month', created_at)
		 ORDER BY date_trunc('month', created_at)`,
	)
	var growth []entities.MonthlyGrowth
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var mg entities.MonthlyGrowth
			if err := rows.Scan(&mg.Month, &mg.New); err == nil {
				growth = append(growth, mg)
			}
		}
	}

	return c.JSON(entities.GrowthStats{
		TotalUsers:    total,
		NewThisMonth:  newMonth,
		MonthlyGrowth: growth,
	})
}

// GetUsageAnalytics returns feature usage analytics.
// GET /api/v1/admin/analytics/usage
func (h *AdminAnalyticsHandler) GetUsageAnalytics(c *fiber.Ctx) error {
	if h.pool == nil {
		return c.JSON(entities.UsageStats{})
	}

	features := []entities.FeatureUsage{}
	for _, table := range []struct{ name, label string }{
		{"apps", "Applications"},
		{"databases", "Databases"},
		{"storage_buckets", "Storage Buckets"},
		{"gateways", "API Gateways"},
		{"webhooks", "Webhooks"},
	} {
		var count, users int
		_ = h.pool.QueryRow(c.Context(),
			"SELECT COUNT(*), COUNT(DISTINCT user_id) FROM "+table.name,
		).Scan(&count, &users)
		features = append(features, entities.FeatureUsage{
			Feature: table.label, UsageCount: count, UserCount: users,
		})
	}

	return c.JSON(entities.UsageStats{TopFeatures: features})
}

// GetCohorts returns cohort retention data.
// GET /api/v1/admin/analytics/cohorts
func (h *AdminAnalyticsHandler) GetCohorts(c *fiber.Ctx) error {
	if h.pool == nil {
		return c.JSON([]entities.CohortData{})
	}

	rows, err := h.pool.Query(c.Context(),
		`WITH cohorts AS (
			SELECT id, date_trunc('month', created_at) AS cohort_month
			FROM users WHERE role = 'customer' AND created_at > now() - interval '6 months'
		)
		SELECT to_char(c.cohort_month, 'YYYY-MM'),
		       COUNT(*) FILTER (WHERE EXISTS (
		         SELECT 1 FROM audit_log a
		         WHERE a.actor = u.email AND a.created_at > now() - interval '30 days'
		       )),
		       COUNT(*)
		FROM cohorts c JOIN users u ON u.id = c.id
		GROUP BY c.cohort_month ORDER BY c.cohort_month`,
	)
	if err != nil {
		return c.JSON([]entities.CohortData{})
	}
	defer rows.Close()

	var cohorts []entities.CohortData
	for rows.Next() {
		var cd entities.CohortData
		var retained, total int
		if err := rows.Scan(&cd.Cohort, &retained, &total); err == nil {
			cd.Retained = retained
			cd.Total = total
			if total > 0 {
				cd.Percentage = float64(retained) / float64(total) * 100
			}
			cohorts = append(cohorts, cd)
		}
	}

	return c.JSON(cohorts)
}

// GetWarRoom returns the war room dashboard data.
// GET /api/v1/admin/war-room
func (h *AdminAnalyticsHandler) GetWarRoom(c *fiber.Ctx) error {
	data := entities.WarRoomData{}

	if h.pool != nil {
		var mrr float64
		_ = h.pool.QueryRow(c.Context(),
			`SELECT COALESCE(SUM(price_cents), 0) / 100.0 FROM subscriptions WHERE status = 'active'`,
		).Scan(&mrr)

		var totalCust, activeCust, newSignups int
		_ = h.pool.QueryRow(c.Context(), "SELECT COUNT(*) FROM users WHERE role = 'customer'").Scan(&totalCust)
		_ = h.pool.QueryRow(c.Context(),
			"SELECT COUNT(*) FROM users WHERE role = 'customer' AND updated_at > now() - interval '7 days'",
		).Scan(&activeCust)
		_ = h.pool.QueryRow(c.Context(),
			"SELECT COUNT(*) FROM users WHERE role = 'customer' AND created_at > date_trunc('month', now())",
		).Scan(&newSignups)

		var openTickets int
		_ = h.pool.QueryRow(c.Context(),
			"SELECT COUNT(*) FROM support_tickets WHERE status IN ('open', 'in-progress')",
		).Scan(&openTickets)

		data.KPIs = entities.WarRoomKPIs{
			MRR:             mrr,
			ActiveCustomers: activeCust,
			TotalCustomers:  totalCust,
			NewSignups:      newSignups,
			HealthScore:     85, // placeholder — computed from service health
		}

		// Active tickets
		ticketRows, _ := h.pool.Query(c.Context(),
			`SELECT id, subject, priority, status, created_at FROM support_tickets
			 WHERE status IN ('open', 'in-progress')
			 ORDER BY CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END
			 LIMIT 5`,
		)
		if ticketRows != nil {
			defer ticketRows.Close()
			for ticketRows.Next() {
				var t entities.TicketSummary
				var createdAt time.Time
				if err := ticketRows.Scan(&t.ID, &t.Subject, &t.Priority, &t.Status, &createdAt); err == nil {
					t.Age = time.Since(createdAt).Truncate(time.Hour).String()
					data.ActiveTickets = append(data.ActiveTickets, t)
				}
			}
		}
	}

	return c.JSON(data)
}
