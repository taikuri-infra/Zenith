package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Options controls health check polling behaviour.
type Options struct {
	// URL to GET — 200 is considered healthy.
	URL string
	// Interval between poll attempts.
	Interval time.Duration
	// RequestTimeout per individual HTTP GET.
	RequestTimeout time.Duration
}

// WaitUntilHealthy polls URL until it returns HTTP 200 or ctx is cancelled.
func WaitUntilHealthy(ctx context.Context, opts Options) error {
	if opts.Interval == 0 {
		opts.Interval = 5 * time.Second
	}
	if opts.RequestTimeout == 0 {
		opts.RequestTimeout = 10 * time.Second
	}

	client := &http.Client{Timeout: opts.RequestTimeout}

	// Try immediately first
	if err := check(ctx, client, opts.URL); err == nil {
		return nil
	}

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check timed out waiting for %s: %w", opts.URL, ctx.Err())
		case <-ticker.C:
			if err := check(ctx, client, opts.URL); err == nil {
				return nil
			}
		}
	}
}

func check(ctx context.Context, client *http.Client, url string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned %d", resp.StatusCode)
	}
	return nil
}

// Probe performs a single health check and returns true if healthy.
func Probe(url string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
