package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// ReferralHandler manages referral endpoints.
type ReferralHandler struct {
	referralRepo ports.ReferralRepository
	eventRepo    ports.UserEventRepository
	baseURL      string
}

func NewReferralHandler(referralRepo ports.ReferralRepository, eventRepo ports.UserEventRepository, baseURL string) *ReferralHandler {
	return &ReferralHandler{referralRepo: referralRepo, eventRepo: eventRepo, baseURL: baseURL}
}

// GetSummary returns the current user's referral dashboard.
// GET /api/v1/referral
func (h *ReferralHandler) GetSummary(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	summary, err := h.referralRepo.GetSummary(c.Context(), userID, h.baseURL)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get referral summary")
	}
	return c.JSON(summary)
}

// ListRewards returns the current user's referral rewards.
// GET /api/v1/referral/rewards
func (h *ReferralHandler) ListRewards(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	rewards, err := h.referralRepo.ListByReferrer(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list referral rewards")
	}
	if rewards == nil {
		rewards = []entities.ReferralReward{}
	}
	return c.JSON(fiber.Map{"items": rewards, "total": len(rewards)})
}

// TrackShare records a share event for analytics.
// POST /api/v1/referral/share
func (h *ReferralHandler) TrackShare(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	if h.eventRepo != nil {
		go h.eventRepo.Track(c.Context(), &entities.UserEvent{
			UserID:    userID,
			EventType: entities.EventReferralShare,
		})
	}
	return c.JSON(fiber.Map{"message": "share tracked"})
}

// AdminList returns all referral rewards for admin.
// GET /api/v1/admin/referrals
func (h *ReferralHandler) AdminList(c *fiber.Ctx) error {
	rewards, err := h.referralRepo.ListAll(c.Context(), 100, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list referrals")
	}
	if rewards == nil {
		rewards = []entities.ReferralReward{}
	}
	return c.JSON(fiber.Map{"items": rewards, "total": len(rewards)})
}
