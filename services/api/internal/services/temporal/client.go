package temporal

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/ports"
	"go.temporal.io/sdk/client"
)

const (
	// TaskQueue is the Temporal task queue for customer provisioning workflows.
	TaskQueue = "zenith-provisioning"

	// WorkflowProvisionCustomer is the workflow type name.
	WorkflowProvisionCustomer = "ProvisionCustomerWorkflow"

	// WorkflowDeprovisionCustomer is the workflow type name.
	WorkflowDeprovisionCustomer = "DeprovisionCustomerWorkflow"
)

// NewClient creates a Temporal client connected to the given host/namespace.
func NewClient(host, namespace string) (client.Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  host,
		Namespace: namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("temporal client dial %s: %w", host, err)
	}
	return c, nil
}

// WorkflowClient implements ports.ProvisioningWorkflow by wrapping
// the Temporal SDK client.
type WorkflowClient struct {
	tc client.Client
}

// Compile-time check.
var _ ports.ProvisioningWorkflow = (*WorkflowClient)(nil)

// NewWorkflowClient creates a ProvisioningWorkflow adapter around a Temporal client.
func NewWorkflowClient(tc client.Client) *WorkflowClient {
	return &WorkflowClient{tc: tc}
}

// RawClient returns the underlying Temporal client (for SetTemporalClient in main.go).
func (w *WorkflowClient) RawClient() client.Client {
	return w.tc
}

func (w *WorkflowClient) StartProvision(ctx context.Context, input ports.ProvisionInput) error {
	opts := client.StartWorkflowOptions{
		ID:        "provision-" + input.CustomerID,
		TaskQueue: TaskQueue,
	}
	_, err := w.tc.ExecuteWorkflow(ctx, opts, WorkflowProvisionCustomer, ProvisionInput{
		CustomerID:   input.CustomerID,
		CustomerName: input.CustomerName,
		Domain:       input.Domain,
		PlanTier:     input.PlanTier,
		ContactEmail: input.ContactEmail,
	})
	return err
}

func (w *WorkflowClient) StartPlanChange(ctx context.Context, input PlanChangeInput) error {
	opts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("plan-change-%s-%d", input.UserID, time.Now().UnixMilli()),
		TaskQueue: TaskQueue,
	}
	_, err := w.tc.ExecuteWorkflow(ctx, opts, WorkflowPlanOrchestrator, input)
	return err
}

func (w *WorkflowClient) StartDeprovision(ctx context.Context, input ports.DeprovisionInput) error {
	opts := client.StartWorkflowOptions{
		ID:        "deprovision-" + input.CustomerID,
		TaskQueue: TaskQueue,
	}
	_, err := w.tc.ExecuteWorkflow(ctx, opts, WorkflowDeprovisionCustomer, DeprovisionInput{
		CustomerID:   input.CustomerID,
		CustomerName: input.CustomerName,
		Domain:       input.Domain,
		Namespace:    input.Namespace,
	})
	return err
}
