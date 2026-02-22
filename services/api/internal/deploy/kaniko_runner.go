package deploy

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/k8s"
)

const (
	kanikoNamespace    = "zenith-builds"
	jobPollInterval    = 5 * time.Second
	jobStartTimeout    = 2 * time.Minute
	jobBuildTimeout    = 30 * time.Minute
)

// KanikoRunner submits a Kaniko K8s Job, streams its logs, and waits for
// completion. It is nil-safe — if the runner is nil (no k8s client in dev
// mode), calling Build is a no-op that returns immediately with nil error.
type KanikoRunner struct {
	k8sClient k8s.Client
	logHub    *LogHub
}

// NewKanikoRunner creates a KanikoRunner. Both arguments are optional (nil-safe).
func NewKanikoRunner(k8sClient k8s.Client, logHub *LogHub) *KanikoRunner {
	if k8sClient == nil {
		return nil
	}
	return &KanikoRunner{
		k8sClient: k8sClient,
		logHub:    logHub,
	}
}

// Build runs the Kaniko build Job end-to-end:
//  1. Submit the K8s Job
//  2. Wait for pod to start
//  3. Stream pod logs → LogHub
//  4. Wait for Job success/failure
//  5. Clean up Job on success
func (r *KanikoRunner) Build(ctx context.Context, spec *KanikoJobSpec, deploymentID string) error {
	if r == nil {
		// Dev mode: no k8s client — skip actual build
		return nil
	}

	r.emitLog(deploymentID, "build", fmt.Sprintf("Submitting Kaniko build job: %s", spec.Name))
	log.Printf("[kaniko] Submitting job %s in namespace %s", spec.Name, kanikoNamespace)

	// 1. Create the K8s Job
	job := &k8s.JobObject{
		Name:      spec.Name,
		Namespace: kanikoNamespace,
		Labels: map[string]string{
			"zenith.dev/component":  "build",
			"zenith.dev/app-id":     spec.AppID,
			"zenith.dev/deployment": spec.DeploymentID,
		},
		Spec: spec.ToK8sJobManifest(),
	}

	if err := r.k8sClient.CreateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to submit build job: %w", err)
	}

	r.emitLog(deploymentID, "build", "Build job queued — waiting for execution...")

	// 2. Wait until Job has a result (Succeeded or Failed)
	buildCtx, cancel := context.WithTimeout(ctx, jobBuildTimeout)
	defer cancel()

	if err := r.waitForJob(buildCtx, deploymentID, spec.Name); err != nil {
		// Clean up on failure (best-effort)
		_ = r.k8sClient.DeleteJob(ctx, kanikoNamespace, spec.Name)
		return err
	}

	// 3. Stream pod logs (non-blocking after job completes)
	r.streamLogs(ctx, deploymentID, spec.Name)

	// 4. Delete the Job after success
	if err := r.k8sClient.DeleteJob(ctx, kanikoNamespace, spec.Name); err != nil {
		log.Printf("[kaniko] Warning: failed to delete job %s: %v", spec.Name, err)
	}

	r.emitLog(deploymentID, "build", "Image built and pushed successfully")
	return nil
}

// waitForJob polls until the Job is Succeeded or Failed.
func (r *KanikoRunner) waitForJob(ctx context.Context, deploymentID, jobName string) error {
	ticker := time.NewTicker(jobPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("build timed out after %s", jobBuildTimeout)

		case <-ticker.C:
			job, err := r.k8sClient.GetJob(ctx, kanikoNamespace, jobName)
			if err != nil {
				// Job might not exist yet — keep waiting
				log.Printf("[kaniko] Waiting for job %s: %v", jobName, err)
				continue
			}

			if job.Succeeded > 0 {
				return nil
			}
			if job.Failed > 0 {
				return fmt.Errorf("kaniko build job failed")
			}

			r.emitLog(deploymentID, "build", "Building image...")
		}
	}
}

// streamLogs fetches pod logs and publishes each line to the LogHub.
// It runs synchronously so log lines arrive before the "built" message.
func (r *KanikoRunner) streamLogs(ctx context.Context, deploymentID, jobName string) {
	logCh := make(chan string, 64)
	podSelector := "zenith.dev/deployment=" + deploymentID

	go func() {
		if err := r.k8sClient.GetPodLogs(ctx, kanikoNamespace, podSelector, logCh); err != nil {
			log.Printf("[kaniko] GetPodLogs error: %v", err)
		}
	}()

	timeout := time.After(30 * time.Second)
	for {
		select {
		case line, ok := <-logCh:
			if !ok {
				return
			}
			r.emitLog(deploymentID, "build", line)

		case <-timeout:
			return

		case <-ctx.Done():
			return
		}
	}
}

// emitLog publishes a log entry to the LogHub if available.
func (r *KanikoRunner) emitLog(deploymentID, level, message string) {
	if r != nil && r.logHub != nil {
		r.logHub.Publish(deploymentID, LogEntry{Level: level, Message: message})
	}
}
