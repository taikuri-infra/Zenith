package entities

import "time"

// ExitSurvey captures feedback when a user cancels their subscription.
type ExitSurvey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	Details   string    `json:"details"`
	PlanTier  string    `json:"plan_tier"`
	CreatedAt time.Time `json:"created_at"`
}

// Exit survey reason constants.
const (
	ExitReasonTooExpensive    = "too_expensive"
	ExitReasonMissingFeatures = "missing_features"
	ExitReasonFoundAlternative = "found_alternative"
	ExitReasonNotUsing        = "not_using"
	ExitReasonTechnicalIssues = "technical_issues"
	ExitReasonTemporary       = "temporary"
	ExitReasonOther           = "other"
)

// ExitSurveyStats aggregates exit survey data.
type ExitSurveyStats struct {
	ByReason map[string]int `json:"by_reason"`
	Total    int            `json:"total"`
}
