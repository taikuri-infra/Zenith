package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
)

// runningBuild tracks a single in-progress deploy.
type runningBuild struct {
	Cancel context.CancelFunc
	UserID string
}

// Pipeline manages async image deploy operations.
type Pipeline struct {
	deployer      Backend
	appRepo       ports.AppRepository
	logHub        *LogHub
	eventHub      *EventHub
	eventBus      ports.EventBus // NATS JetStream (optional)
	webhookSvc    *services.WebhookDeliveryService
	hookRepo      ports.DeployHookRepository
	mu            sync.Mutex
	running       map[string]*runningBuild // deploymentID -> build info
	maxConcurrent int
	maxPerUser    int
}

// NewPipeline creates a new Pipeline.
func NewPipeline(deployer Backend, appRepo ports.AppRepository, logHub *LogHub, eventHub *EventHub, maxConcurrent int) *Pipeline {
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	return &Pipeline{
		deployer:      deployer,
		appRepo:       appRepo,
		logHub:        logHub,
		eventHub:      eventHub,
		running:       make(map[string]*runningBuild),
		maxConcurrent: maxConcurrent,
		// SaaS-safe default: cap concurrent deploys per user so one tenant can't
		// starve others. Standalone self-host raises this via SetMaxPerUser
		// (single user deploying their whole multi-service stack at once).
		maxPerUser: 2,
	}
}

// SetMaxPerUser overrides the per-user concurrent-deploy cap. Used by standalone
// self-host to allow a whole multi-service app to deploy at once. Not used in
// SaaS mode, which keeps the fairness cap.
func (p *Pipeline) SetMaxPerUser(n int) {
	if n > 0 {
		p.maxPerUser = n
	}
}

// TriggerImageDeploy deploys a pre-built image.
// Returns an error if the deploy queue is full or the user has too many concurrent deploys.
func (p *Pipeline) TriggerImageDeploy(app *entities.App, deployment *entities.Deployment, image string) error {
	p.mu.Lock()
	if len(p.running) >= p.maxConcurrent {
		p.mu.Unlock()
		return fmt.Errorf("deploy queue full (%d/%d), try again later", len(p.running), p.maxConcurrent)
	}
	userDeploys := 0
	for _, b := range p.running {
		if b.UserID == app.UserID {
			userDeploys++
		}
	}
	if userDeploys >= p.maxPerUser {
		p.mu.Unlock()
		return fmt.Errorf("max concurrent deploys reached for this user (%d/%d)", userDeploys, p.maxPerUser)
	}

	// 20-minute hard timeout per deploy — prevents hung K8s API calls from leaking slots forever.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	p.running[deployment.ID] = &runningBuild{Cancel: cancel, UserID: app.UserID}
	p.mu.Unlock()

	go func() {
		defer func() {
			p.mu.Lock()
			delete(p.running, deployment.ID)
			p.mu.Unlock()
			cancel()
		}()

		slog.Info("deploying image", "image", image, "app", app.Name, "deploy_id", deployment.ID[:min(8, len(deployment.ID))])
		p.emitLog(deployment.ID, "info", fmt.Sprintf("Deploying %s → %s...", app.Name, image))
		p.emitEvent(EventDeploymentStarted, app, deployment, image, fmt.Sprintf("Deploying %s", image))

		// Mark previous active as superseded
		if active, err := p.appRepo.GetActiveDeployment(ctx, app.ID); err == nil {
			p.appRepo.UpdateDeploymentStatus(ctx, active.ID, entities.DeployStatusSuperseded, "", "")
		}

		p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID, entities.DeployStatusDeploying, "", "")

		deployingStatus := entities.AppStatusDeploying
		p.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{Status: &deployingStatus})

		if p.deployer != nil {
			if err := p.deployer.DeployApp(ctx, app, image); err != nil {
				p.emitLog(deployment.ID, "error", fmt.Sprintf("Deploy failed: %v", err))
				p.emitEvent(EventDeployFailed, app, deployment, image, fmt.Sprintf("Deploy failed: %v", err))
				p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID, entities.DeployStatusFailed, "", err.Error())
				failedStatus := entities.AppStatusFailed
				p.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{Status: &failedStatus})
				if p.webhookSvc != nil {
					p.webhookSvc.DispatchEvent(ctx, app.UserID, entities.WebhookEventDeployFailed, map[string]interface{}{
						"app":           app.Name,
						"app_id":        app.ID,
						"deployment_id": deployment.ID,
						"image":         image,
						"status":        "failed",
						"error":         err.Error(),
					})
				}
				return
			}
		} else {
			// No deployer — mark as running directly (dev mode)
			runningStatus := entities.AppStatusRunning
			p.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{Status: &runningStatus})
		}

		p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID, entities.DeployStatusActive, "", "")
		p.emitLog(deployment.ID, "deploy", fmt.Sprintf("✓ %s is live — %s", app.Name, image))
		p.emitEvent(EventDeployComplete, app, deployment, image, fmt.Sprintf("%s is live", app.Name))

		// Fire webhook delivery for deploy.success
		if p.webhookSvc != nil {
			p.webhookSvc.DispatchEvent(ctx, app.UserID, entities.WebhookEventDeploySuccess, map[string]interface{}{
				"app":           app.Name,
				"app_id":        app.ID,
				"deployment_id": deployment.ID,
				"image":         image,
				"status":        "success",
			})
		}

		// Execute post-deploy hooks
		if p.hookRepo != nil {
			hooks, hErr := p.hookRepo.ListHooksByApp(ctx, app.ID)
			if hErr == nil {
				for _, hook := range hooks {
					if !hook.Active {
						continue
					}
					p.emitLog(deployment.ID, "hook", fmt.Sprintf("Running hook: %s", hook.Name))
					if hook.Type == entities.DeployHookHTTP && hook.URL != "" {
						p.executeHTTPHook(hook, app, deployment, image)
					}
					if hook.Type == entities.DeployHookCommand && hook.Command != "" {
						p.executeCommandHook(ctx, hook, app, deployment)
					}
				}
			}
		}
	}()

	return nil
}

// CancelBuild cancels an in-progress deploy.
func (p *Pipeline) CancelBuild(deploymentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	rb, ok := p.running[deploymentID]
	if !ok {
		return fmt.Errorf("no running deploy for deployment %s", deploymentID)
	}

	rb.Cancel()
	delete(p.running, deploymentID)
	return nil
}

// IsRunning checks if a deploy is currently running for a deployment.
func (p *Pipeline) IsRunning(deploymentID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.running[deploymentID]
	return ok
}

// RunningCount returns the number of currently running deploys.
func (p *Pipeline) RunningCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.running)
}

// LogHub returns the pipeline's log hub for WebSocket access.
func (p *Pipeline) LogHub() *LogHub {
	return p.logHub
}

// emitLog publishes a log entry if a LogHub is configured.
func (p *Pipeline) emitLog(deploymentID, level, message string) {
	if p.logHub != nil {
		p.logHub.Publish(deploymentID, LogEntry{Level: level, Message: message})
	}
}

// SetEventBus configures the NATS event bus for durable event publishing.
func (p *Pipeline) SetEventBus(bus ports.EventBus) {
	p.eventBus = bus
}

// SetWebhookService configures the webhook delivery service.
func (p *Pipeline) SetWebhookService(svc *services.WebhookDeliveryService) {
	p.webhookSvc = svc
}

// SetHookRepo configures the deploy hook repository for post-deploy hooks.
func (p *Pipeline) SetHookRepo(repo ports.DeployHookRepository) {
	p.hookRepo = repo
}

// executeHTTPHook sends an HTTP POST to the hook URL with deploy context.
func (p *Pipeline) executeHTTPHook(hook entities.DeployHook, app *entities.App, deployment *entities.Deployment, image string) {
	payload, _ := json.Marshal(map[string]string{
		"app":           app.Name,
		"app_id":        app.ID,
		"deployment_id": deployment.ID,
		"image":         image,
		"status":        "success",
		"hook":          hook.Name,
	})
	req, err := http.NewRequest("POST", hook.URL, bytes.NewReader(payload))
	if err != nil {
		slog.Warn("hook: failed to create request", "hook", hook.Name, "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Zenith-Hook", hook.Name)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("hook: request failed", "hook", hook.Name, "error", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		slog.Warn("hook: returned error", "hook", hook.Name, "status", resp.StatusCode)
	}
}

// executeCommandHook runs a shell command inside the app's running container via kubectl exec.
// Hooks are best-effort: failures are logged but never abort the deployment.
func (p *Pipeline) executeCommandHook(ctx context.Context, hook entities.DeployHook, app *entities.App, deployment *entities.Deployment) {
	// Step 1: find the name of a running pod for this app.
	// Pods are labelled with app=<subdomain> by the k8s resource generator.
	getPod := exec.CommandContext(ctx,
		"kubectl", "get", "pods",
		"-l", "app="+app.Subdomain,
		"-n", "zenith-apps",
		"--field-selector=status.phase=Running",
		"-o", "jsonpath={.items[0].metadata.name}",
	)
	podOut, err := getPod.Output()
	if err != nil {
		slog.Warn("command hook: kubectl get pods failed",
			"hook", hook.Name, "app", app.Subdomain, "error", err)
		p.emitLog(deployment.ID, "hook", fmt.Sprintf("hook %s: could not find running pod for %s", hook.Name, app.Subdomain))
		return
	}

	podName := strings.TrimSpace(string(podOut))
	if podName == "" {
		slog.Warn("command hook: no running pod found", "hook", hook.Name, "app", app.Subdomain)
		p.emitLog(deployment.ID, "hook", fmt.Sprintf("hook %s: no running pod for %s — skipped", hook.Name, app.Subdomain))
		return
	}

	// Step 2: exec the command inside the container.
	execCmd := exec.CommandContext(ctx,
		"kubectl", "exec", podName,
		"-n", "zenith-apps",
		"--", "sh", "-c", hook.Command,
	)
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	if err := execCmd.Run(); err != nil {
		slog.Warn("command hook: exec failed",
			"hook", hook.Name, "pod", podName,
			"error", err,
			"stderr", stderr.String(),
		)
		p.emitLog(deployment.ID, "hook", fmt.Sprintf("hook %s failed on pod %s: %v — %s", hook.Name, podName, err, strings.TrimSpace(stderr.String())))
		return
	}

	output := strings.TrimSpace(stdout.String())
	slog.Info("command hook executed", "hook", hook.Name, "pod", podName, "output", output)
	if output != "" {
		p.emitLog(deployment.ID, "hook", fmt.Sprintf("hook %s: %s", hook.Name, output))
	}
}

// emitEvent publishes a deployment event if an EventHub is configured.
func (p *Pipeline) emitEvent(eventType EventType, app *entities.App, deployment *entities.Deployment, image, message string) {
	if p.eventHub != nil {
		p.eventHub.Publish(DeployEvent{
			Type:         eventType,
			AppID:        app.ID,
			AppName:      app.Name,
			DeploymentID: deployment.ID,
			Status:       string(deployment.Status),
			Image:        image,
			Message:      message,
		})
	}

	// Also publish to NATS event bus for durable processing
	if p.eventBus != nil {
		var subject entities.EventSubject
		switch eventType {
		case EventDeploymentStarted:
			subject = entities.EventDeployStarted
		case EventDeployComplete:
			subject = entities.EventDeployCompleted
		case EventDeployFailed:
			subject = entities.EventDeployFailed
		default:
			return
		}

		evt := &entities.PlatformEvent{
			Subject:   subject,
			UserID:    app.UserID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"app_id":        app.ID,
				"app_name":      app.Name,
				"deployment_id": deployment.ID,
				"image":         image,
				"message":       message,
			},
		}
		if err := p.eventBus.Publish(context.Background(), evt); err != nil {
			slog.Error("failed to publish event to bus", "error", err)
		}
	}
}
