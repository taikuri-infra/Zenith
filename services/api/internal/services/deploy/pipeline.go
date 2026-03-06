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

// runningBuild tracks a single in-progress deploy.
type runningBuild struct {
	Cancel context.CancelFunc
	UserID string
}

// Pipeline manages async image deploy operations.
type Pipeline struct {
	deployer      *Deployer
	appRepo       ports.AppRepository
	logHub        *LogHub
	eventHub      *EventHub
	mu            sync.Mutex
	running       map[string]*runningBuild // deploymentID -> build info
	maxConcurrent int
	maxPerUser    int
}

// NewPipeline creates a new Pipeline.
func NewPipeline(deployer *Deployer, appRepo ports.AppRepository, logHub *LogHub, eventHub *EventHub, maxConcurrent int) *Pipeline {
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
		maxPerUser:    2,
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

	ctx, cancel := context.WithCancel(context.Background())
	p.running[deployment.ID] = &runningBuild{Cancel: cancel, UserID: app.UserID}
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
