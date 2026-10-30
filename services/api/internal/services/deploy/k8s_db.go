// services/api/internal/services/deploy/k8s_db.go
package deploy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

const (
	cnpgAPIVersion    = "postgresql.cnpg.io/v1"
	mariadbAPIVersion = "k8s.mariadb.com/v1alpha1"
	redisAPIVersion   = "redis.redis.opstreelabs.in/v1beta2"
	redisImage        = "quay.io/opstree/redis:v7.0.15"
)

// K8sDBBackend runs managed databases through Kubernetes operators:
//   - PostgreSQL: CloudNativePG, one Cluster per database (<name>-rw Service)
//   - MySQL: mariadb-operator, one MariaDB per database (MySQL wire protocol)
//   - Redis: OpsTree redis-operator, one Redis per database
//
// Credentials live in a Secret the backend owns; access is in-cluster.
type K8sDBBackend struct {
	client    k8sclient.Client
	namespace string // fallback when no resolver is set
	resolve   NamespaceResolver
	pollEvery time.Duration
}

// SetNamespaceResolver puts each database in its project's namespace.
func (b *K8sDBBackend) SetNamespaceResolver(r NamespaceResolver) { b.resolve = r }

func (b *K8sDBBackend) nsFor(ctx context.Context, db *entities.UserDatabase) string {
	if b.resolve != nil {
		return b.resolve(ctx, db.ProjectID)
	}
	return b.namespace
}

func NewK8sDBBackend(c k8sclient.Client, namespace string) *K8sDBBackend {
	return &K8sDBBackend{client: c, namespace: namespace, pollEvery: 3 * time.Second}
}

// SupportsEngine: the engines that have an operator behind them.
func (b *K8sDBBackend) SupportsEngine(e entities.DatabaseEngine) bool {
	switch e {
	case entities.DatabaseEnginePostgres, entities.DatabaseEngineMySQL, entities.DatabaseEngineRedis:
		return true
	}
	return false
}

// k8sDBName is the operator resource name: db-<first 8 chars of the database id>.
func k8sDBName(db *entities.UserDatabase) string {
	id := db.ID
	if len(id) > 8 {
		id = id[:8]
	}
	return "db-" + id
}

// dbKind is the operator custom resource (apiVersion, kind) for an engine.
func dbKind(e entities.DatabaseEngine) (string, string) {
	switch e {
	case entities.DatabaseEngineMySQL:
		return mariadbAPIVersion, "MariaDB"
	case entities.DatabaseEngineRedis:
		return redisAPIVersion, "Redis"
	}
	return cnpgAPIVersion, "Cluster"
}

func (b *K8sDBBackend) Provision(ctx context.Context, db *entities.UserDatabase, pw string) (string, int, error) {
	if !b.SupportsEngine(db.Engine) {
		return "", 0, fmt.Errorf("engine %s is not available on Kubernetes yet", db.Engine)
	}
	name := k8sDBName(db)
	ns := b.nsFor(ctx, db)
	if err := EnsureNamespace(ctx, b.client, ns); err != nil {
		return "", 0, fmt.Errorf("namespace %s: %w", ns, err)
	}
	secret := name + "-auth"
	labels := map[string]string{"zenith.dev/db-id": db.ID, "zenith.dev/engine": string(db.Engine)}

	data := map[string][]byte{"username": []byte(db.DBUser), "password": []byte(pw)}
	if db.Engine == entities.DatabaseEngineMySQL {
		// The operator wants a root password too; it is never shown to users.
		data["root-password"] = []byte(randomHex(16))
	}
	// Idempotent: replace credentials and the resource if a previous attempt left them.
	_ = b.client.DeleteSecret(ctx, ns, secret)
	if err := b.client.CreateSecret(ctx, ns, secret, data, labels); err != nil {
		return "", 0, fmt.Errorf("create secret %s: %w", secret, err)
	}

	apiVersion, kind := dbKind(db.Engine)
	spec, host, port := b.spec(db, name, secret, ns)
	specJSON, _ := json.Marshal(spec)
	_ = b.client.DeleteCRDWithVersion(ctx, apiVersion, kind, ns, name)
	if err := b.client.CreateCRD(ctx, &k8sclient.CRDObject{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata:   k8sclient.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
		Spec:       specJSON,
	}); err != nil {
		return "", 0, fmt.Errorf("create %s %s: %w", kind, name, err)
	}
	return host, port, nil
}

// spec builds the operator resource spec and the in-cluster address.
func (b *K8sDBBackend) spec(db *entities.UserDatabase, name, secret, ns string) (map[string]any, string, int) {
	size := dbStorage(db)
	switch db.Engine {
	case entities.DatabaseEngineMySQL:
		return map[string]any{
			"rootPasswordSecretKeyRef": map[string]any{"name": secret, "key": "root-password"},
			"database":                 db.DBName,
			"username":                 db.DBUser,
			"passwordSecretKeyRef":     map[string]any{"name": secret, "key": "password"},
			"storage":                  map[string]any{"size": size},
			"replicas":                 1,
		}, fmt.Sprintf("%s.%s.svc.cluster.local", name, ns), 3306
	case entities.DatabaseEngineRedis:
		return map[string]any{
			"kubernetesConfig": map[string]any{
				"image":           redisImage,
				"imagePullPolicy": "IfNotPresent",
				"redisSecret":     map[string]any{"name": secret, "key": "password"},
			},
			"storage": map[string]any{"volumeClaimTemplate": map[string]any{"spec": map[string]any{
				"accessModes": []string{"ReadWriteOnce"},
				"resources":   map[string]any{"requests": map[string]any{"storage": size}},
			}}},
		}, fmt.Sprintf("%s.%s.svc.cluster.local", name, ns), 6379
	}
	return map[string]any{
		"instances": 1,
		"storage":   map[string]any{"size": size},
		"bootstrap": map[string]any{"initdb": map[string]any{
			"database": db.DBName,
			"owner":    db.DBUser,
			"secret":   map[string]any{"name": secret},
		}},
	}, fmt.Sprintf("%s-rw.%s.svc.cluster.local", name, ns), 5432
}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func dbStorage(db *entities.UserDatabase) string {
	if db.MaxSizeMB >= 1024 {
		return fmt.Sprintf("%dGi", (db.MaxSizeMB+1023)/1024)
	}
	return "1Gi"
}

// WaitReady polls until the engine is serving or ctx ends.
func (b *K8sDBBackend) WaitReady(ctx context.Context, db *entities.UserDatabase) error {
	name := k8sDBName(db)
	t := time.NewTicker(b.pollEvery)
	defer t.Stop()
	for {
		if b.ready(ctx, db, name) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("database %s not ready: %w", name, ctx.Err())
		case <-t.C:
		}
	}
}

func (b *K8sDBBackend) ready(ctx context.Context, db *entities.UserDatabase, name string) bool {
	ns := b.nsFor(ctx, db)
	if db.Engine == entities.DatabaseEngineRedis {
		// The Redis resource has no status; its StatefulSet carries readiness.
		obj, err := b.client.GetCRDWithVersion(ctx, "apps/v1", "StatefulSet", ns, name)
		return err == nil && statefulSetReady(obj.Status)
	}
	apiVersion, kind := dbKind(db.Engine)
	obj, err := b.client.GetCRDWithVersion(ctx, apiVersion, kind, ns, name)
	if err != nil {
		return false
	}
	if db.Engine == entities.DatabaseEngineMySQL {
		return mariadbReady(obj.Status)
	}
	return clusterReady(obj.Status)
}

func statefulSetReady(status []byte) bool {
	var st struct {
		ReadyReplicas int `json:"readyReplicas"`
	}
	return len(status) > 0 && json.Unmarshal(status, &st) == nil && st.ReadyReplicas >= 1
}

func mariadbReady(status []byte) bool {
	var st struct {
		Conditions []struct {
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"conditions"`
	}
	if len(status) == 0 || json.Unmarshal(status, &st) != nil {
		return false
	}
	for _, c := range st.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return true
		}
	}
	return false
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

// Delete removes the operator resource (the operator drops its volumes) and the credentials.
func (b *K8sDBBackend) Delete(ctx context.Context, db *entities.UserDatabase, _ bool) error {
	name := k8sDBName(db)
	ns := b.nsFor(ctx, db)
	apiVersion, kind := dbKind(db.Engine)
	err := b.client.DeleteCRDWithVersion(ctx, apiVersion, kind, ns, name)
	_ = b.client.DeleteSecret(ctx, ns, name+"-auth")
	if db.Engine == entities.DatabaseEngineMySQL {
		// mariadb-operator keeps its data volume after the resource is gone.
		_ = b.client.DeleteCRDWithVersion(ctx, "v1", "PersistentVolumeClaim", ns, "storage-"+name+"-0")
	}
	return ignoreNotFound(err)
}

func ignoreNotFound(err error) error {
	if err != nil && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound")) {
		return nil
	}
	return err
}
