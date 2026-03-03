package temporal

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// NewWorker creates and returns a Temporal worker that processes provisioning workflows.
// Call worker.Start() to begin processing, and worker.Stop() for graceful shutdown.
func NewWorker(c client.Client, activities *Activities) worker.Worker {
	w := worker.New(c, TaskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(ProvisionCustomerWorkflow)
	w.RegisterWorkflow(DeprovisionCustomerWorkflow)

	// Register all activities on the shared struct
	w.RegisterActivity(activities)

	return w
}
