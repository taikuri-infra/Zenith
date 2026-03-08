package temporal

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// NewWorker creates and returns a Temporal worker that processes provisioning workflows.
// Call worker.Start() to begin processing, and worker.Stop() for graceful shutdown.
func NewWorker(c client.Client, activities *Activities, planActivities *PlanActivities) worker.Worker {
	w := worker.New(c, TaskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(ProvisionCustomerWorkflow)
	w.RegisterWorkflow(DeprovisionCustomerWorkflow)
	w.RegisterWorkflow(PlanOrchestratorWorkflow)

	// Register all activities on the shared structs
	w.RegisterActivity(activities)
	if planActivities != nil {
		w.RegisterActivity(planActivities)
	}

	return w
}
