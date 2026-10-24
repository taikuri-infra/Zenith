// services/api/internal/services/deploy/k8s_db.go
package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/services"
)

var (
	_ services.DBBackend       = (*K8sDBBackend)(nil)
	_ services.EngineSupporter = (*K8sDBBackend)(nil)
)

const cnpgAPIVersion = "postgresql.cnpg.io/v1"

// K8sDBBackend runs managed databases through Kubernetes operators. PostgreSQL
// uses CloudNativePG: one Cluster per database, credentials from a Secret the
// backend owns, in-cluster access through the operator's <name>-rw Service.
type K8sDBBackend struct {
	client    k8sclient.Client
	namespace string
	pollEvery time.Duration
}

func NewK8sDBBackend(c k8sclient.Client, namespace string) *K8sDBBackend {
	return &K8sDBBackend{client: c, namespace: namespace, pollEvery: 3 * time.Second}
}

// SupportsEngine: only operator-backed engines are offered. MySQL and Redis
// join when their operators are wired in.
func (b *K8sDBBackend) SupportsEngine(e entities.DatabaseEngine) bool {
	return e == entities.DatabaseEnginePostgres
}

func k8sDBName(db *entities.UserDatabase) string { return "db-" + dbShortID(db) }

func (b *K8sDBBackend) Provision(ctx context.Context, db *entities.UserDatabase, pw string) (string, int, error) {
	if !b.SupportsEngine(db.Engine) {
		return "", 0, fmt.Errorf("engine %s is not available on Kubernetes yet", db.Engine)
	}
	name := k8sDBName(db)
	secret := name + "-auth"
	labels := map[string]string{"zenith.dev/db-id": db.ID, "zenith.dev/engine": string(db.Engine)}

	// Idempotent: replace credentials and the Cluster if a previous attempt left them.
	_ = b.client.DeleteSecret(ctx, b.namespace, secret)
	if err := b.client.CreateSecret(ctx, b.namespace, secret, map[string][]byte{
		"username": []byte(db.DBUser),
		"password": []byte(pw),
	}, labels); err != nil {
		return "", 0, fmt.Errorf("create secret %s: %w", secret, err)
	}

	spec, _ := json.Marshal(map[string]any{
		"instances": 1,
		"storage":   map[string]any{"size": dbStorage(db)},
		"bootstrap": map[string]any{"initdb": map[string]any{
			"database": db.DBName,
			"owner":    db.DBUser,
			"secret":   map[string]any{"name": secret},
		}},
	})
	_ = b.client.DeleteCRDWithVersion(ctx, cnpgAPIVersion, "Cluster", b.namespace, name)
	if err := b.client.CreateCRD(ctx, &k8sclient.CRDObject{
		APIVersion: cnpgAPIVersion,
		Kind:       "Cluster",
		Metadata:   k8sclient.ObjectMeta{Name: name, Namespace: b.namespace, Labels: labels},
		Spec:       spec,
	}); err != nil {
		return "", 0, fmt.Errorf("create cluster %s: %w", name, err)
	}
	return fmt.Sprintf("%s-rw.%s.svc.cluster.local", name, b.namespace), 5432, nil
}

func dbStorage(db *entities.UserDatabase) string {
	if db.MaxSizeMB >= 1024 {
		return fmt.Sprintf("%dGi", (db.MaxSizeMB+1023)/1024)
	}
	return "1Gi"
}

// WaitReady polls the Cluster until an instance is ready or ctx ends.
func (b *K8sDBBackend) WaitReady(ctx context.Context, db *entities.UserDatabase) error {
	name := k8sDBName(db)
	t := time.NewTicker(b.pollEvery)
	defer t.Stop()
	for {
		obj, err := b.client.GetCRDWithVersion(ctx, cnpgAPIVersion, "Cluster", b.namespace, name)
		if err == nil && clusterReady(obj.Status) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("database %s not ready: %w", name, ctx.Err())
		case <-t.C:
		}
	}
}

func clusterReady(status []byte) bool {
	if len(status) == 0 {
		return false
	}
	var st struct {
		ReadyInstances int    `json:"readyInstances"`
		Phase          string `json:"phase"`
	}
	if json.Unmarshal(status, &st) != nil {
		return false
	}
	return st.ReadyInstances >= 1 || strings.Contains(strings.ToLower(st.Phase), "healthy")
}

// Delete removes the Cluster (the operator drops its volumes) and the credentials.
func (b *K8sDBBackend) Delete(ctx context.Context, db *entities.UserDatabase, _ bool) error {
	name := k8sDBName(db)
	err := b.client.DeleteCRDWithVersion(ctx, cnpgAPIVersion, "Cluster", b.namespace, name)
	_ = b.client.DeleteSecret(ctx, b.namespace, name+"-auth")
	return ignoreNotFound(err)
}

func ignoreNotFound(err error) error {
	if err != nil && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound")) {
		return nil
	}
	return err
}
