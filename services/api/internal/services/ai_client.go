package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// AIResponse holds the result of an AI completion.
type AIResponse struct {
	Content   string `json:"content"`
	TokensIn  int    `json:"tokens_in"`
	TokensOut int    `json:"tokens_out"`
	Model     string `json:"model"`
}

// AIClient wraps an OpenAI-compatible API (e.g. LiteLLM).
type AIClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
	enabled    bool
}

// NewAIClient creates a new AI client. If enabled is false, all calls return nil gracefully.
func NewAIClient(baseURL, apiKey, model string, enabled bool) *AIClient {
	return &AIClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
		enabled:    enabled,
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   chatUsage    `json:"usage"`
	Model   string       `json:"model"`
}

// Complete sends a system+user prompt to the AI and returns the response.
// On any error or when disabled, returns nil, nil (graceful degradation — NEVER blocks).
func (c *AIClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (*AIResponse, error) {
	if !c.enabled {
		return nil, nil
	}

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		slog.Warn("ai_client: marshal error", "error", err)
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		slog.Warn("ai_client: build request error", "error", err)
		return nil, nil
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("ai_client: request error", "error", err)
		return nil, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("ai_client: read response error", "error", err)
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		slog.Warn("ai_client: non-200 status", "status", resp.StatusCode, "body", string(respBody))
		return nil, nil
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		slog.Warn("ai_client: parse response error", "error", err)
		return nil, nil
	}

	if len(chatResp.Choices) == 0 {
		slog.Warn("ai_client: no choices in response")
		return nil, nil
	}

	return &AIResponse{
		Content:   chatResp.Choices[0].Message.Content,
		TokensIn:  chatResp.Usage.PromptTokens,
		TokensOut: chatResp.Usage.CompletionTokens,
		Model:     chatResp.Model,
	}, nil
}

// IsEnabled returns whether AI features are active.
func (c *AIClient) IsEnabled() bool {
	if c == nil {
		return false
	}
	return c.enabled
}

// ModelName returns the configured model name.
func (c *AIClient) ModelName() string {
	if c == nil {
		return ""
	}
	return c.model
}

// CompleteJSON sends a prompt and attempts to parse the response as JSON into dest.
func (c *AIClient) CompleteJSON(ctx context.Context, systemPrompt, userPrompt string, dest interface{}) (*AIResponse, error) {
	resp, err := c.Complete(ctx, systemPrompt, userPrompt)
	if err != nil || resp == nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(resp.Content), dest); err != nil {
		return resp, fmt.Errorf("ai_client: parse JSON response: %w", err)
	}
	return resp, nil
}
