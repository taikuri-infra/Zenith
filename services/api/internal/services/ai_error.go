package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/lokiclient"
)

// ErrorAnalysis holds the AI-generated error analysis result.
type ErrorAnalysis struct {
	Problem       string `json:"problem"`
	Cause         string `json:"cause"`
	Fix           string `json:"fix"`
	Confidence    string `json:"confidence"` // high, medium, low
	PIIDisclaimer string `json:"pii_disclaimer"`
}

// AIErrorAnalyzer uses AI to analyze application errors from logs.
type AIErrorAnalyzer struct {
	aiClient   *AIClient
	lokiClient *lokiclient.Client
}

// NewAIErrorAnalyzer creates a new AI error analyzer.
func NewAIErrorAnalyzer(aiClient *AIClient, lokiClient *lokiclient.Client) *AIErrorAnalyzer {
	return &AIErrorAnalyzer{
		aiClient:   aiClient,
		lokiClient: lokiClient,
	}
}

const errorAnalysisSystemPrompt = `You are a DevOps expert analyzing application error logs from a Kubernetes-hosted app.
Analyze the provided log lines and return a JSON object with these fields:
- "problem": A concise description of what went wrong (1-2 sentences)
- "cause": The most likely root cause (1-2 sentences)
- "fix": Actionable steps to fix the issue (2-3 bullet points as a single string)
- "confidence": Your confidence level: "high", "medium", or "low"

Return ONLY valid JSON, no markdown, no explanation.
Example: {"problem":"Application crashes on startup with OOM","cause":"Memory limit is too low for the Java heap size configured","fix":"1. Increase memory limit to at least 512Mi\n2. Set -Xmx to match container memory limit\n3. Add readiness probe to detect startup failures","confidence":"high"}`

// AnalyzeError fetches recent logs and sends them to AI for analysis.
func (a *AIErrorAnalyzer) AnalyzeError(ctx context.Context, appSlug, namespace string, logLines int) (*ErrorAnalysis, *AIResponse, error) {
	if a.aiClient == nil || !a.aiClient.IsEnabled() {
		return nil, nil, nil
	}

	if logLines <= 0 {
		logLines = 100
	}

	// Fetch recent logs from Loki
	var rawLogs string
	if a.lokiClient != nil {
		query := `{namespace="` + namespace + `", app="` + appSlug + `"}`
		end := time.Now()
		start := end.Add(-1 * time.Hour)
		entries, err := a.lokiClient.QueryRange(ctx, query, start, end, logLines)
		if err != nil {
			slog.Warn("ai_error: loki query failed", "error", err)
		} else {
			var lines []string
			for _, e := range entries {
				lines = append(lines, e.Line)
			}
			rawLogs = strings.Join(lines, "\n")
		}
	}

	if rawLogs == "" {
		return nil, nil, nil
	}

	// Scrub PII before sending to AI
	scrubbedLogs := ScrubPII(rawLogs)

	resp, err := a.aiClient.Complete(ctx, errorAnalysisSystemPrompt, scrubbedLogs)
	if err != nil || resp == nil {
		return nil, nil, nil
	}

	var analysis ErrorAnalysis
	if err := parseJSONResponse(resp.Content, &analysis); err != nil {
		slog.Warn("ai_error: failed to parse AI response", "error", err)
		return nil, resp, nil
	}

	analysis.PIIDisclaimer = "Log data was scrubbed of personally identifiable information before AI analysis. No emails, IPs, tokens, or credentials were sent to the AI model."

	return &analysis, resp, nil
}

// parseJSONResponse tries to extract JSON from a response that may contain markdown fences.
func parseJSONResponse(content string, dest interface{}) error {
	// Try direct parse first
	content = strings.TrimSpace(content)

	// Strip markdown code fences if present
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	return jsonUnmarshal([]byte(content), dest)
}

// jsonUnmarshal is a thin wrapper for testing seams.
var jsonUnmarshal = jsonUnmarshalImpl

func jsonUnmarshalImpl(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
