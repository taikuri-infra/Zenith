package temporal

import (
	"fmt"

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
