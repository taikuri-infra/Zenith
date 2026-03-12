package entities

import "time"

// EmailSend tracks a drip campaign email sent to a user.
type EmailSend struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	TemplateKey string     `json:"template_key"`
	SentAt      time.Time  `json:"sent_at"`
	OpenedAt    *time.Time `json:"opened_at,omitempty"`
	ClickedAt   *time.Time `json:"clicked_at,omitempty"`
}

// Email template constants.
const (
	EmailWelcome         = "welcome"
	EmailDay1Deploy      = "day1_deploy"
	EmailDay3Engage      = "day3_engage"
	EmailDay3Nudge       = "day3_nudge"
	EmailDay7Trial       = "day7_trial"
	EmailDay14Value      = "day14_value"
	EmailDay30Dormant    = "day30_dormant"
	EmailUpgradeCongrats = "upgrade_congrats"
	EmailTrialEnding     = "trial_ending"
	EmailTrialExpired    = "trial_expired"
	EmailChurnWinback    = "churn_winback"
)

// EmailStats aggregates email campaign metrics.
type EmailStats struct {
	Sent       int            `json:"sent"`
	Opened     int            `json:"opened"`
	Clicked    int            `json:"clicked"`
	ByTemplate map[string]int `json:"by_template"`
}
