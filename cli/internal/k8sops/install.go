package k8sops

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// installScript is infra/scripts/install-k3s.sh; a test keeps the two in sync.
//
//go:embed install-k3s.sh
var installScript []byte

// DefaultNamespace is where the zenith-selfhost chart installs the platform.
const DefaultNamespace = "zenith-system"

// k3sKubeconfig is where k3s writes its kubeconfig.
const k3sKubeconfig = "/etc/rancher/k3s/k3s.yaml"

// Install is a Zenith installation on a Kubernetes cluster.
type Install struct {
	Kubeconfig string // empty = kubectl's default resolution
	Namespace  string
}

// Detect finds the installation reachable with kubeconfig (or $KUBECONFIG, or the
// k3s default) by looking for the platform namespace.
func Detect(ctx context.Context, r Runner, kubeconfig string) (Install, bool) {
	inst := Install{Kubeconfig: ResolveKubeconfig(kubeconfig), Namespace: DefaultNamespace}
	if _, err := inst.kubectl(ctx, r, nil, "get", "deploy", "zenith-api", "-o", "name"); err != nil {
		return inst, false
	}
	return inst, true
}

// ResolveKubeconfig picks the flag, then $KUBECONFIG, then the k3s default if it exists.
func ResolveKubeconfig(flag string) string {
	if flag != "" {
		return flag
	}
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	if _, err := os.Stat(k3sKubeconfig); err == nil {
		return k3sKubeconfig
	}
	return ""
}

func (i Install) args(args []string) []string {
	var pre []string
	if i.Kubeconfig != "" {
		pre = append(pre, "--kubeconfig", i.Kubeconfig)
	}
	pre = append(pre, "-n", i.Namespace)
	return append(pre, args...)
}

func (i Install) kubectl(ctx context.Context, r Runner, stdin readerOrNil, args ...string) ([]byte, error) {
	return r.Run(ctx, "", stdin, "kubectl", i.args(args)...)
}

func (i Install) kubectlTo(ctx context.Context, r Runner, w writer, args ...string) error {
	return r.RunTo(ctx, "", w, "kubectl", i.args(args)...)
}

// jsonpath returns one kubectl jsonpath value, trimmed.
func (i Install) jsonpath(ctx context.Context, r Runner, path string, args ...string) (string, error) {
	out, err := i.kubectl(ctx, r, nil, append(args, "-o", "jsonpath="+path)...)
	return strings.TrimSpace(string(out)), err
}

// primaryPod returns the primary pod of a CloudNativePG cluster.
func (i Install) primaryPod(ctx context.Context, r Runner, cluster string) (string, error) {
	name, err := i.jsonpath(ctx, r, "{.items[0].metadata.name}", "get", "pods",
		"-l", "cnpg.io/cluster="+cluster+",cnpg.io/instanceRole=primary")
	if err != nil || name == "" {
		return "", fmt.Errorf("no primary pod for cluster %s (is it running?): %v", cluster, err)
	}
	return name, nil
}

// Version is the image tag the API runs.
func (i Install) Version(ctx context.Context, r Runner) string {
	img, err := i.jsonpath(ctx, r, "{.spec.template.spec.containers[0].image}", "get", "deploy", "zenith-api")
	if err != nil {
		return ""
	}
	if idx := strings.LastIndex(img, ":"); idx >= 0 && !strings.Contains(img[idx:], "/") {
		return img[idx+1:]
	}
	return ""
}

// AppsNamespace is where user databases and apps run.
const AppsNamespace = "zenith-apps"

// DBRef is a managed database. Engine: postgresql|mysql|redis.
type DBRef struct{ ID, Engine, Name, DBName, DBUser, Namespace string }

// Resource is the name of the database's operator resource and secret prefix.
func (d DBRef) Resource() string { return "db-" + d.ID[:min(8, len(d.ID))] }

var (
	dbNSPattern = regexp.MustCompile(`^(zenith-apps|zp-[a-z0-9-]{1,58})$`)
)

// Databases lists ready managed databases from the control-plane Postgres.
func (i Install) Databases(ctx context.Context, r Runner) ([]DBRef, error) {
	pod, err := i.primaryPod(ctx, r, "zenith-postgres")
	if err != nil {
		return nil, err
	}
	// JSON keeps user-controlled names from shifting columns or splitting rows.
	out, err := i.kubectl(ctx, r, nil, "exec", pod, "-c", "postgres", "--", "psql", "-U", "postgres", "-d", "zenith",
		"-At", "-c",
		"SELECT COALESCE(json_agg(json_build_object('ID', d.id, 'Engine', d.engine, 'Name', d.name, "+
			"'DBName', d.db_name, 'DBUser', d.db_user, 'Namespace', COALESCE(p.namespace, '"+AppsNamespace+"')) ORDER BY d.created_at), '[]'::json) "+
			"FROM user_databases d LEFT JOIN projects p ON p.id = d.project_id WHERE d.status = 'ready'")
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}
	var dbs []DBRef
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &dbs); err != nil {
		return nil, fmt.Errorf("list databases: unexpected output: %w", err)
	}
	for _, d := range dbs {
		if err := d.Validate(); err != nil {
			return nil, err
		}
	}
	return dbs, nil
}

// Validate rejects database records whose identifiers could reach kubectl or SQL in a surprising way.
func (d DBRef) Validate() error {
	if !dbIDPattern.MatchString(d.ID) || !dbNSPattern.MatchString(d.Namespace) ||
		(d.Engine != "postgresql" && d.Engine != "mysql" && d.Engine != "redis") {
		return fmt.Errorf("invalid database record %q (engine %q, namespace %q)", d.ID, d.Engine, d.Namespace)
	}
	return nil
}

// Uninstall removes the Helm release and the platform and apps namespaces.
func Uninstall(ctx context.Context, r Runner, inst Install) error {
	args := []string{"uninstall", "zenith", "-n", inst.Namespace}
	if inst.Kubeconfig != "" {
		args = append(args, "--kubeconfig", inst.Kubeconfig)
	}
	if _, err := r.Run(ctx, "", nil, "helm", args...); err != nil {
		return fmt.Errorf("helm uninstall: %w", err)
	}
	nss := []string{inst.Namespace, AppsNamespace}
	// Project namespaces (zp-*) are labelled when Zenith creates them.
	if out, err := r.Run(ctx, "", nil, "kubectl", kubeArgs(inst, "get", "namespace", "-l", "zenith.dev/managed=true", "-o", "name")...); err == nil {
		for _, n := range strings.Fields(string(out)) {
			nss = append(nss, strings.TrimPrefix(n, "namespace/"))
		}
	}
	for _, ns := range nss {
		a := []string{"delete", "namespace", ns, "--ignore-not-found", "--wait=false"}
		if inst.Kubeconfig != "" {
			a = append([]string{"--kubeconfig", inst.Kubeconfig}, a...)
		}
		if _, err := r.Run(ctx, "", nil, "kubectl", a...); err != nil {
			return fmt.Errorf("delete namespace %s: %w", ns, err)
		}
	}
	return nil
}

func kubeArgs(inst Install, args ...string) []string {
	if inst.Kubeconfig != "" {
		return append([]string{"--kubeconfig", inst.Kubeconfig}, args...)
	}
	return args
}
