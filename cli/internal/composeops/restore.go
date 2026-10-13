package composeops

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var dbIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// RestoreOptions tunes the readiness polling of Restore.
type RestoreOptions struct {
	Wait time.Duration // default 5m
	Poll time.Duration // default 3s
}

// secretKeys are the only .env keys copied from the archive.
var secretKeys = []string{
	"JWT_SECRET", "SECRETS_ENCRYPTION_KEY", "OLD_SECRETS_KEYS", "DB_PASSWORD",
	"S3_ACCESS_KEY", "S3_SECRET_KEY", "ADMIN_EMAIL", "ADMIN_PASSWORD",
}

// extractArchive unpacks a backup archive into dst, rejecting unsafe entries.
func extractArchive(archive, dst string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		name := filepath.ToSlash(h.Name)
		if name == "" || strings.HasPrefix(name, "/") || filepath.IsAbs(h.Name) {
			return fmt.Errorf("unsafe path in archive: %q", h.Name)
		}
		for _, part := range strings.Split(name, "/") {
			if part == ".." {
				return fmt.Errorf("unsafe path in archive: %q", h.Name)
			}
		}
		target := filepath.Join(dst, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported entry type in archive: %q", h.Name)
		}
	}
}

func parseEnv(path string) map[string]string {
	m := map[string]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			m[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return m
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

func stdinFile(path string, fn func(*os.File) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}

// Restore rebuilds the install from an archive written by Backup.
func Restore(ctx context.Context, r Runner, inst Install, archive string, opt RestoreOptions) (*Manifest, error) {
	if opt.Wait <= 0 {
		opt.Wait = 5 * time.Minute
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
	mb, err := os.ReadFile(filepath.Join(tmp, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("archive has no manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(mb, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}
	if m.Format != 1 {
		return nil, fmt.Errorf("unsupported archive format %d", m.Format)
	}
	cur := inst.EnvGet("ZENITH_VERSION")
	if cur == "" {
		fmt.Fprintln(os.Stderr, "warning: install has no ZENITH_VERSION; skipping the newer-archive check")
	} else if CompareVersions(m.Version, cur) > 0 {
		return nil, fmt.Errorf("archive is from Zenith %s, newer than this install (%s) — upgrade first", m.Version, cur)
	}
	for _, d := range m.Databases {
		if !dbIDPattern.MatchString(d.ID) {
			return nil, fmt.Errorf("invalid database id in manifest: %q", d.ID)
		}
	}
	required := []string{"platform.sql", "storage.tar", "env"}
	for _, d := range m.Databases {
		switch d.Engine {
		case "redis":
			required = append(required, "databases/"+d.ID+".rdb")
		default:
			required = append(required, "databases/"+d.ID+".sql")
		}
	}
	for _, f := range required {
		if _, err := os.Stat(filepath.Join(tmp, filepath.FromSlash(f))); err != nil {
			return nil, fmt.Errorf("archive incomplete: missing %s", f)
		}
	}

	dc := func(args ...string) error {
		_, err := r.Run(ctx, inst.Dir, nil, "docker", append([]string{"compose"}, args...)...)
		return err
	}
	if err := dc("stop", "api", "web"); err != nil {
		return nil, fmt.Errorf("stop api and web: %w", err)
	}
	platformTouched := false
	stopped := func(step string, err error) error {
		if platformTouched {
			return fmt.Errorf("restore stopped at %s: %w — the platform database was dropped or only partially loaded, so Zenith's data is incomplete until the restore is re-run; fix the cause and re-run 'zen backup restore' (do not start the stack as it is)", step, err)
		}
		return fmt.Errorf("restore stopped at %s: %w — Zenith's API and dashboard are stopped; fix the cause and re-run 'zen backup restore', or run 'docker compose up -d' in %s to start them as they are", step, err, inst.Dir)
	}
	archEnv := parseEnv(filepath.Join(tmp, "env"))
	for _, k := range secretKeys {
		if v, ok := archEnv[k]; ok {
			if err := inst.EnvSet(k, v); err != nil {
				return nil, stopped("restoring "+k, err)
			}
		}
	}
	if err := dc("up", "-d", "postgres", "storage"); err != nil {
		return nil, stopped("starting postgres and storage", err)
	}
	if err := waitFor(ctx, opt, "postgres", func() bool {
		return dc("exec", "-T", "postgres", "pg_isready", "-U", "zenith") == nil
	}); err != nil {
		return nil, stopped("waiting for postgres", err)
	}
	if pw := inst.EnvGet("DB_PASSWORD"); pw != "" {
		// SQL goes over stdin so the password is not in argv or error text.
		q := "ALTER USER zenith PASSWORD '" + strings.ReplaceAll(pw, "'", "''") + "';"
		if _, err := r.Run(ctx, inst.Dir, strings.NewReader(q), "docker", "compose", "exec", "-T", "postgres",
			"psql", "-U", "zenith", "-d", "postgres"); err != nil {
			return nil, stopped("resetting the control-plane password", err)
		}
	}
	platformTouched = true
	if err := dc("exec", "-T", "postgres", "psql", "-U", "zenith", "-d", "postgres",
		"-c", "DROP DATABASE IF EXISTS zenith WITH (FORCE)", "-c", "CREATE DATABASE zenith OWNER zenith"); err != nil {
		return nil, stopped("recreating the platform database", err)
	}
	if err := stdinFile(filepath.Join(tmp, "platform.sql"), func(f *os.File) error {
		_, err := r.Run(ctx, inst.Dir, f, "docker", "compose", "exec", "-T", "postgres",
			"psql", "-U", "zenith", "-d", "zenith", "-v", "ON_ERROR_STOP=1")
		return err
	}); err != nil {
		return nil, stopped("loading the platform database", err)
	}
	platformTouched = false // platform data is fully loaded; only storage remains

	cidOut, err := r.Run(ctx, inst.Dir, nil, "docker", "compose", "ps", "-q", "storage")
	if err != nil {
		return nil, stopped("finding the storage container", err)
	}
	cid := strings.TrimSpace(string(cidOut))
	if cid == "" {
		return nil, stopped("finding the storage container", fmt.Errorf("storage container not found"))
	}
	vol, err := mountVolume(ctx, r, inst.Dir, cid)
	if err != nil {
		return nil, stopped("finding the storage volume", err)
	}
	if err := dc("stop", "storage"); err != nil {
		return nil, stopped("stopping storage", err)
	}
	if _, err := r.Run(ctx, inst.Dir, nil, "docker", "run", "--rm", "-v", vol+":/data", "-v", tmp+":/in:ro",
		"alpine:3.20", "sh", "-c", "find /data -mindepth 1 -delete && tar -C /data -xf /in/storage.tar"); err != nil {
		return nil, stopped("restoring the storage volume", err)
	}
	if err := dc("up", "-d"); err != nil {
		return nil, stopped("starting the stack", err)
	}

	var errs []error
	for _, db := range m.Databases {
		if err := restoreDB(ctx, r, inst, opt, tmp, db); err != nil {
			errs = append(errs, fmt.Errorf("database %s (%s): %w", db.Name, db.Engine, err))
		}
	}
	return &m, errors.Join(errs...)
}

const redisRestoreScript = `rm -rf /data/appendonlydir && cp "/in/$1.rdb" /data/dump.rdb && ` +
	`redis-server --dir /data --appendonly no --daemonize yes --port 6399 >/dev/null && sleep 1 && ` +
	`redis-cli -p 6399 config set appendonly yes >/dev/null && ` +
	`ok=0; for i in $(seq 1 50); do ` +
	`if redis-cli -p 6399 info persistence | grep -q "aof_rewrite_in_progress:0" && redis-cli -p 6399 info persistence | grep -q "aof_rewrite_scheduled:0"; then ok=1; break; fi; sleep 0.2; done; ` +
	`redis-cli -p 6399 shutdown; chown -R redis:redis /data; ` +
	`[ "$ok" = 1 ] || { echo "AOF rewrite did not finish" >&2; exit 1; }`

func mountVolume(ctx context.Context, r Runner, dir, container string) (string, error) {
	out, err := r.Run(ctx, dir, nil, "docker", "inspect", "-f",
		`{{range .Mounts}}{{if eq .Destination "/data"}}{{.Name}}{{end}}{{end}}`, container)
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "", fmt.Errorf("no /data volume on %s", container)
	}
	return v, nil
}

func restoreDB(ctx context.Context, r Runner, inst Install, opt RestoreOptions, tmp string, db DBRef) error {
	c := DBContainer(db.ID)
	if err := waitFor(ctx, opt, c+" to become healthy", func() bool {
		out, err := r.Run(ctx, inst.Dir, nil, "docker", "inspect", "-f", "{{.State.Health.Status}}", c)
		return err == nil && strings.TrimSpace(string(out)) == "healthy"
	}); err != nil {
		return err
	}
	dbDir := filepath.Join(tmp, "databases")
	switch db.Engine {
	case "postgresql", "mysql":
		args := []string{"exec", "-i", c, "sh", "-c"}
		if db.Engine == "postgresql" {
			args = append(args, `psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1`)
		} else {
			args = append(args, `exec mysql -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE"`)
		}
		return stdinFile(filepath.Join(dbDir, db.ID+".sql"), func(f *os.File) error {
			_, err := r.Run(ctx, inst.Dir, f, "docker", args...)
			return err
		})
	case "redis":
		vol, err := mountVolume(ctx, r, inst.Dir, c)
		if err != nil {
			return err
		}
		if _, err := r.Run(ctx, inst.Dir, nil, "docker", "stop", c); err != nil {
			return err
		}
		// With appendonly yes Redis 7 ignores dump.rdb, so load it in a throwaway
		// server and let that server rewrite the AOF before shutting down.
		_, cpErr := r.Run(ctx, inst.Dir, nil, "docker", "run", "--rm", "-v", vol+":/data", "-v", dbDir+":/in:ro",
			"redis:7-alpine", "sh", "-c", redisRestoreScript, "sh", db.ID)
		_, startErr := r.Run(ctx, inst.Dir, nil, "docker", "start", c)
		if cpErr != nil {
			return cpErr
		}
		return startErr
	}
	return fmt.Errorf("unsupported engine %q", db.Engine)
}

// CompareVersions compares two semver strings (with optional prerelease): -1, 0 or 1.
func CompareVersions(a, b string) int {
	split := func(v string) ([]string, []string) {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		core, pre, hasPre := strings.Cut(v, "-")
		var p []string
		if hasPre {
			p = strings.Split(pre, ".")
		}
		return strings.Split(core, "."), p
	}
	ac, ap := split(a)
	bc, bp := split(b)
	for i := 0; i < len(ac) || i < len(bc); i++ {
		var x, y int
		if i < len(ac) {
			x, _ = strconv.Atoi(ac[i])
		}
		if i < len(bc) {
			y, _ = strconv.Atoi(bc[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	switch {
	case ap == nil && bp == nil:
		return 0
	case ap == nil:
		return 1
	case bp == nil:
		return -1
	}
	for i := 0; i < len(ap) || i < len(bp); i++ {
		if i >= len(ap) {
			return -1
		}
		if i >= len(bp) {
			return 1
		}
		x, xe := strconv.Atoi(ap[i])
		y, ye := strconv.Atoi(bp[i])
		switch {
		case xe == nil && ye == nil:
			if x != y {
				if x < y {
					return -1
				}
				return 1
			}
		case xe == nil:
			return -1
		case ye == nil:
			return 1
		default:
			if c := strings.Compare(ap[i], bp[i]); c != 0 {
				return c
			}
		}
	}
	return 0
}
