package k8sops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// RestoreOptions tunes the readiness polling of Restore.
type RestoreOptions struct {
	Wait time.Duration // default 10m
	Poll time.Duration // default 3s
}

func waitFor(ctx context.Context, opt RestoreOptions, what string, probe func() bool) error {
	deadline := time.Now().Add(opt.Wait)
	for {
		if probe() {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s", what)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opt.Poll):
		}
	}
}

func readManifest(dir string) (*Manifest, error) {
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("archive has no manifest.json: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("manifest.json: %w", err)
	}
	if m.Format != 2 {
		return nil, fmt.Errorf("unsupported archive format %d (this zen restores format 2, written on Kubernetes)", m.Format)
	}
	for i := range m.Databases {
		if m.Databases[i].Namespace == "" { // archives written before projects had namespaces
			m.Databases[i].Namespace = AppsNamespace
		}
		if err := m.Databases[i].Validate(); err != nil {
			return nil, fmt.Errorf("manifest.json: %w", err)
		}
	}
	return &m, nil
}

// secretManifest rebuilds a minimal Secret from the JSON `kubectl get` wrote:
// resourceVersion, uid and friends would make apply fail on another cluster.
func secretManifest(raw []byte, namespace string) ([]byte, error) {
	var in struct {
		Data map[string]string `json:"data"`
		Type string            `json:"type"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("secrets.json: %w", err)
	}
	if len(in.Data) == 0 {
		return nil, errors.New("secrets.json holds no keys")
	}
	if in.Type == "" {
		in.Type = "Opaque"
	}
	return json.Marshal(map[string]any{
		"apiVersion": "v1", "kind": "Secret", "type": in.Type, "data": in.Data,
		"metadata": map[string]any{"name": secretsName, "namespace": namespace},
	})
}

func fileReader(path string, fn func(io.Reader) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}

// Restore rebuilds the install from an archive written by Backup. The API is
// stopped while the data is replaced; on start it recreates the managed
// databases and apps recorded in the platform database, and Restore then loads
// each database's dump into its fresh instance.
func Restore(ctx context.Context, r Runner, inst Install, archive string, opt RestoreOptions) (*Manifest, error) {
	if opt.Wait <= 0 {
		opt.Wait = 10 * time.Minute
	}
	if opt.Poll <= 0 {
		opt.Poll = 3 * time.Second
	}
	tmp, err := os.MkdirTemp("", "zenith-restore-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	if err := extractArchive(archive, tmp); err != nil {
		return nil, err
	}
	m, err := readManifest(tmp)
	if err != nil {
		return nil, err
	}
	if cur := inst.Version(ctx, r); cur != "" && m.Version != "" && CompareVersions(m.Version, cur) > 0 {
		return nil, fmt.Errorf("archive is from Zenith %s, newer than the installed %s: upgrade first", m.Version, cur)
	}

	if _, err := inst.kubectl(ctx, r, nil, "scale", "deploy/zenith-api", "--replicas=0"); err != nil {
		return nil, fmt.Errorf("stop the API: %w", err)
	}
	if err := waitFor(ctx, opt, "the API to stop", func() bool {
		n, err := inst.jsonpath(ctx, r, "{.status.replicas}", "get", "deploy", "zenith-api")
		return err == nil && (n == "" || n == "0")
	}); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(filepath.Join(tmp, "secrets.json"))
	if err != nil {
		return nil, fmt.Errorf("archive has no secrets.json: %w", err)
	}
	sm, err := secretManifest(raw, inst.Namespace)
	if err != nil {
		return nil, err
	}
	if _, err := inst.kubectl(ctx, r, bytes.NewReader(sm), "apply", "-f", "-"); err != nil {
		return nil, fmt.Errorf("restore %s: %w", secretsName, err)
	}

	pod, err := inst.primaryPod(ctx, r, platformCluster)
	if err != nil {
		return nil, err
	}
	if err := fileReader(filepath.Join(tmp, "platform.sql"), func(f io.Reader) error {
		_, err := inst.kubectl(ctx, r, io.MultiReader(strings.NewReader("SET ROLE zenith;\n"), f),
			"exec", "-i", pod, "-c", "postgres", "--", "psql", "-U", "postgres", "-d", "zenith", "-v", "ON_ERROR_STOP=1", "-q")
		return err
	}); err != nil {
		return nil, fmt.Errorf("restore platform database: %w", err)
	}

	if err := fileReader(filepath.Join(tmp, "storage.tar"), func(f io.Reader) error {
		_, err := inst.kubectl(ctx, r, f, "exec", "-i", storagePod, "--", "tar", "-C", "/data", "-xf", "-")
		return err
	}); err != nil {
		return nil, fmt.Errorf("restore object storage: %w", err)
	}
	// The storage server reads its access keys from the Secret restored above.
	if _, err := inst.kubectl(ctx, r, nil, "rollout", "restart", "statefulset/zenith-storage"); err != nil {
		return nil, fmt.Errorf("restart object storage: %w", err)
	}

	if _, err := inst.kubectl(ctx, r, nil, "scale", "deploy/zenith-api", "--replicas=1"); err != nil {
		return nil, fmt.Errorf("start the API: %w", err)
	}
	if _, err := inst.kubectl(ctx, r, nil, "rollout", "status", "deploy/zenith-api", "--timeout="+opt.Wait.String()); err != nil {
		return nil, fmt.Errorf("API did not come back: %w", err)
	}

	var errs []error
	for _, db := range m.Databases {
		if err := restoreDB(ctx, r, inst, opt, tmp, db); err != nil {
			errs = append(errs, fmt.Errorf("database %s (%s): %w", db.Name, db.Engine, err))
		}
	}
	return m, errors.Join(errs...)
}

func podReady(ctx context.Context, apps Install, r Runner, pod string) bool {
	s, err := apps.jsonpath(ctx, r, `{.status.conditions[?(@.type=="Ready")].status}`, "get", "pod", pod)
	return err == nil && s == "True"
}

func restoreDB(ctx context.Context, r Runner, inst Install, opt RestoreOptions, tmp string, db DBRef) error {
	if !dbIDPattern.MatchString(db.ID) {
		return fmt.Errorf("invalid database id %q", db.ID)
	}
	apps := inst.apps(db)
	base := filepath.Join(tmp, "databases", db.ID)
	switch db.Engine {
	case "postgresql":
		// The API recreates the instance shortly after it starts; wait for its primary.
		var pod string
		if err := waitFor(ctx, opt, "database "+db.Name, func() bool {
			p, err := apps.primaryPod(ctx, r, db.Resource())
			if err != nil || !podReady(ctx, apps, r, p) {
				return false
			}
			pod = p
			return true
		}); err != nil {
			return err
		}
		return fileReader(base+".sql", func(f io.Reader) error {
			_, err := apps.kubectl(ctx, r, io.MultiReader(strings.NewReader("SET ROLE "+quoteIdent(db.DBUser)+";\n"), f),
				"exec", "-i", pod, "-c", "postgres", "--", "psql", "-U", "postgres", "-d", db.DBName, "-v", "ON_ERROR_STOP=1", "-q")
			return err
		})
	case "mysql":
		pod := db.Resource() + "-0"
		if err := waitFor(ctx, opt, "database "+db.Name, func() bool { return podReady(ctx, apps, r, pod) }); err != nil {
			return err
		}
		pw, err := dbSecret(ctx, apps, r, db, "root-password")
		if err != nil {
			return err
		}
		return fileReader(base+".sql", func(f io.Reader) error {
			_, err := apps.kubectl(ctx, r, f, "exec", "-i", pod, "--", "env", "MYSQL_PWD="+pw, "mariadb", "-uroot", db.DBName)
			return err
		})
	case "redis":
		pod := db.Resource() + "-0"
		if err := waitFor(ctx, opt, "database "+db.Name, func() bool { return podReady(ctx, apps, r, pod) }); err != nil {
			return err
		}
		// Redis loads dump.rdb at start-up, so write it and restart the pod.
		if err := fileReader(base+".rdb", func(f io.Reader) error {
			_, err := apps.kubectl(ctx, r, f, "exec", "-i", pod, "--", "sh", "-c", "cat > /data/dump.rdb")
			return err
		}); err != nil {
			return err
		}
		_, err := apps.kubectl(ctx, r, nil, "delete", "pod", pod, "--wait=false")
		return err
	}
	return fmt.Errorf("unsupported engine %q", db.Engine)
}

// quoteIdent quotes a SQL identifier.
func quoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

var dbIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
