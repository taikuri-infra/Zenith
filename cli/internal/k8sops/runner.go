package k8sops

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
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
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command -- name and args are assembled by k8sops from constants plus validated identifiers and version strings; no raw user input reaches this call.
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

// readerOrNil and writer keep Install's helper signatures short.
type (
	readerOrNil = io.Reader
	writer      = io.Writer
)
