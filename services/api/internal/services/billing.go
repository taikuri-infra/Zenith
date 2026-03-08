package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/adapters/harborclient"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// PriceForTier returns (priceCents, priceID) for a given tier.
func PriceForTier(tier entities.PlanTier, proPriceID, teamPriceID, businessPriceID string) (int, string) {
	switch tier {
	case entities.PlanPro:
		return 2900, proPriceID
	case entities.PlanTeam:
		return 9900, teamPriceID
	case entities.PlanBusiness:
		return 14900, businessPriceID
	default:
		return 0, ""
	}
}

// BillingService handles billing business logic.
type BillingService struct {
	payments    ports.PaymentGateway
	billingRepo ports.BillingRepository
	planRepo    ports.UserPlanRepository
	appRepo     ports.AppRepository
	dbRepo      ports.DatabaseRepository
	storageRepo ports.StorageRepository
	authRepo    ports.AppAuthRepository
	proPriceID      string
	teamPriceID     string
	businessPriceID string
	baseDomain      string
	harbor          *harborclient.Client // optional: Pro+ user project creation
}

// NewBillingService creates a new BillingService.
func NewBillingService(
	payments ports.PaymentGateway,
	billingRepo ports.BillingRepository,
	planRepo ports.UserPlanRepository,
	appRepo ports.AppRepository,
	dbRepo ports.DatabaseRepository,
	storageRepo ports.StorageRepository,
	authRepo ports.AppAuthRepository,
	proPriceID, teamPriceID, businessPriceID, baseDomain string,
) *BillingService {
	return &BillingService{
		payments:        payments,
		billingRepo:     billingRepo,
		planRepo:        planRepo,
		appRepo:         appRepo,
		dbRepo:          dbRepo,
		storageRepo:     storageRepo,
		authRepo:        authRepo,
		proPriceID:      proPriceID,
		teamPriceID:     teamPriceID,
		businessPriceID: businessPriceID,
		baseDomain:      baseDomain,
	}
}

// GetBillingStatus returns the user's plan, subscription, and usage info.
func (s *BillingService) GetBillingStatus(ctx context.Context, userID string) (*dto.BillingStatusResponse, error) {
	plan, err := s.planRepo.GetUserPlan(ctx, userID)
	if err != nil {
		return nil, err
	}

	usage := s.calculateUsage(ctx, userID)
	priceCents, _ := PriceForTier(plan.Tier, s.proPriceID, s.teamPriceID, s.businessPriceID)

	resp := &dto.BillingStatusResponse{
		Tier:          string(plan.Tier),
		BillingStatus: "none",
		PriceCents:    priceCents,
		Currency:      "eur",
		Limits:        plan.Limits,
		Usage:         usage,
		StripeEnabled: s.payments != nil,
	}

	sub, err := s.billingRepo.GetSubscriptionByUser(ctx, userID)
	if err == nil && sub != nil {
		resp.BillingStatus = string(sub.Status)
		resp.CancelAtPeriodEnd = sub.CancelAtPeriodEnd
		periodEnd := sub.CurrentPeriodEnd.Format("2006-01-02T15:04:05Z")
		resp.PeriodEnd = &periodEnd
	}

	return resp, nil
}

// CreateCheckoutSession creates a Stripe checkout session and returns the URL.
func (s *BillingService) CreateCheckoutSession(ctx context.Context, userID, userEmail, tierStr string) (*dto.CheckoutResponse, error) {
	if s.payments == nil {
		return nil, fmt.Errorf("Stripe billing is not enabled")
	}

	tier := entities.PlanTier(tierStr)
	_, priceID := PriceForTier(tier, s.proPriceID, s.teamPriceID, s.businessPriceID)
	if priceID == "" {
		return nil, fmt.Errorf("invalid tier or tier not available for checkout")
	}

	plan, _ := s.planRepo.GetUserPlan(ctx, userID)
	if plan != nil && plan.Tier == tier {
		return nil, fmt.Errorf("you are already on the %s plan", tierStr)
	}

	customerID, _ := s.billingRepo.GetStripeCustomerID(ctx, userID)

	successURL := "https://" + s.baseDomain + "/billing?success=true"
	cancelURL := "https://" + s.baseDomain + "/billing?canceled=true"

	result, err := s.payments.CreateCheckoutSession(ctx, ports.CheckoutParams{
		CustomerID: customerID,
		PriceID:    priceID,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
		UserEmail:  userEmail,
		Metadata: map[string]string{
			"user_id": userID,
			"tier":    tierStr,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session")
	}

	return &dto.CheckoutResponse{
		CheckoutURL: result.URL,
		SessionID:   result.SessionID,
	}, nil
}

// CreatePortalSession creates a Stripe customer portal session.
func (s *BillingService) CreatePortalSession(ctx context.Context, userID string) (*dto.PortalResponse, error) {
	if s.payments == nil {
		return nil, fmt.Errorf("Stripe billing is not enabled")
	}

	customerID, _ := s.billingRepo.GetStripeCustomerID(ctx, userID)
	if customerID == "" {
		return nil, fmt.Errorf("no Stripe customer found; subscribe first")
	}

	returnURL := "https://" + s.baseDomain + "/billing"

	result, err := s.payments.CreatePortalSession(ctx, customerID, returnURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create portal session")
	}

	return &dto.PortalResponse{PortalURL: result.URL}, nil
}

// CancelSubscription cancels the user's subscription.
func (s *BillingService) CancelSubscription(ctx context.Context, userID string, immediate bool) error {
	if s.payments == nil {
		return fmt.Errorf("Stripe billing is not enabled")
	}

	sub, err := s.billingRepo.GetSubscriptionByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("no active subscription found")
	}

	return s.payments.CancelSubscription(ctx, sub.StripeSubscriptionID, !immediate)
}

// ListInvoices returns the user's invoice history.
func (s *BillingService) ListInvoices(ctx context.Context, userID string) ([]dto.InvoiceResponse, error) {
	invoices, err := s.billingRepo.ListInvoicesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.InvoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		items = append(items, dto.InvoiceResponse{
			ID:          inv.ID,
			AmountCents: inv.AmountCents,
			Currency:    inv.Currency,
			Status:      string(inv.Status),
			InvoiceURL:  inv.InvoiceURL,
			InvoicePDF:  inv.InvoicePDF,
			PeriodStart: inv.PeriodStart.Format("2006-01-02T15:04:05Z"),
			PeriodEnd:   inv.PeriodEnd.Format("2006-01-02T15:04:05Z"),
			CreatedAt:   inv.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return items, nil
}

// GetAdminBillingOverview returns admin billing metrics.
func (s *BillingService) GetAdminBillingOverview(ctx context.Context) (*dto.AdminBillingOverviewResponse, error) {
	overview, err := s.billingRepo.GetBillingOverview(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.AdminBillingOverviewResponse{
		MRRCents:            overview.MRRCents,
		ActiveSubscriptions: overview.ActiveSubscriptions,
		PastDueCount:        overview.PastDueCount,
		CanceledThisMonth:   overview.CanceledThisMonth,
		ChurnRatePercent:    overview.ChurnRatePercent,
	}, nil
}

// ProPriceID returns the configured Pro price ID.
func (s *BillingService) ProPriceID() string { return s.proPriceID }

// TeamPriceID returns the configured Team price ID.
func (s *BillingService) TeamPriceID() string { return s.teamPriceID }

// BusinessPriceID returns the configured Business price ID.
func (s *BillingService) BusinessPriceID() string { return s.businessPriceID }

// BillingRepo exposes the billing repository for webhook handler usage.
func (s *BillingService) BillingRepo() ports.BillingRepository { return s.billingRepo }

// PlanRepo exposes the plan repository for webhook handler usage.
func (s *BillingService) PlanRepo() ports.UserPlanRepository { return s.planRepo }

// Payments exposes the payment gateway for webhook handler usage.
func (s *BillingService) Payments() ports.PaymentGateway { return s.payments }

// SetHarborClient configures the Harbor client for Pro+ project creation.
func (s *BillingService) SetHarborClient(h *harborclient.Client) {
	s.harbor = h
}

// ProvisionUpgradeResources handles post-upgrade provisioning for a user.
// S3 storage uses a shared platform bucket with prefix isolation — no per-user
// bucket creation needed.
func (s *BillingService) ProvisionUpgradeResources(ctx context.Context, userID string, tier entities.PlanTier) {
	if tier == entities.PlanFree {
		return
	}

	// S3: No-op. Storage uses a single shared bucket with prefix-based isolation.
	// User "buckets" are virtual (DB records only), created on-demand via the storage API.
	slog.Info("user upgraded, storage uses shared bucket", "user_id", userID, "tier", tier)

	// Create Harbor project for Pro+ users (private container registry per user)
	if tier == entities.PlanPro || tier == entities.PlanTeam || tier == entities.PlanBusiness {
		if s.harbor != nil {
			projectName := "user-" + userID
			// Storage quota: Pro=5GB, Team=20GB, Business=50GB
			var quota int64
			switch tier {
			case entities.PlanPro:
				quota = 5 * 1024 * 1024 * 1024
			case entities.PlanTeam:
				quota = 20 * 1024 * 1024 * 1024
			case entities.PlanBusiness:
				quota = 50 * 1024 * 1024 * 1024
			}
			if err := s.harbor.CreateProject(ctx, projectName, quota); err != nil {
				slog.Error("failed to create Harbor project", "user_id", userID, "error", err)
			} else {
				slog.Info("harbor project created", "user_id", userID, "project", projectName, "tier", tier)
			}
		} else {
			slog.Info("harbor not configured, skipping project creation", "user_id", userID, "tier", tier)
		}
	}
}

func (s *BillingService) calculateUsage(ctx context.Context, userID string) dto.PlanUsage {
	appCount, _ := s.appRepo.CountAppsByUser(ctx, userID)
	dbCount, _ := s.dbRepo.CountDatabasesByUser(ctx, userID)
	bucketCount, _ := s.storageRepo.CountBucketsByUser(ctx, userID)

	return dto.PlanUsage{
		Apps:      appCount,
		Databases: dbCount,
		Buckets:   bucketCount,
	}
}
