package temporal

import (
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// WorkflowPlanOrchestrator is the workflow type name for plan changes.
	WorkflowPlanOrchestrator = "PlanOrchestratorWorkflow"
)

// PlanChangeInput is the workflow input for plan orchestration.
type PlanChangeInput struct {
	UserID             string            `json:"userId"`
	UserEmail          string            `json:"userEmail"`
	OldTier            entities.PlanTier `json:"oldTier"`
	NewTier            entities.PlanTier `json:"newTier"`
	StripeSubscription string            `json:"stripeSubscription"`
	StripeCustomer     string            `json:"stripeCustomer"`
}

// PlanOrchestratorWorkflow orchestrates all steps of a plan upgrade or downgrade.
// Each activity is idempotent; Temporal handles retries automatically.
func PlanOrchestratorWorkflow(ctx workflow.Context, input PlanChangeInput) error {
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    1 * time.Minute,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	isUpgrade := tierRank(input.NewTier) > tierRank(input.OldTier)

	if isUpgrade {
		return planUpgradeWorkflow(ctx, input)
	}
	return planDowngradeWorkflow(ctx, input)
}

func planUpgradeWorkflow(ctx workflow.Context, input PlanChangeInput) error {
	// Step 1: Verify payment with Stripe
	if err := workflow.ExecuteActivity(ctx, (*PlanActivities).VerifyPayment, input).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*PlanActivities).NotifyPlanChangeFailed, input, err.Error()).Get(ctx, nil)
		return fmt.Errorf("verify payment: %w", err)
	}

	// Step 2: Update plan in database
	if err := workflow.ExecuteActivity(ctx, (*PlanActivities).UpdatePlanDB, input).Get(ctx, nil); err != nil {
		return fmt.Errorf("update plan db: %w", err)
	}

	// Step 3: Provision tier-specific infrastructure (Business+: dedicated namespace)
	if input.NewTier == entities.PlanBusiness || input.NewTier == entities.PlanEnterprise {
		if err := workflow.ExecuteActivity(ctx, (*PlanActivities).ProvisionDedicatedInfra, input).Get(ctx, nil); err != nil {
			// Non-fatal — user already has the plan, infra will be retried
			_ = workflow.ExecuteActivity(ctx, (*PlanActivities).AuditLog, input, fmt.Sprintf("infrastructure provisioning failed: %v", err)).Get(ctx, nil)
		}
	}

	// Step 4: Update gateway rate limits
	if err := workflow.ExecuteActivity(ctx, (*PlanActivities).UpdateGatewayLimits, input).Get(ctx, nil); err != nil {
		// Non-fatal
		_ = workflow.ExecuteActivity(ctx, (*PlanActivities).AuditLog, input, fmt.Sprintf("gateway limit update failed: %v", err)).Get(ctx, nil)
	}

	// Step 5: Enable tier features (SSO, audit log, WAF, etc.)
	if err := workflow.ExecuteActivity(ctx, (*PlanActivities).EnableFeatures, input).Get(ctx, nil); err != nil {
		// Non-fatal — features are plan-gated at the API level anyway
		_ = workflow.ExecuteActivity(ctx, (*PlanActivities).AuditLog, input, fmt.Sprintf("feature enable failed: %v", err)).Get(ctx, nil)
	}

	// Step 6: Notify user
	if err := workflow.ExecuteActivity(ctx, (*PlanActivities).NotifyPlanChanged, input).Get(ctx, nil); err != nil {
		// Non-fatal — notification failure shouldn't block upgrade
		_ = workflow.ExecuteActivity(ctx, (*PlanActivities).AuditLog, input, fmt.Sprintf("notification failed: %v", err)).Get(ctx, nil)
	}

	// Step 7: Audit log
	_ = workflow.ExecuteActivity(ctx, (*PlanActivities).AuditLog, input, fmt.Sprintf("Plan upgraded from %s to %s", input.OldTier, input.NewTier)).Get(ctx, nil)

	return nil
}

func planDowngradeWorkflow(ctx workflow.Context, input PlanChangeInput) error {
	// Step 1: Check resource usage (verify user is under new tier limits)
	if err := workflow.ExecuteActivity(ctx, (*PlanActivities).CheckResourceUsage, input).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*PlanActivities).NotifyPlanChangeFailed, input, err.Error()).Get(ctx, nil)
		return fmt.Errorf("resource check failed: %w", err)
	}

	// Step 2: Update plan in database
	if err := workflow.ExecuteActivity(ctx, (*PlanActivities).UpdatePlanDB, input).Get(ctx, nil); err != nil {
		return fmt.Errorf("update plan db: %w", err)
	}

	// Step 3: Disable features not available on new tier
	if err := workflow.ExecuteActivity(ctx, (*PlanActivities).DisableFeatures, input).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*PlanActivities).AuditLog, input, fmt.Sprintf("feature disable failed: %v", err)).Get(ctx, nil)
	}

	// Step 4: Update gateway rate limits (lower limits)
	if err := workflow.ExecuteActivity(ctx, (*PlanActivities).UpdateGatewayLimits, input).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*PlanActivities).AuditLog, input, fmt.Sprintf("gateway limit update failed: %v", err)).Get(ctx, nil)
	}

	// Step 5: Migrate from dedicated to shared namespace if stepping down from Business+
	if (input.OldTier == entities.PlanBusiness || input.OldTier == entities.PlanEnterprise) &&
		input.NewTier != entities.PlanBusiness && input.NewTier != entities.PlanEnterprise {
		if err := workflow.ExecuteActivity(ctx, (*PlanActivities).MigrateToSharedInfra, input).Get(ctx, nil); err != nil {
			_ = workflow.ExecuteActivity(ctx, (*PlanActivities).AuditLog, input, fmt.Sprintf("migration to shared infra failed: %v", err)).Get(ctx, nil)
		}
	}

	// Step 6: Notify user
	if err := workflow.ExecuteActivity(ctx, (*PlanActivities).NotifyPlanChanged, input).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*PlanActivities).AuditLog, input, fmt.Sprintf("notification failed: %v", err)).Get(ctx, nil)
	}

	// Step 7: Audit log
	_ = workflow.ExecuteActivity(ctx, (*PlanActivities).AuditLog, input, fmt.Sprintf("Plan downgraded from %s to %s", input.OldTier, input.NewTier)).Get(ctx, nil)

	return nil
}

// tierRank returns numeric rank for plan comparison (higher = better tier).
func tierRank(tier entities.PlanTier) int {
	switch tier {
	case entities.PlanFree:
		return 0
	case entities.PlanPro:
		return 1
	case entities.PlanTeam:
		return 2
	case entities.PlanBusiness:
		return 3
	case entities.PlanEnterprise:
		return 4
	default:
		return 0
	}
}
