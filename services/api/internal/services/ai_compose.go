package services

import (
	"context"
	"encoding/json"
	"log/slog"
)

// AIComposeValidator uses AI to perform Layer 3 validation on docker-compose files.
type AIComposeValidator struct {
	aiClient *AIClient
}

// NewAIComposeValidator creates a new AI compose validator.
func NewAIComposeValidator(aiClient *AIClient) *AIComposeValidator {
	return &AIComposeValidator{aiClient: aiClient}
}

const composeSystemPrompt = `You are a DevOps expert reviewing docker-compose.yml files for production deployment on Kubernetes.
Analyze the compose file and return a JSON array of suggestion strings.
Focus on:
- Security issues (running as root, exposed debug ports, hardcoded secrets)
- Performance (missing resource limits, inefficient configurations)
- Reliability (missing health checks, restart policies, logging)
- Kubernetes compatibility (unsupported features like network_mode: host)
Return ONLY a JSON array of strings, no markdown, no explanation.
Example: ["Add health checks to your API service","Use environment variables instead of hardcoded database passwords","Consider adding resource limits"]
If everything looks good, return: ["Your compose file looks production-ready!"]`

type aiComposeResponse []string

// ValidateCompose performs AI-powered compose file analysis.
// Returns suggestion strings. On any error, returns empty list (never blocks).
func (v *AIComposeValidator) ValidateCompose(ctx context.Context, composeContent string) []string {
	if v.aiClient == nil || !v.aiClient.IsEnabled() {
		return nil
	}

	resp, err := v.aiClient.Complete(ctx, composeSystemPrompt, composeContent)
	if err != nil || resp == nil {
		return nil
	}

	var suggestions aiComposeResponse
	if err := json.Unmarshal([]byte(resp.Content), &suggestions); err != nil {
		slog.Warn("ai_compose: failed to parse AI response", "error", err, "content", resp.Content)
		return nil
	}

	return suggestions
}

// ValidateComposeWithUsage performs AI-powered compose file analysis and returns usage info.
func (v *AIComposeValidator) ValidateComposeWithUsage(ctx context.Context, composeContent string) ([]string, *AIResponse) {
	if v.aiClient == nil || !v.aiClient.IsEnabled() {
		return nil, nil
	}

	resp, err := v.aiClient.Complete(ctx, composeSystemPrompt, composeContent)
	if err != nil || resp == nil {
		return nil, nil
	}

	var suggestions aiComposeResponse
	if err := json.Unmarshal([]byte(resp.Content), &suggestions); err != nil {
		slog.Warn("ai_compose: failed to parse AI response", "error", err)
		return nil, resp
	}

	return suggestions, resp
}
