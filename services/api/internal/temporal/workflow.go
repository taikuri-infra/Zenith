package temporal

import (
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ProvisionCustomerWorkflow orchestrates all 10 steps of tenant provisioning.
// Each activity is idempotent; Temporal handles retries automatically.
func ProvisionCustomerWorkflow(ctx workflow.Context, input ProvisionInput) error {
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    1 * time.Minute,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Step 0: Mark customer as provisioning
	if err := workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusProvisioning, input.CustomerID).Get(ctx, nil); err != nil {
		return err
	}

	// Step 1: Create Keycloak realm + OIDC client
	var keycloakResult CreateKeycloakRealmResult
	if err := workflow.ExecuteActivity(ctx, (*Activities).CreateKeycloakRealm, input).Get(ctx, &keycloakResult); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusError, input.CustomerID).Get(ctx, nil)
		return fmt.Errorf("create keycloak realm: %w", err)
	}

	// Step 2: Create database
	var dbResult CreateDatabaseResult
	if err := workflow.ExecuteActivity(ctx, (*Activities).CreateDatabase, input).Get(ctx, &dbResult); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusError, input.CustomerID).Get(ctx, nil)
		return fmt.Errorf("create database: %w", err)
	}

	// Step 3: Create S3 bucket
	var s3Result CreateS3BucketResult
	if err := workflow.ExecuteActivity(ctx, (*Activities).CreateS3Bucket, input).Get(ctx, &s3Result); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusError, input.CustomerID).Get(ctx, nil)
		return fmt.Errorf("create s3 bucket: %w", err)
	}

	// Step 4: Create Kubernetes namespace
	var nsResult CreateNamespaceResult
	if err := workflow.ExecuteActivity(ctx, (*Activities).CreateNamespace, input).Get(ctx, &nsResult); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusError, input.CustomerID).Get(ctx, nil)
		return fmt.Errorf("create namespace: %w", err)
	}

	// Step 5: Create secrets in namespace
	secretsInput := CreateSecretsInput{
		ProvisionInput: input,
		Namespace:      nsResult.Namespace,
		DBName:         dbResult.DBName,
		DBUser:         dbResult.DBUser,
		DBPass:         dbResult.DBPass,
		ClientSecret:   keycloakResult.ClientSecret,
		RealmName:      keycloakResult.RealmName,
		BucketName:     s3Result.BucketName,
	}
	if err := workflow.ExecuteActivity(ctx, (*Activities).CreateSecrets, secretsInput).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusError, input.CustomerID).Get(ctx, nil)
		return fmt.Errorf("create secrets: %w", err)
	}

	// Step 6: Create resource quota + limit range
	if err := workflow.ExecuteActivity(ctx, (*Activities).CreateResourceQuota, input, nsResult.Namespace).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusError, input.CustomerID).Get(ctx, nil)
		return fmt.Errorf("create resource quota: %w", err)
	}

	// Step 7: Create APISIX routing
	if err := workflow.ExecuteActivity(ctx, (*Activities).CreateRouting, input, nsResult.Namespace).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusError, input.CustomerID).Get(ctx, nil)
		return fmt.Errorf("create routing: %w", err)
	}

	// Step 8: Create TLS certificate
	if err := workflow.ExecuteActivity(ctx, (*Activities).CreateTLS, input, nsResult.Namespace).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusError, input.CustomerID).Get(ctx, nil)
		return fmt.Errorf("create tls certificate: %w", err)
	}

	// Step 9: Create ArgoCD Application
	if err := workflow.ExecuteActivity(ctx, (*Activities).CreateArgoCD, input, nsResult.Namespace).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusError, input.CustomerID).Get(ctx, nil)
		return fmt.Errorf("create argocd app: %w", err)
	}

	// Step 10: Mark as ready + audit log
	if err := workflow.ExecuteActivity(ctx, (*Activities).NotifyReady, input, nsResult.Namespace).Get(ctx, nil); err != nil {
		return fmt.Errorf("notify ready: %w", err)
	}

	return nil
}

// DeprovisionCustomerWorkflow tears down all tenant resources.
// Best-effort: continues even if individual steps fail.
func DeprovisionCustomerWorkflow(ctx workflow.Context, input DeprovisionInput) error {
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    1 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Update status to deleting
	_ = workflow.ExecuteActivity(ctx, (*Activities).UpdateStatusProvisioning, input.CustomerID).Get(ctx, nil)

	// Delete ArgoCD app first (stops syncing)
	_ = workflow.ExecuteActivity(ctx, (*Activities).DeleteArgoCD, input.Domain).Get(ctx, nil)

	// Delete namespace (cleans up all k8s resources: secrets, quotas, routes, certs)
	_ = workflow.ExecuteActivity(ctx, (*Activities).DeleteNamespace, input.Domain).Get(ctx, nil)

	// Delete external resources
	_ = workflow.ExecuteActivity(ctx, (*Activities).DeleteS3Bucket, input.Domain).Get(ctx, nil)
	_ = workflow.ExecuteActivity(ctx, (*Activities).DeleteDatabase, input.Domain).Get(ctx, nil)
	_ = workflow.ExecuteActivity(ctx, (*Activities).DeleteKeycloakRealm, input.Domain).Get(ctx, nil)

	// Audit
	_ = workflow.ExecuteActivity(ctx, (*Activities).NotifyReady, ProvisionInput{
		CustomerID:   input.CustomerID,
		CustomerName: input.CustomerName,
		Domain:       input.Domain,
	}, input.Namespace).Get(ctx, nil)

	return nil
}

// auditEntry creates an entities.AuditEntry (unexported helper, used in activities).
func auditEntry(actor, action string) entities.AuditEntry {
	return entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: action,
	}
}
