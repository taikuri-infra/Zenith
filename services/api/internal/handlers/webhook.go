package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/services/deploy"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// WebhookHandler handles GitHub webhook events.
type WebhookHandler struct {
	appRepo       ports.AppRepository
	pipeline      *deploy.Pipeline
	webhookSecret string
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(appRepo ports.AppRepository, pipeline *deploy.Pipeline, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{appRepo: appRepo, pipeline: pipeline, webhookSecret: webhookSecret}
}

// GitHubPushEvent represents the relevant fields of a GitHub push event payload.
type GitHubPushEvent struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"head_commit"`
}

// HandlePush handles POST /api/v1/webhooks/github
func (h *WebhookHandler) HandlePush(c *fiber.Ctx) error {
	// Fail-closed: reject all requests if no webhook secret is configured
	if h.webhookSecret == "" {
		log.Println("[webhook] SECURITY: webhook secret not configured — rejecting request")
		return NewUnauthorized("webhook secret not configured")
	}

	signature := c.Get("X-Hub-Signature-256")
	if signature == "" {
		return NewUnauthorized("missing webhook signature")
	}
	if !h.verifySignature(c.Body(), signature) {
		return NewUnauthorized("invalid webhook signature")
	}

	// Only process push events
	event := c.Get("X-GitHub-Event")
	if event != "push" {
		// Acknowledge but ignore non-push events (e.g., ping)
		return c.JSON(fiber.Map{"message": "event ignored", "event": event})
	}

	var payload GitHubPushEvent
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return NewBadRequest("invalid webhook payload")
	}

	// Extract branch from ref (refs/heads/main -> main)
	branch := extractBranch(payload.Ref)
	if branch == "" {
		return c.JSON(fiber.Map{"message": "not a branch push, ignoring"})
	}

	repoURL := payload.Repository.CloneURL
	gitSHA := payload.After

	log.Printf("[webhook] Push received: repo=%s branch=%s sha=%s",
		payload.Repository.FullName, branch, gitSHA[:min(8, len(gitSHA))])

	// Find apps that match this repo URL and branch
	apps, err := h.findAppsByRepo(c, repoURL, branch)
	if err != nil || len(apps) == 0 {
		log.Printf("[webhook] No matching apps found for repo=%s branch=%s", repoURL, branch)
		return c.JSON(fiber.Map{
			"message": "no matching apps",
			"repo":    payload.Repository.FullName,
			"branch":  branch,
		})
	}

	// Create deployments for each matching app
	var triggered []string
	for _, app := range apps {
		deployment, err := h.appRepo.CreateDeployment(c.Context(), app.ID, gitSHA)
		if err != nil {
			log.Printf("[webhook] Failed to create deployment for app %s: %v", app.Name, err)
			continue
		}

		// Update app status to building
		status := entities.AppStatusBuilding
		h.appRepo.UpdateApp(c.Context(), app.ID, &dto.UpdateAppInput{
			Status: &status,
		})

		log.Printf("[webhook] Deployment created: app=%s deploy_id=%s sha=%s",
			app.Name, deployment.ID, gitSHA[:min(8, len(gitSHA))])
		triggered = append(triggered, app.Name)

		// Trigger async build pipeline
		if h.pipeline != nil {
			h.pipeline.TriggerBuild(&app, deployment)
		}
	}

	return c.JSON(fiber.Map{
		"message":     "deployments triggered",
		"triggered":   triggered,
		"commit":      gitSHA,
		"branch":      branch,
	})
}

// findAppsByRepo looks up apps matching the given repo URL and branch.
// MVP: scans all apps. Production: use a repo URL index in the database.
func (h *WebhookHandler) findAppsByRepo(c *fiber.Ctx, repoURL, branch string) ([]entities.App, error) {
	// Normalize repo URL (remove .git suffix for comparison)
	normalizedURL := strings.TrimSuffix(repoURL, ".git")

	// Scan all apps (MVP approach — works fine for small scale)
	// Pass empty userID to get all apps across users
	allApps, err := h.appRepo.ListAppsByUser(c.Context(), "")
	if err != nil {
		return nil, err
	}

	var matched []entities.App
	for _, app := range allApps {
		appURL := strings.TrimSuffix(app.RepoURL, ".git")
		if appURL == normalizedURL && app.Branch == branch {
			matched = append(matched, app)
		}
	}
	return matched, nil
}

// verifySignature validates the GitHub webhook HMAC-SHA256 signature.
func (h *WebhookHandler) verifySignature(body []byte, signature string) bool {
	// GitHub sends "sha256=<hex>"
	if len(signature) < 8 || signature[:7] != "sha256=" {
		return false
	}

	expected := signature[7:]
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	computed := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(computed))
}

// HandleGitLabPush handles POST /api/v1/webhooks/gitlab
func (h *WebhookHandler) HandleGitLabPush(c *fiber.Ctx) error {
	// Fail-closed: reject all requests if no webhook secret is configured
	if h.webhookSecret == "" {
		log.Println("[webhook:gitlab] SECURITY: webhook secret not configured — rejecting request")
		return NewUnauthorized("webhook secret not configured")
	}

	token := c.Get("X-Gitlab-Token")
	if !ConstantTimeCompare(token, h.webhookSecret) {
		return NewUnauthorized("invalid webhook token")
	}

	// Only process Push Hook events
	event := c.Get("X-Gitlab-Event")
	if event != "Push Hook" {
		return c.JSON(fiber.Map{"message": "event ignored", "event": event})
	}

	var payload struct {
		Ref     string `json:"ref"`
		After   string `json:"checkout_sha"`
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
			WebURL            string `json:"web_url"`
			GitHTTPURL        string `json:"git_http_url"`
		} `json:"project"`
	}
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return NewBadRequest("invalid webhook payload")
	}

	branch := extractBranch(payload.Ref)
	if branch == "" {
		return c.JSON(fiber.Map{"message": "not a branch push, ignoring"})
	}

	repoURL := payload.Project.GitHTTPURL
	gitSHA := payload.After

	log.Printf("[webhook:gitlab] Push received: repo=%s branch=%s sha=%s",
		payload.Project.PathWithNamespace, branch, gitSHA[:min(8, len(gitSHA))])

	apps, err := h.findAppsByRepo(c, repoURL, branch)
	if err != nil || len(apps) == 0 {
		return c.JSON(fiber.Map{
			"message": "no matching apps",
			"repo":    payload.Project.PathWithNamespace,
			"branch":  branch,
		})
	}

	var triggered []string
	for _, app := range apps {
		deployment, err := h.appRepo.CreateDeployment(c.Context(), app.ID, gitSHA)
		if err != nil {
			log.Printf("[webhook:gitlab] Failed to create deployment for app %s: %v", app.Name, err)
			continue
		}

		status := entities.AppStatusBuilding
		h.appRepo.UpdateApp(c.Context(), app.ID, &dto.UpdateAppInput{Status: &status})

		log.Printf("[webhook:gitlab] Deployment created: app=%s deploy_id=%s", app.Name, deployment.ID)
		triggered = append(triggered, app.Name)

		if h.pipeline != nil {
			h.pipeline.TriggerBuild(&app, deployment)
		}
	}

	return c.JSON(fiber.Map{
		"message":   "deployments triggered",
		"triggered": triggered,
		"commit":    gitSHA,
		"branch":    branch,
	})
}

// HandleBitbucketPush handles POST /api/v1/webhooks/bitbucket
func (h *WebhookHandler) HandleBitbucketPush(c *fiber.Ctx) error {
	// Fail-closed: reject all requests if no webhook secret is configured
	if h.webhookSecret == "" {
		log.Println("[webhook:bitbucket] SECURITY: webhook secret not configured — rejecting request")
		return NewUnauthorized("webhook secret not configured")
	}

	signature := c.Get("X-Hub-Signature")
	if signature == "" {
		return NewUnauthorized("missing webhook signature")
	}
	if !h.verifySignature(c.Body(), signature) {
		return NewUnauthorized("invalid webhook signature")
	}

	eventKey := c.Get("X-Event-Key")
	if eventKey != "repo:push" {
		return c.JSON(fiber.Map{"message": "event ignored", "event": eventKey})
	}

	var payload struct {
		Push struct {
			Changes []struct {
				New struct {
					Type   string `json:"type"`
					Name   string `json:"name"`
					Target struct {
						Hash string `json:"hash"`
					} `json:"target"`
				} `json:"new"`
			} `json:"changes"`
		} `json:"push"`
		Repository struct {
			FullName string `json:"full_name"`
			Links    struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return NewBadRequest("invalid webhook payload")
	}

	if len(payload.Push.Changes) == 0 {
		return c.JSON(fiber.Map{"message": "no changes in push"})
	}

	change := payload.Push.Changes[0]
	if change.New.Type != "branch" {
		return c.JSON(fiber.Map{"message": "not a branch push, ignoring"})
	}

	branch := change.New.Name
	gitSHA := change.New.Target.Hash
	repoURL := payload.Repository.Links.HTML.Href

	log.Printf("[webhook:bitbucket] Push received: repo=%s branch=%s sha=%s",
		payload.Repository.FullName, branch, gitSHA[:min(8, len(gitSHA))])

	apps, err := h.findAppsByRepo(c, repoURL, branch)
	if err != nil || len(apps) == 0 {
		return c.JSON(fiber.Map{
			"message": "no matching apps",
			"repo":    payload.Repository.FullName,
			"branch":  branch,
		})
	}

	var triggered []string
	for _, app := range apps {
		deployment, err := h.appRepo.CreateDeployment(c.Context(), app.ID, gitSHA)
		if err != nil {
			log.Printf("[webhook:bitbucket] Failed to create deployment for app %s: %v", app.Name, err)
			continue
		}

		status := entities.AppStatusBuilding
		h.appRepo.UpdateApp(c.Context(), app.ID, &dto.UpdateAppInput{Status: &status})

		log.Printf("[webhook:bitbucket] Deployment created: app=%s deploy_id=%s", app.Name, deployment.ID)
		triggered = append(triggered, app.Name)

		if h.pipeline != nil {
			h.pipeline.TriggerBuild(&app, deployment)
		}
	}

	return c.JSON(fiber.Map{
		"message":   "deployments triggered",
		"triggered": triggered,
		"commit":    gitSHA,
		"branch":    branch,
	})
}

// extractBranch extracts the branch name from a Git ref.
// e.g., "refs/heads/main" -> "main", "refs/tags/v1.0" -> "" (not a branch)
func extractBranch(ref string) string {
	const prefix = "refs/heads/"
	if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
		return ref[len(prefix):]
	}
	return ""
}

// ConstantTimeCompare provides timing-safe string comparison for webhook tokens.
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
