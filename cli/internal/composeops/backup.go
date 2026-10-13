package composeops

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manifest describes a backup archive.
type Manifest struct {
	Format    int     `json:"format"`     // 1
	Version   string  `json:"version"`    // ZENITH_VERSION of the install
	CreatedAt string  `json:"created_at"` // RFC3339 UTC
	Databases []DBRef `json:"databases"`
}

func dumpTo(ctx context.Context, r Runner, dir, path, name string, args ...string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := r.RunTo(ctx, dir, f, name, args...); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Backup writes one archive into outDir and returns its path.
func Backup(ctx context.Context, r Runner, inst Install, outDir string, now time.Time) (path string, err error) {
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(outDir, ".zenith-backup-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	defer func() {
		if err != nil && path != "" {
			os.Remove(path)
			path = ""
		}
	}()

	dbs, err := inst.Databases(ctx, r)
	if err != nil {
		return "", err
	}
	version := inst.EnvGet("ZENITH_VERSION")

	if err = dumpTo(ctx, r, inst.Dir, filepath.Join(tmp, "platform.sql"), "docker", "compose", "exec", "-T", "postgres",
		"pg_dump", "-U", "zenith", "-d", "zenith", "--clean", "--if-exists"); err != nil {
		return "", fmt.Errorf("dump platform database: %w", err)
	}

	cidOut, err := r.Run(ctx, inst.Dir, nil, "docker", "compose", "ps", "-q", "storage")
	if err != nil {
		return "", fmt.Errorf("find storage container: %w", err)
	}
	cid := strings.TrimSpace(string(cidOut))
	volOut, err := r.Run(ctx, inst.Dir, nil, "docker", "inspect", "-f",
		`{{range .Mounts}}{{if eq .Destination "/data"}}{{.Name}}{{end}}{{end}}`, cid)
	if err != nil {
		return "", fmt.Errorf("find storage volume: %w", err)
	}
	vol := strings.TrimSpace(string(volOut))
	if err = dumpTo(ctx, r, inst.Dir, filepath.Join(tmp, "storage.tar"), "docker", "run", "--rm",
		"-v", vol+":/data:ro", "alpine:3.20", "tar", "-C", "/data", "-cf", "-", "."); err != nil {
		return "", fmt.Errorf("archive storage volume: %w", err)
	}

	if err = os.MkdirAll(filepath.Join(tmp, "databases"), 0o700); err != nil {
		return "", err
	}
	for _, db := range dbs {
		c := DBContainer(db.ID)
		base := filepath.Join(tmp, "databases", db.ID)
		switch db.Engine {
		case "postgresql":
			user, e := r.Run(ctx, inst.Dir, nil, "docker", "exec", c, "printenv", "POSTGRES_USER")
			if e != nil {
				return "", fmt.Errorf("database %s: %w", db.Name, e)
			}
			name, e := r.Run(ctx, inst.Dir, nil, "docker", "exec", c, "printenv", "POSTGRES_DB")
			if e != nil {
				return "", fmt.Errorf("database %s: %w", db.Name, e)
			}
			err = dumpTo(ctx, r, inst.Dir, base+".sql", "docker", "exec", c, "pg_dump",
				"-U", strings.TrimSpace(string(user)), "-d", strings.TrimSpace(string(name)), "--clean", "--if-exists")
		case "mysql":
			err = dumpTo(ctx, r, inst.Dir, base+".sql", "docker", "exec", c, "sh", "-c",
				`exec mysqldump -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" --single-transaction --no-tablespaces "$MYSQL_DATABASE"`)
		case "redis":
			if _, err = r.Run(ctx, inst.Dir, nil, "docker", "exec", c, "sh", "-c",
				`redis-cli -a "$REDIS_PASSWORD" --no-auth-warning SAVE`); err == nil {
				// Pre-create the target so it exists with 0600; docker cp overwrites it.
				if err = os.WriteFile(base+".rdb", nil, 0o600); err == nil {
					_, err = r.Run(ctx, inst.Dir, nil, "docker", "cp", c+":/data/dump.rdb", base+".rdb")
				}
			}
		default:
			err = fmt.Errorf("unsupported engine %q", db.Engine)
		}
		if err != nil {
			return "", fmt.Errorf("back up database %s (%s): %w", db.Name, db.Engine, err)
		}
	}

	envBytes, err := os.ReadFile(filepath.Join(inst.Dir, ".env"))
	if err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(tmp, "env"), envBytes, 0o600); err != nil {
		return "", err
	}
	if dbs == nil {
		dbs = []DBRef{}
	}
	mb, err := json.MarshalIndent(Manifest{Format: 1, Version: version, CreatedAt: now.UTC().Format(time.RFC3339), Databases: dbs}, "", "  ")
	if err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(tmp, "manifest.json"), mb, 0o600); err != nil {
		return "", err
	}

	path = filepath.Join(outDir, fmt.Sprintf("zenith-backup-%s-%s.tar.gz", version, now.UTC().Format("20060102T150405Z")))
	if err = writeArchive(path, tmp); err != nil {
		return "", err
	}
	return path, nil
}

func writeArchive(path, src string) (err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel, _ := filepath.Rel(src, p)
		if rel == "." || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		h, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		h.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(tw, in)
		return err
	})
	if err != nil {
		return err
	}
	if err = tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}
