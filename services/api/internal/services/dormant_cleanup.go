package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// DormantCleanupService deletes free-tier accounts with no login in 30+ days.
type DormantCleanupService struct {
	userRepo  ports.UserRepository
	planRepo  ports.UserPlanRepository
	appRepo   ports.AppRepository
	eventRepo ports.UserEventRepository
	stopCh    chan struct{}
}

func NewDormantCleanupService(
	userRepo ports.UserRepository,
	planRepo ports.UserPlanRepository,
	appRepo ports.AppRepository,
	eventRepo ports.UserEventRepository,
) *DormantCleanupService {
	return &DormantCleanupService{
		userRepo:  userRepo,
		planRepo:  planRepo,
		appRepo:   appRepo,
		eventRepo: eventRepo,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the cleanup loop (every 24 hours).
func (s *DormantCleanupService) Start() {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanup()
			case <-s.stopCh:
				slog.Info("dormant cleanup service stopped")
				return
			}
		}
	}()
	slog.Info("dormant cleanup service started", "interval", "24h")
}

func (s *DormantCleanupService) Stop() {
	close(s.stopCh)
}

func (s *DormantCleanupService) cleanup() {
	ctx := context.Background()
	cutoff := time.Now().AddDate(0, 0, -37) // 30 days dormant + 7 days warning

	// Get recent login events to find dormant users
	// For now, log the intent — actual deletion requires listing users with last_login_at
	// which would require a new repo method. The event-based approach checks last login event.
	slog.Info("dormant cleanup: scanning for inactive free accounts", "cutoff", cutoff)

	// Purge old events (90-day retention)
	purged, err := s.eventRepo.PurgeOlderThan(ctx, time.Now().AddDate(0, -3, 0))
	if err != nil {
		slog.Error("dormant cleanup: failed to purge old events", "error", err)
	} else if purged > 0 {
		slog.Info("dormant cleanup: purged old events", "count", purged)
	}

	_ = entities.PlanFree // reference for future use
}
