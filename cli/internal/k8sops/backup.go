package k8sops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manifest describes a backup archive.
type Manifest struct {
	Format    int     `json:"format"`     // 2: Kubernetes
	Version   string  `json:"version"`    // image tag the API ran
	CreatedAt string  `json:"created_at"` // RFC3339 UTC
	Databases []DBRef `json:"databases"`
}

const (
	platformCluster = "zenith-postgres"
	secretsName     = "zenith-secrets" // nosemgrep: go-hardcoded-password -- Kubernetes Secret resource name, not secret data
	storagePod      = "zenith-storage-0"
)

func dumpTo(ctx context.Context, i Install, r Runner, path string, args ...string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := i.kubectlTo(ctx, r, f, args...); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Backup writes one archive into outDir and returns its path. It holds the
// platform database, the object storage volume, every ready managed database
// and the zenith-secrets Secret (the encryption keys the data needs).
func Backup(ctx context.Context, r Runner, inst Install, outDir string, now time.Time) (path string, err error) {
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(outDir, ".zenith-backup-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	dbs, err := inst.Databases(ctx, r)
	if err != nil {
		return "", err
	}
	version := inst.Version(ctx, r)

	pod, err := inst.primaryPod(ctx, r, platformCluster)
	if err != nil {
		return "", err
	}
	if err = dumpTo(ctx, inst, r, filepath.Join(tmp, "platform.sql"), "exec", pod, "-c", "postgres", "--",
		"pg_dump", "-U", "postgres", "-d", "zenith", "--clean", "--if-exists", "--no-owner", "--no-privileges"); err != nil {
		return "", fmt.Errorf("dump platform database: %w", err)
	}
	if err = dumpTo(ctx, inst, r, filepath.Join(tmp, "storage.tar"), "exec", storagePod, "--",
		"tar", "-C", "/data", "-cf", "-", "."); err != nil {
		return "", fmt.Errorf("archive object storage: %w", err)
	}
	if err = dumpTo(ctx, inst, r, filepath.Join(tmp, "secrets.json"), "get", "secret", secretsName, "-o", "json"); err != nil {
		return "", fmt.Errorf("read %s: %w", secretsName, err)
	}

	if err = os.MkdirAll(filepath.Join(tmp, "databases"), 0o700); err != nil {
		return "", err
	}
	for _, db := range dbs {
		if err = backupDB(ctx, inst, r, tmp, db); err != nil {
			return "", fmt.Errorf("back up database %s (%s): %w", db.Name, db.Engine, err)
		}
	}

	if dbs == nil {
		dbs = []DBRef{}
	}
	mb, err := json.MarshalIndent(Manifest{Format: 2, Version: version, CreatedAt: now.UTC().Format(time.RFC3339), Databases: dbs}, "", "  ")
	if err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(tmp, "manifest.json"), mb, 0o600); err != nil {
		return "", err
	}

	if version == "" {
		version = "unknown"
	}
	archive := filepath.Join(outDir, fmt.Sprintf("zenith-backup-%s-%s.tar.gz", version, now.UTC().Format("20060102T150405Z")))
	if err = writeArchive(archive, tmp); err != nil {
		return "", err
	}
	return archive, nil
}

// appsKubectl runs kubectl against the apps namespace.
func (i Install) apps() Install { return Install{Kubeconfig: i.Kubeconfig, Namespace: AppsNamespace} }

// dbSecret reads one key of the database's credentials Secret.
func dbSecret(ctx context.Context, apps Install, r Runner, db DBRef, key string) (string, error) {
	out, err := apps.jsonpath(ctx, r, "{.data."+key+"}", "get", "secret", db.Resource()+"-auth")
	if err != nil {
		return "", err
	}
	return decodeB64(out)
}

func backupDB(ctx context.Context, inst Install, r Runner, tmp string, db DBRef) error {
	apps := inst.apps()
	base := filepath.Join(tmp, "databases", db.ID)
	switch db.Engine {
	case "postgresql":
		pod, err := apps.primaryPod(ctx, r, db.Resource())
		if err != nil {
			return err
		}
		return dumpTo(ctx, apps, r, base+".sql", "exec", pod, "-c", "postgres", "--", "pg_dump", "-U", "postgres",
			"-d", db.DBName, "--clean", "--if-exists", "--no-owner", "--no-privileges")
	case "mysql":
		pw, err := dbSecret(ctx, apps, r, db, "root-password")
		if err != nil {
			return err
		}
		return dumpTo(ctx, apps, r, base+".sql", "exec", db.Resource()+"-0", "--", "env", "MYSQL_PWD="+pw,
			"mariadb-dump", "-uroot", "--single-transaction", "--no-tablespaces", db.DBName)
	case "redis":
		pw, err := dbSecret(ctx, apps, r, db, "password")
		if err != nil {
			return err
		}
		out, err := apps.kubectl(ctx, r, nil, "exec", db.Resource()+"-0", "--", "redis-cli", "-a", pw, "--no-auth-warning", "SAVE")
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(out)) != "OK" {
			return fmt.Errorf("redis SAVE did not return OK: %q", strings.TrimSpace(string(out)))
		}
		return dumpTo(ctx, apps, r, base+".rdb", "exec", db.Resource()+"-0", "--", "cat", "/data/dump.rdb")
	}
	return fmt.Errorf("unsupported engine %q", db.Engine)
}
