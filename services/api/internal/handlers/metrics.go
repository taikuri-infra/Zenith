package handlers

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// BusinessMetrics exposes Prometheus metrics for business dashboards.
type BusinessMetrics struct {
	userRepo ports.UserRepository
	appRepo  ports.AppRepository
	dbRepo   ports.DatabaseRepository
	planRepo ports.UserPlanRepository

	mrrEuros          prometheus.Gauge
	totalUsers        prometheus.Gauge
	payingUsers       prometheus.Gauge
	totalApps         prometheus.Gauge
	totalDatabases    prometheus.Gauge
	usersByPlan       *prometheus.GaugeVec
	churnRate30d      prometheus.Gauge
	stripePaySuccess  prometheus.Counter
	stripePayFailed   prometheus.Counter

	registry *prometheus.Registry
	once     sync.Once
}

// NewBusinessMetrics creates a new BusinessMetrics exporter.
func NewBusinessMetrics(
	userRepo ports.UserRepository,
	appRepo ports.AppRepository,
	dbRepo ports.DatabaseRepository,
	planRepo ports.UserPlanRepository,
) *BusinessMetrics {
	reg := prometheus.NewRegistry()

	// Also register the default Go + process collectors
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	bm := &BusinessMetrics{
		userRepo: userRepo,
		appRepo:  appRepo,
		dbRepo:   dbRepo,
		planRepo: planRepo,
		registry: reg,

		mrrEuros: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zenith_mrr_euros",
			Help: "Monthly recurring revenue in EUR",
		}),
		totalUsers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zenith_total_users",
			Help: "Total registered users",
		}),
		payingUsers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zenith_paying_users",
			Help: "Total users on paid plans",
		}),
		totalApps: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zenith_total_apps",
			Help: "Total deployed apps",
		}),
		totalDatabases: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zenith_total_databases",
			Help: "Total managed databases",
		}),
		usersByPlan: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "zenith_users_by_plan",
			Help: "User count by plan tier",
		}, []string{"plan"}),
		churnRate30d: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zenith_churn_rate_30d",
			Help: "30-day churn rate (0.0-1.0)",
		}),
		stripePaySuccess: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "zenith_stripe_payment_succeeded_total",
			Help: "Total successful Stripe payments",
		}),
		stripePayFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "zenith_stripe_payment_failed_total",
			Help: "Total failed Stripe payments",
		}),
	}

	reg.MustRegister(
		bm.mrrEuros,
		bm.totalUsers,
		bm.payingUsers,
		bm.totalApps,
		bm.totalDatabases,
		bm.usersByPlan,
		bm.churnRate30d,
		bm.stripePaySuccess,
		bm.stripePayFailed,
	)

	return bm
}

// RecordPaymentSuccess increments the successful payment counter.
func (bm *BusinessMetrics) RecordPaymentSuccess() {
	bm.stripePaySuccess.Inc()
}

// RecordPaymentFailure increments the failed payment counter.
func (bm *BusinessMetrics) RecordPaymentFailure() {
	bm.stripePayFailed.Inc()
}

// StartCollector begins periodic metric collection in the background.
func (bm *BusinessMetrics) StartCollector(ctx context.Context) {
	bm.once.Do(func() {
		go func() {
			// Collect immediately on startup
			bm.collect(ctx)

			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					bm.collect(ctx)
				}
			}
		}()
	})
}

// collect queries all repos and updates gauges.
func (bm *BusinessMetrics) collect(ctx context.Context) {
	timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Total users
	if users, err := bm.userRepo.Count(timeout); err == nil {
		bm.totalUsers.Set(float64(users))
	} else {
		slog.Warn("metrics: count users", "error", err)
	}

	// Total apps
	if apps, err := bm.appRepo.CountApps(timeout); err == nil {
		bm.totalApps.Set(float64(apps))
	} else {
		slog.Warn("metrics: count apps", "error", err)
	}

	// Total databases
	if dbs, err := bm.dbRepo.CountDatabases(timeout); err == nil {
		bm.totalDatabases.Set(float64(dbs))
	} else {
		slog.Warn("metrics: count databases", "error", err)
	}

	// Plan distribution + MRR + paying users
	if plans, err := bm.planRepo.ListAllPlans(timeout); err == nil {
		planCounts := map[string]int{"free": 0, "pro": 0, "team": 0, "business": 0, "enterprise": 0}
		var mrrCents int64
		paying := 0
		for _, p := range plans {
			tier := string(p.Tier)
			planCounts[tier]++
			switch p.Tier {
			case "pro":
				mrrCents += 2900
				paying++
			case "team":
				mrrCents += 19900
				paying++
			case "business":
				mrrCents += 49900
				paying++
			case "enterprise":
				mrrCents += 99900
				paying++
			}
		}
		for tier, count := range planCounts {
			bm.usersByPlan.WithLabelValues(tier).Set(float64(count))
		}
		bm.mrrEuros.Set(float64(mrrCents) / 100.0)
		bm.payingUsers.Set(float64(paying))
	} else {
		slog.Warn("metrics: list plans", "error", err)
	}
}

// Handler returns the Fiber handler for the /metrics endpoint.
func (bm *BusinessMetrics) Handler() fiber.Handler {
	handler := promhttp.HandlerFor(bm.registry, promhttp.HandlerOpts{})
	return adaptor.HTTPHandler(handler)
}
