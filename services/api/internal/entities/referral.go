package entities

import "time"

// ReferralReward tracks a referral bonus between users.
type ReferralReward struct {
	ID           string     `json:"id"`
	ReferrerID   string     `json:"referrer_id"`
	ReferredID   string     `json:"referred_id"`
	Status       string     `json:"status"`
	RewardType   string     `json:"reward_type"`
	RewardAmount int        `json:"reward_amount"`
	CreditedAt   *time.Time `json:"credited_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Referral reward statuses.
const (
	ReferralPending  = "pending"
	ReferralCredited = "credited"
	ReferralExpired  = "expired"
)

// ReferralSummary is the user-facing referral dashboard data.
type ReferralSummary struct {
	Code           string `json:"code"`
	Link           string `json:"link"`
	TotalReferrals int    `json:"total_referrals"`
	Credited       int    `json:"credited"`
	Pending        int    `json:"pending"`
}
