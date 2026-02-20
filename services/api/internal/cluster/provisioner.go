package cluster

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/capi"
	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/dotechhq/zenith/services/api/internal/store"
)

// Provisioner orchestrates CAPI cluster lifecycle for customers.
type Provisioner struct {
	capi      *capi.Client
	customers store.CustomerRepository
	admin     store.AdminRepository
	stopCh    chan struct{}
}

// NewProvisioner creates a new cluster Provisioner.
func NewProvisioner(capiClient *capi.Client, customers store.CustomerRepository, admin store.AdminRepository) *Provisioner {
	return &Provisioner{
		capi:      capiClient,
		customers: customers,
		admin:     admin,
		stopCh:    make(chan struct{}),
	}
}

// ProvisionCluster creates a CAPI cluster for a customer and updates DB status.
func (p *Provisioner) ProvisionCluster(ctx context.Context, customer *models.Customer) error {
	// Update status to provisioning
	if err := p.customers.UpdateClusterStatus(ctx, customer.ID, models.ClusterStatusProvisioning); err != nil {
		return err
	}

	// Create CAPI cluster CRD
	input := models.CreateClusterInput{
		Name:       customer.CAPIClusterName,
		Region:     customer.ClusterRegion,
		Type:       "dedicated",
		Tenant:     customer.CAPIClusterName,
		Nodes:      customer.ClusterNodes,
		K8sVersion: customer.ClusterK8sVersion,
	}

	if _, err := p.capi.CreateCluster(ctx, input); err != nil {
		_ = p.customers.UpdateClusterStatus(ctx, customer.ID, models.ClusterStatusError)
		return err
	}

	// Audit log
	_ = p.admin.AddAuditEntry(ctx, models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  "system",
		Action: "Provisioning cluster " + customer.CAPIClusterName + " for " + customer.Name,
	})

	return nil
}

// TeardownCluster deletes the CAPI cluster for a customer and updates DB status.
func (p *Provisioner) TeardownCluster(ctx context.Context, customer *models.Customer) error {
	if customer.CAPIClusterName == "" {
		return nil
	}

	_ = p.customers.UpdateClusterStatus(ctx, customer.ID, models.ClusterStatusDeleting)

	if err := p.capi.DeleteCluster(ctx, customer.CAPIClusterName); err != nil {
		log.Printf("Warning: failed to delete CAPI cluster %s: %v", customer.CAPIClusterName, err)
	}

	_ = p.admin.AddAuditEntry(ctx, models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  "system",
		Action: "Tearing down cluster " + customer.CAPIClusterName + " for " + customer.Name,
	})

	return nil
}

// ScaleCluster scales the CAPI cluster and updates DB.
func (p *Provisioner) ScaleCluster(ctx context.Context, customer *models.Customer, nodes int) error {
	if err := p.capi.ScaleCluster(ctx, customer.CAPIClusterName, nodes); err != nil {
		return err
	}

	if err := p.customers.UpdateClusterInfo(ctx, customer.ID, nodes, customer.ClusterK8sVersion); err != nil {
		return err
	}

	_ = p.admin.AddAuditEntry(ctx, models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  "system",
		Action: "Scaled cluster " + customer.CAPIClusterName + " to " + itoa(nodes) + " nodes",
	})

	return nil
}

// UpgradeCluster upgrades the CAPI cluster K8s version and updates DB.
func (p *Provisioner) UpgradeCluster(ctx context.Context, customer *models.Customer, version string) error {
	if err := p.capi.UpgradeCluster(ctx, customer.CAPIClusterName, version); err != nil {
		return err
	}

	if err := p.customers.UpdateClusterInfo(ctx, customer.ID, customer.ClusterNodes, version); err != nil {
		return err
	}

	_ = p.admin.AddAuditEntry(ctx, models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  "system",
		Action: "Upgrading cluster " + customer.CAPIClusterName + " to " + version,
	})

	return nil
}

// GetCluster retrieves the CAPI cluster resource for a customer.
func (p *Provisioner) GetCluster(ctx context.Context, clusterName string) (*models.Cluster, error) {
	return p.capi.GetCluster(ctx, clusterName)
}

// StartSync starts a background goroutine that polls CAPI status and updates
// customer cluster_status in the DB.
func (p *Provisioner) StartSync(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-p.stopCh:
				return
			case <-ticker.C:
				p.syncOnce()
			}
		}
	}()
}

// Stop signals the sync goroutine to stop.
func (p *Provisioner) Stop() {
	close(p.stopCh)
}

func (p *Provisioner) syncOnce() {
	ctx := context.Background()

	customers, err := p.customers.ListProvisioningCustomers(ctx)
	if err != nil {
		log.Printf("cluster sync: failed to list provisioning customers: %v", err)
		return
	}

	for _, cust := range customers {
		if cust.CAPIClusterName == "" {
			continue
		}

		cluster, err := p.capi.GetCluster(ctx, cust.CAPIClusterName)
		if err != nil {
			continue
		}

		// Map CAPI cluster status to customer cluster status
		var newStatus string
		switch cluster.Status {
		case "healthy":
			newStatus = models.ClusterStatusRunning
		case "error":
			newStatus = models.ClusterStatusError
		default:
			continue
		}

		if newStatus != cust.ClusterStatus {
			if err := p.customers.UpdateClusterStatus(ctx, cust.ID, newStatus); err != nil {
				log.Printf("cluster sync: failed to update status for %s: %v", cust.CAPIClusterName, err)
			}
		}
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
