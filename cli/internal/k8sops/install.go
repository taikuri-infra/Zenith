package k8sops

import (
	"context"
	_ "embed"
	"fmt"
	"os"
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

// Databases lists ready managed databases from the control-plane Postgres.
func (i Install) Databases(ctx context.Context, r Runner) ([]DBRef, error) {
	pod, err := i.primaryPod(ctx, r, "zenith-postgres")
	if err != nil {
		return nil, err
	}
	out, err := i.kubectl(ctx, r, nil, "exec", pod, "-c", "postgres", "--", "psql", "-U", "postgres", "-d", "zenith",
		"-At", "-F", "|", "-c",
		"SELECT d.id, d.engine, d.name, d.db_name, d.db_user, COALESCE(p.namespace, '"+AppsNamespace+"') "+
			"FROM user_databases d LEFT JOIN projects p ON p.id = d.project_id WHERE d.status = 'ready' ORDER BY d.created_at")
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}
	var dbs []DBRef
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p := strings.SplitN(line, "|", 6)
		if len(p) != 6 {
			return nil, fmt.Errorf("unexpected database row %q", line)
		}
		dbs = append(dbs, DBRef{ID: p[0], Engine: p[1], Name: p[2], DBName: p[3], DBUser: p[4], Namespace: p[5]})
	}
	return dbs, nil
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
