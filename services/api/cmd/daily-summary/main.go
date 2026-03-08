// daily-summary queries Prometheus for Zenith business metrics and posts
// a summary to Telegram. Designed to run as a CronJob in Kubernetes.
//
// Required env vars:
//   PROMETHEUS_URL     — Prometheus base URL (default: in-cluster)
//   TELEGRAM_BOT_TOKEN — Telegram bot API token
//   TELEGRAM_CHAT_ID   — Telegram chat ID to post to
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/promclient"
)

func main() {
	promURL := envOrDefault("PROMETHEUS_URL", "http://kube-prometheus-stack-prometheus.monitoring.svc.cluster.local:9090")
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if botToken == "" || chatID == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prom := promclient.New(promURL)

	// Gather metrics
	mrr, _ := prom.QueryInstant(ctx, "zenith_mrr_euros")
	totalUsers, _ := prom.QueryInstant(ctx, "zenith_total_users")
	payingUsers, _ := prom.QueryInstant(ctx, "zenith_paying_users")
	totalApps, _ := prom.QueryInstant(ctx, "zenith_total_apps")
	totalDBs, _ := prom.QueryInstant(ctx, "zenith_total_databases")
	newUsers24h, _ := prom.QueryInstant(ctx, "increase(zenith_total_users[24h])")
	paySuccess24h, _ := prom.QueryInstant(ctx, "increase(zenith_stripe_payment_succeeded_total[24h])")
	payFailed24h, _ := prom.QueryInstant(ctx, "increase(zenith_stripe_payment_failed_total[24h])")
	apiErrorRate, _ := prom.QueryInstant(ctx, `sum(rate(http_requests_total{namespace="zenith-platform",status=~"5.."}[24h])) / clamp_min(sum(rate(http_requests_total{namespace="zenith-platform"}[24h])), 0.001) * 100`)
	buildSuccess24h, _ := prom.QueryInstant(ctx, "increase(kube_job_status_succeeded{namespace=\"zenith-builds\"}[24h])")
	buildFailed24h, _ := prom.QueryInstant(ctx, "increase(kube_job_status_failed{namespace=\"zenith-builds\"}[24h])")

	now := time.Now().UTC().Format("2006-01-02")
	conversion := float64(0)
	if totalUsers > 0 {
		conversion = payingUsers / totalUsers * 100
	}

	msg := fmt.Sprintf(`📊 *Zenith Daily Summary — %s*

💰 *Revenue*
  MRR: €%.2f
  Payments (24h): %.0f succeeded, %.0f failed

👥 *Users*
  Total: %.0f
  Paying: %.0f (%.1f%% conversion)
  New signups (24h): %.0f

🚀 *Platform*
  Apps deployed: %.0f
  Databases: %.0f
  Builds (24h): %.0f succeeded, %.0f failed
  API error rate (24h): %.2f%%

_Sent by Zenith Platform Bot_`,
		now,
		mrr, paySuccess24h, payFailed24h,
		totalUsers, payingUsers, conversion, newUsers24h,
		totalApps, totalDBs,
		buildSuccess24h, buildFailed24h, apiErrorRate,
	)

	if err := sendTelegram(botToken, chatID, msg); err != nil {
		log.Fatalf("Failed to send Telegram message: %v", err)
	}

	log.Println("Daily summary sent successfully")
}

func sendTelegram(botToken, chatID, text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	resp, err := http.PostForm(apiURL, url.Values{
		"chat_id":    {chatID},
		"text":       {text},
		"parse_mode": {"Markdown"},
	})
	if err != nil {
		return fmt.Errorf("telegram request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
