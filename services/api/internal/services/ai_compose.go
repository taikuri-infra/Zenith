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

// ComposeSystemPrompt is defined in ai_prompts.go as ComposeSystemPrompt

type aiComposeResponse []string

// ValidateCompose performs AI-powered compose file analysis.
// Returns suggestion strings. On any error, returns empty list (never blocks).
func (v *AIComposeValidator) ValidateCompose(ctx context.Context, composeContent string) []string {
	if v.aiClient == nil || !v.aiClient.IsEnabled() {
		return nil
	}

	resp, err := v.aiClient.Complete(ctx, ComposeSystemPrompt, composeContent)
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

// FormatCompose uses AI to fix and reformat broken docker-compose YAML.
// Returns the corrected YAML string. On error, returns empty string.
func (v *AIComposeValidator) FormatCompose(ctx context.Context, composeContent string) string {
	if v.aiClient == nil || !v.aiClient.IsEnabled() {
		return ""
	}

	resp, err := v.aiClient.Complete(ctx, ComposeFormatPrompt, composeContent)
	if err != nil || resp == nil {
		return ""
	}

	return resp.Content
}

// ValidateComposeWithUsage performs AI-powered compose file analysis and returns usage info.
func (v *AIComposeValidator) ValidateComposeWithUsage(ctx context.Context, composeContent string) ([]string, *AIResponse) {
	if v.aiClient == nil || !v.aiClient.IsEnabled() {
		return nil, nil
	}

	resp, err := v.aiClient.Complete(ctx, ComposeSystemPrompt, composeContent)
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
