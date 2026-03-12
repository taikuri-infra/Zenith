package memory

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestExitSurveyCreate(t *testing.T) {
	repo := NewMemoryExitSurveyRepository()
	ctx := context.Background()

	survey := &entities.ExitSurvey{
		UserID:   "user-1",
		Reason:   entities.ExitReasonTooExpensive,
		Details:  "pricing is too high",
		PlanTier: "pro",
	}

	err := repo.Create(ctx, survey)
	if err != nil {
		t.Fatalf("Create: expected no error, got %v", err)
	}
	if survey.ID == "" {
		t.Error("Create: expected ID to be auto-generated")
	}
	if survey.CreatedAt.IsZero() {
		t.Error("Create: expected CreatedAt to be set")
	}
}

func TestExitSurveyList(t *testing.T) {
	repo := NewMemoryExitSurveyRepository()
	ctx := context.Background()

	repo.Create(ctx, &entities.ExitSurvey{UserID: "u1", Reason: entities.ExitReasonTooExpensive, PlanTier: "pro"})
	repo.Create(ctx, &entities.ExitSurvey{UserID: "u2", Reason: entities.ExitReasonMissingFeatures, PlanTier: "pro"})
	repo.Create(ctx, &entities.ExitSurvey{UserID: "u3", Reason: entities.ExitReasonTooExpensive, PlanTier: "team"})

	surveys, err := repo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(surveys) != 3 {
		t.Errorf("List: expected 3 surveys, got %d", len(surveys))
	}

	// Test limit
	surveys, err = repo.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List limit: %v", err)
	}
	if len(surveys) != 2 {
		t.Errorf("List limit: expected 2, got %d", len(surveys))
	}

	// Test offset
	surveys, err = repo.List(ctx, 10, 2)
	if err != nil {
		t.Fatalf("List offset: %v", err)
	}
	if len(surveys) != 1 {
		t.Errorf("List offset: expected 1, got %d", len(surveys))
	}
}

func TestExitSurveyGetStats(t *testing.T) {
	repo := NewMemoryExitSurveyRepository()
	ctx := context.Background()

	repo.Create(ctx, &entities.ExitSurvey{UserID: "u1", Reason: entities.ExitReasonTooExpensive, PlanTier: "pro"})
	repo.Create(ctx, &entities.ExitSurvey{UserID: "u2", Reason: entities.ExitReasonTooExpensive, PlanTier: "pro"})
	repo.Create(ctx, &entities.ExitSurvey{UserID: "u3", Reason: entities.ExitReasonMissingFeatures, PlanTier: "team"})

	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.Total != 3 {
		t.Errorf("GetStats Total: expected 3, got %d", stats.Total)
	}
	if stats.ByReason[entities.ExitReasonTooExpensive] != 2 {
		t.Errorf("GetStats too_expensive: expected 2, got %d", stats.ByReason[entities.ExitReasonTooExpensive])
	}
	if stats.ByReason[entities.ExitReasonMissingFeatures] != 1 {
		t.Errorf("GetStats missing_features: expected 1, got %d", stats.ByReason[entities.ExitReasonMissingFeatures])
	}
}
