package entities

import "time"

// UserEvent represents a tracked user action for analytics.
type UserEvent struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	EventType  string                 `json:"event_type"`
	Properties map[string]interface{} `json:"properties"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// Event type constants.
const (
	EventSignup          = "signup"
	EventLogin           = "login"
	EventAppCreate       = "app.create"
	EventAppDeploy       = "app.deploy"
	EventAppDelete       = "app.delete"
	EventDBCreate        = "db.create"
	EventDomainAdd       = "domain.add"
	EventBucketCreate    = "bucket.create"
	EventUpgradeStart    = "upgrade.start"
	EventUpgradeComplete = "upgrade.complete"
	EventPlanCancel      = "plan.cancel"
	EventOnboardingStep  = "onboarding.step"
	EventOnboardingDone  = "onboarding.done"
	EventReferralShare   = "referral.share"
	EventReferralSignup  = "referral.signup"
	EventFeatureGated    = "feature.gated"
	EventTrialStart      = "trial.start"
	EventTrialEnd        = "trial.end"
)
