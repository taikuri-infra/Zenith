package deploy

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// BuildResult contains the output of a build pipeline execution.
type BuildResult struct {
	ImageTag  string
	BuildLog  string
	Framework entities.Framework
	Error     error
}

// Builder orchestrates the build pipeline for an app deployment.
type Builder struct {
	appRepo       ports.AppRepository
	workDir       string
	registry      string
	kanikoRunner  *KanikoRunner
}

// NewBuilder creates a new Builder.
//
//   - k8sClient: optional — if nil, Kaniko build is skipped (dev mode).
//   - logHub: optional — if non-nil, build log lines are streamed in real time.
func NewBuilder(appRepo ports.AppRepository, workDir, registry string, k8sClient k8sclient.Client, logHub *LogHub) *Builder {
	if workDir == "" {
		workDir = "/tmp/zenith-builds"
	}
	if registry == "" {
		registry = "registry.freezenith.com"
	}
	return &Builder{
		appRepo:      appRepo,
		workDir:      workDir,
		registry:     registry,
		kanikoRunner: NewKanikoRunner(k8sClient, logHub),
	}
}

// BuildApp runs the full build pipeline for an app deployment:
//  1. Clone repo
//  2. Detect framework
//  3. Generate Dockerfile (if needed)
//  4. Submit Kaniko K8s Job (skipped in dev mode when no k8sClient)
//  5. Update deployment status
func (b *Builder) BuildApp(ctx context.Context, app *entities.App, deployment *entities.Deployment) *BuildResult {
	result := &BuildResult{}
	buildLog := ""

	appendLog := func(msg string) {
		line := fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), msg)
		buildLog += line
		log.Printf("[build:%s] %s", app.Name, msg)
	}

	// Step 1: Create build directory
	buildDir := filepath.Join(b.workDir, app.ID, deployment.ID)
	cloneDir := filepath.Join(buildDir, "source")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		result.Error = fmt.Errorf("failed to create build dir: %w", err)
		result.BuildLog = buildLog
		return result
	}
	defer os.RemoveAll(buildDir)

	// Step 2: Clone repo
	appendLog(fmt.Sprintf("Cloning %s (branch: %s)...", app.RepoURL, app.Branch))
	b.appRepo.UpdateDeploymentStatus(ctx, deployment.ID, entities.DeployStatusBuilding, buildLog, "")

	if err := CloneRepo(ctx, app.RepoURL, app.Branch, cloneDir); err != nil {
		appendLog(fmt.Sprintf("Clone failed: %v", err))
		result.Error = err
		result.BuildLog = buildLog
		return result
	}

	// Get commit SHA
	sha, err := GetLatestCommitSHA(ctx, cloneDir)
	if err != nil {
		appendLog(fmt.Sprintf("Warning: could not get commit SHA: %v", err))
	} else {
		appendLog(fmt.Sprintf("Commit: %s", sha[:min(8, len(sha))]))
	}

	// Step 3: Detect framework
	framework := DetectFramework(cloneDir)
	appendLog(fmt.Sprintf("Detected framework: %s", framework))
	result.Framework = framework

	// Update app with detected framework
	fw := framework
	b.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{
		Framework: &fw,
	})

	// Step 4: Generate Dockerfile if needed
	var generatedDockerfile string
	if framework != entities.FrameworkDockerfile {
		appendLog("Generating Dockerfile...")
		content, err := GenerateDockerfile(framework, app.Name, app.Port)
		if err != nil {
			appendLog(fmt.Sprintf("Dockerfile generation failed: %v", err))
			result.Error = err
			result.BuildLog = buildLog
			return result
		}
		generatedDockerfile = content
		appendLog("Dockerfile generated")
	} else {
		appendLog("Using existing Dockerfile from repo")
	}

	// Step 5: Build container image via Kaniko
	imageTag := fmt.Sprintf("%s/%s:%s", b.registry, app.Subdomain, deployment.ID[:min(8, len(deployment.ID))])
	appendLog(fmt.Sprintf("Building image: %s", imageTag))
	result.ImageTag = imageTag

	b.appRepo.UpdateDeploymentStatus(ctx, deployment.ID, entities.DeployStatusBuilding, buildLog, "")

	if b.kanikoRunner != nil {
		// Production: submit Kaniko K8s Job with repo URL/branch.
		// The Kaniko pod's init container clones the repo — no local path needed.
		jobSpec := NewKanikoJobSpec(app, deployment.ID, imageTag)
		jobSpec.GeneratedDockerfile = generatedDockerfile
		if err := b.kanikoRunner.Build(ctx, jobSpec, deployment.ID); err != nil {
			appendLog(fmt.Sprintf("Image build failed: %v", err))
			result.Error = err
			result.BuildLog = buildLog
			return result
		}
	} else {
		// Dev mode: no k8s client — log skipped build
		appendLog("(dev mode) Skipping actual image build — no k8s client configured")
	}

	appendLog("Build complete")
	result.BuildLog = buildLog
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
