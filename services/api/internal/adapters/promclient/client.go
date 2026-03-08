package promclient

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

// Client wraps the Prometheus HTTP API for instant and range queries.
type Client struct {
	baseURL string
	http    *http.Client
}

// New creates a Prometheus query client.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// DataPoint represents a single time-series data point.
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// queryResponse models the Prometheus API JSON envelope.
type queryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string            `json:"resultType"`
		Result     json.RawMessage   `json:"result"`
	} `json:"data"`
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"errorType,omitempty"`
}

type vectorResult struct {
	Metric map[string]string `json:"metric"`
	Value  [2]json.RawMessage `json:"value"` // [timestamp, "value"]
}

type matrixResult struct {
	Metric map[string]string   `json:"metric"`
	Values [][2]json.RawMessage `json:"values"` // [[timestamp, "value"], ...]
}

// QueryInstant executes a PromQL instant query and returns the scalar sum of all results.
func (c *Client) QueryInstant(ctx context.Context, promQL string) (float64, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", c.baseURL, url.QueryEscape(promQL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, fmt.Errorf("promclient: build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("promclient: query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("promclient: read body: %w", err)
	}

	var qr queryResponse
	if err := json.Unmarshal(body, &qr); err != nil {
		return 0, fmt.Errorf("promclient: parse response: %w", err)
	}
	if qr.Status != "success" {
		return 0, fmt.Errorf("promclient: query failed: %s (%s)", qr.Error, qr.ErrorType)
	}

	if qr.Data.ResultType == "vector" {
		var results []vectorResult
		if err := json.Unmarshal(qr.Data.Result, &results); err != nil {
			return 0, fmt.Errorf("promclient: parse vector: %w", err)
		}
		var sum float64
		for _, r := range results {
			v, _ := strconv.ParseFloat(string(trimQuotes(r.Value[1])), 64)
			sum += v
		}
		return sum, nil
	}

	return 0, nil
}

// QueryRange executes a PromQL range query and returns aggregated data points.
func (c *Client) QueryRange(ctx context.Context, promQL string, start, end time.Time, step time.Duration) ([]DataPoint, error) {
	u := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%d",
		c.baseURL,
		url.QueryEscape(promQL),
		start.Unix(),
		end.Unix(),
		int(step.Seconds()),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("promclient: build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("promclient: query_range: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("promclient: read body: %w", err)
	}

	var qr queryResponse
	if err := json.Unmarshal(body, &qr); err != nil {
		return nil, fmt.Errorf("promclient: parse response: %w", err)
	}
	if qr.Status != "success" {
		return nil, fmt.Errorf("promclient: query_range failed: %s (%s)", qr.Error, qr.ErrorType)
	}

	if qr.Data.ResultType != "matrix" {
		return nil, nil
	}

	var results []matrixResult
	if err := json.Unmarshal(qr.Data.Result, &results); err != nil {
		return nil, fmt.Errorf("promclient: parse matrix: %w", err)
	}

	// Aggregate across all series by timestamp
	tsMap := make(map[int64]float64)
	var timestamps []int64
	for _, r := range results {
		for _, v := range r.Values {
			ts, _ := strconv.ParseFloat(string(trimQuotes(v[0])), 64)
			val, _ := strconv.ParseFloat(string(trimQuotes(v[1])), 64)
			tsInt := int64(ts)
			if _, exists := tsMap[tsInt]; !exists {
				timestamps = append(timestamps, tsInt)
			}
			tsMap[tsInt] += val
		}
	}

	points := make([]DataPoint, 0, len(timestamps))
	for _, ts := range timestamps {
		points = append(points, DataPoint{
			Timestamp: time.Unix(ts, 0).UTC(),
			Value:     tsMap[ts],
		})
	}

	return points, nil
}

// trimQuotes removes surrounding quotes from a JSON raw message byte slice.
func trimQuotes(b json.RawMessage) []byte {
	s := []byte(b)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
