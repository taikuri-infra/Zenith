package deploy

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// Pipeline manages async build+deploy operations.
type Pipeline struct {
	builder  *Builder
	deployer *Deployer
	appRepo  ports.AppRepository
	logHub   *LogHub
	eventHub *EventHub
	mu       sync.Mutex
	running  map[string]context.CancelFunc // deploymentID -> cancel
}

// NewPipeline creates a new Pipeline.
func NewPipeline(builder *Builder, deployer *Deployer, appRepo ports.AppRepository, logHub *LogHub, eventHub *EventHub) *Pipeline {
	return &Pipeline{
		builder:  builder,
		deployer: deployer,
		appRepo:  appRepo,
		logHub:   logHub,
		eventHub: eventHub,
		running:  make(map[string]context.CancelFunc),
	}
}

// TriggerBuild starts an async build+deploy for an app deployment.
// It runs in a goroutine and updates deployment status as it progresses.
func (p *Pipeline) TriggerBuild(app *entities.App, deployment *entities.Deployment) {
	ctx, cancel := context.WithCancel(context.Background())

	p.mu.Lock()
	p.running[deployment.ID] = cancel
	p.mu.Unlock()

	go func() {
		defer func() {
			p.mu.Lock()
			delete(p.running, deployment.ID)
			p.mu.Unlock()
			cancel()
		}()

		log.Printf("[pipeline] Starting build for app=%s deploy=%s", app.Name, deployment.ID[:min(8, len(deployment.ID))])
		p.emitLog(deployment.ID, "info", fmt.Sprintf("Starting build for %s...", app.Name))
		p.emitEvent(EventDeploymentStarted, app, deployment, "", "Build started")

		// Run build
		result := p.builder.BuildApp(ctx, app, deployment)

		if result.Error != nil {
			log.Printf("[pipeline] Build failed for app=%s: %v", app.Name, result.Error)
			p.emitLog(deployment.ID, "error", fmt.Sprintf("Build failed: %v", result.Error))
			p.emitEvent(EventDeployFailed, app, deployment, "", fmt.Sprintf("Build failed: %v", result.Error))
			p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID,
				entities.DeployStatusFailed, result.BuildLog, result.Error.Error())

			status := entities.AppStatusFailed
			p.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{
				Status: &status,
			})
			return
		}

		// Mark previous active deployment as superseded
		if active, err := p.appRepo.GetActiveDeployment(ctx, app.ID); err == nil {
			p.appRepo.UpdateDeploymentStatus(ctx, active.ID, entities.DeployStatusSuperseded, "", "")
		}

		// Mark this deployment as deploying
		p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID,
			entities.DeployStatusDeploying, result.BuildLog, "")

		// Update app status to deploying
		status := entities.AppStatusDeploying
		p.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{
			Status: &status,
		})

		log.Printf("[pipeline] Build complete for app=%s image=%s", app.Name, result.ImageTag)
		p.emitLog(deployment.ID, "build", fmt.Sprintf("Build complete: %s", result.ImageTag))
		p.emitLog(deployment.ID, "deploy", "Deploying to Kubernetes...")
		p.emitEvent(EventBuildComplete, app, deployment, result.ImageTag, "Build complete")
		p.emitEvent(EventDeployStarted, app, deployment, result.ImageTag, "Deploying to Kubernetes")

		// Deploy to Kubernetes
		if p.deployer != nil {
			if err := p.deployer.DeployApp(ctx, app, result.ImageTag); err != nil {
				log.Printf("[pipeline] Deploy failed for app=%s: %v", app.Name, err)
				p.emitLog(deployment.ID, "error", fmt.Sprintf("Deploy failed: %v", err))
				p.emitEvent(EventDeployFailed, app, deployment, result.ImageTag, fmt.Sprintf("Deploy failed: %v", err))
				p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID,
					entities.DeployStatusFailed, result.BuildLog, err.Error())

				failedStatus := entities.AppStatusFailed
				p.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{
					Status: &failedStatus,
				})
				return
			}
		} else {
			// No deployer — mark as running directly (dev mode)
			runningStatus := entities.AppStatusRunning
			p.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{
				Status: &runningStatus,
			})
		}

		// Mark deployment as active
		p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID,
			entities.DeployStatusActive, result.BuildLog, "")
		p.emitLog(deployment.ID, "deploy", fmt.Sprintf("✓ Deployed successfully — %s is live", app.Name))
		p.emitEvent(EventDeployComplete, app, deployment, result.ImageTag, fmt.Sprintf("%s is live", app.Name))
	}()
}

// CancelBuild cancels an in-progress build.
func (p *Pipeline) CancelBuild(deploymentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cancel, ok := p.running[deploymentID]
	if !ok {
		return fmt.Errorf("no running build for deployment %s", deploymentID)
	}

	cancel()
	delete(p.running, deploymentID)
	return nil
}

// IsRunning checks if a build is currently running for a deployment.
func (p *Pipeline) IsRunning(deploymentID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.running[deploymentID]
	return ok
}

// RunningCount returns the number of currently running builds.
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
}

// TriggerImageDeploy deploys a pre-built image without a build step.
// Used when a customer selects a specific release version from the panel.
func (p *Pipeline) TriggerImageDeploy(app *entities.App, deployment *entities.Deployment, image string) {
	ctx, cancel := context.WithCancel(context.Background())

	p.mu.Lock()
	p.running[deployment.ID] = cancel
	p.mu.Unlock()

	go func() {
		defer func() {
			p.mu.Lock()
			delete(p.running, deployment.ID)
			p.mu.Unlock()
			cancel()
		}()

		log.Printf("[pipeline] Deploying image=%s app=%s deploy=%s", image, app.Name, deployment.ID[:min(8, len(deployment.ID))])
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
	}()
}
