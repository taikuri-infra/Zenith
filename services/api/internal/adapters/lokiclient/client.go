package lokiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client wraps the Loki HTTP API for log queries.
type Client struct {
	baseURL string
	http    *http.Client
}

// New creates a Loki query client.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// LogEntry represents a single log line.
type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Line      string            `json:"line"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// lokiResponse models the Loki API JSON envelope.
type lokiResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string            `json:"resultType"`
		Result     json.RawMessage   `json:"result"`
	} `json:"data"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"` // [["timestamp_ns", "line"], ...]
}

// QueryRange queries Loki for log entries matching logQL within the time range.
func (c *Client) QueryRange(ctx context.Context, logQL string, start, end time.Time, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	u := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&start=%d&end=%d&limit=%d&direction=backward",
		c.baseURL,
		url.QueryEscape(logQL),
		start.UnixNano(),
		end.UnixNano(),
		limit,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("lokiclient: build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lokiclient: query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lokiclient: read body: %w", err)
	}

	var lr lokiResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, fmt.Errorf("lokiclient: parse response: %w", err)
	}
	if lr.Status != "success" {
		return nil, fmt.Errorf("lokiclient: query failed (status=%s)", lr.Status)
	}

	var streams []lokiStream
	if err := json.Unmarshal(lr.Data.Result, &streams); err != nil {
		return nil, fmt.Errorf("lokiclient: parse streams: %w", err)
	}

	var entries []LogEntry
	for _, s := range streams {
		for _, v := range s.Values {
			tsNano, _ := strconv.ParseInt(v[0], 10, 64)
			entries = append(entries, LogEntry{
				Timestamp: time.Unix(0, tsNano).UTC(),
				Line:      v[1],
				Labels:    s.Stream,
			})
		}
	}

	return entries, nil
}

// Tail polls Loki at intervals and sends new entries to the out channel.
// It runs until the context is cancelled.
func (c *Client) Tail(ctx context.Context, logQL string, out chan<- LogEntry) error {
	start := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			end := time.Now()
			entries, err := c.QueryRange(ctx, logQL, start, end, 50)
			if err != nil {
				continue // transient errors — keep tailing
			}
			// Send entries in chronological order (QueryRange returns backward)
			for i := len(entries) - 1; i >= 0; i-- {
				select {
				case out <- entries[i]:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			start = end
		}
	}
}
