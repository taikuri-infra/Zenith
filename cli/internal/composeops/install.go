// Package composeops implements operations on a Docker Compose Zenith install
// (backup, restore, upgrade) by driving docker through a Runner.
package composeops

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner executes external commands.
type Runner interface {
	// Run executes name+args in dir. stdin may be nil; stdout is returned.
	Run(ctx context.Context, dir string, stdin io.Reader, name string, args ...string) ([]byte, error)
	// RunTo executes and streams stdout into w (large dumps).
	RunTo(ctx context.Context, dir string, w io.Writer, name string, args ...string) error
}

// ExecRunner is the os/exec implementation of Runner.
type ExecRunner struct{}

type tailBuffer struct {
	buf []byte
	max int
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.max {
		t.buf = t.buf[len(t.buf)-t.max:]
	}
	return len(p), nil
}

func (ExecRunner) exec(ctx context.Context, dir string, stdin io.Reader, stdout io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	stderr := &tailBuffer{max: 2048}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(stderr.buf)))
	}
	return nil
}

// Run implements Runner.
func (e ExecRunner) Run(ctx context.Context, dir string, stdin io.Reader, name string, args ...string) ([]byte, error) {
	var out bytes.Buffer
	err := e.exec(ctx, dir, stdin, &out, name, args...)
	return out.Bytes(), err
}

// RunTo implements Runner.
func (e ExecRunner) RunTo(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
	return e.exec(ctx, dir, nil, w, name, args...)
}

// Install is a Compose install directory.
type Install struct{ Dir string }

func isInstall(dir string) bool {
	if dir == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); err != nil {
		return false
	}
	b, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	return err == nil && bytes.Contains(b, []byte("zenith-api"))
}

// Detect finds a Compose install. If flagDir is set it is the only candidate.
func Detect(flagDir string) (Install, bool) {
	var cands []string
	if flagDir != "" {
		cands = []string{flagDir}
	} else {
		if wd, err := os.Getwd(); err == nil {
			cands = append(cands, wd)
		}
		if home, err := os.UserHomeDir(); err == nil {
			cands = append(cands, filepath.Join(home, "zenith"))
		}
		cands = append(cands, "/root/zenith")
	}
	for _, c := range cands {
		if isInstall(c) {
			return Install{Dir: c}, true
		}
	}
	return Install{}, false
}

func (i Install) envPath() string { return filepath.Join(i.Dir, ".env") }

// EnvGet returns the value of key in .env (last occurrence wins), or "".
func (i Install) EnvGet(key string) string {
	b, err := os.ReadFile(i.envPath())
	if err != nil {
		return ""
	}
	val := ""
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(k) == key {
			val = v
		}
	}
	return val
}

// EnvSet sets key=value in .env, replacing the line or appending, atomically.
func (i Install) EnvSet(key, value string) error {
	b, err := os.ReadFile(i.envPath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var lines []string
	if len(b) > 0 {
		lines = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	}
	found := false
	for n, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") {
			continue
		}
		if k, _, ok := strings.Cut(t, "="); ok && strings.TrimSpace(k) == key {
			lines[n] = key + "=" + value
			found = true
		}
	}
	if !found {
		lines = append(lines, key+"="+value)
	}
	return i.writeEnv(lines)
}

func (i Install) writeEnv(lines []string) error {
	tmp, err := os.CreateTemp(i.Dir, ".env.tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), i.envPath())
}

// EnvUnset removes every line setting key from .env, atomically.
func (i Install) EnvUnset(key string) error {
	b, err := os.ReadFile(i.envPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var out []string
	for _, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "#") {
			if k, _, ok := strings.Cut(t, "="); ok && strings.TrimSpace(k) == key {
				continue
			}
		}
		out = append(out, line)
	}
	return i.writeEnv(out)
}

// DBRef is a managed database. Engine: postgresql|mysql|redis.
type DBRef struct{ ID, Engine, Name string }

// DBContainer returns the container name of a managed database.
func DBContainer(id string) string {
	return "zenith-db-" + id[:min(8, len(id))]
}

// Databases lists ready managed databases from the control-plane Postgres.
func (i Install) Databases(ctx context.Context, r Runner) ([]DBRef, error) {
	out, err := r.Run(ctx, i.Dir, nil, "docker", "compose", "exec", "-T", "postgres", "psql",
		"-U", "zenith", "-d", "zenith", "-At", "-F", "|", "-c",
		"SELECT id, engine, name FROM user_databases WHERE status = 'ready' ORDER BY created_at")
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}
	var dbs []DBRef
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p := strings.SplitN(line, "|", 3)
		if len(p) != 3 {
			return nil, fmt.Errorf("unexpected database row %q", line)
		}
		dbs = append(dbs, DBRef{ID: p[0], Engine: p[1], Name: p[2]})
	}
	return dbs, nil
}
