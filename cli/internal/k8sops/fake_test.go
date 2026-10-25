package k8sops

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
)

type call struct{ dir, cmd string }

type fakeRunner struct {
	mu      sync.Mutex
	calls   []call
	answers map[string]string // command prefix -> stdout
	fail    map[string]bool   // command prefix -> error
	stdins  map[string]string // full command -> stdin content (recorded when stdin is given)
}

func (f *fakeRunner) match(cmd string, m map[string]string) (string, bool) {
	best := ""
	for p := range m {
		if strings.HasPrefix(cmd, p) && len(p) > len(best) {
			best = p
		}
	}
	if best == "" {
		return "", false
	}
	return m[best], true
}

func (f *fakeRunner) record(dir, name string, args []string) (string, error) {
	cmd := strings.TrimSpace(name + " " + strings.Join(args, " "))
	f.mu.Lock()
	f.calls = append(f.calls, call{dir, cmd})
	f.mu.Unlock()
	for p := range f.fail {
		if strings.HasPrefix(cmd, p) || (strings.HasPrefix(p, "~") && strings.Contains(cmd, p[1:])) {
			return "", fmt.Errorf("fake failure: %s", p)
		}
	}
	out, _ := f.match(cmd, f.answers)
	return out, nil
}

func (f *fakeRunner) Run(_ context.Context, dir string, stdin io.Reader, name string, args ...string) ([]byte, error) {
	if stdin != nil {
		b, _ := io.ReadAll(stdin)
		f.mu.Lock()
		if f.stdins == nil {
			f.stdins = map[string]string{}
		}
		f.stdins[strings.TrimSpace(name+" "+strings.Join(args, " "))] = string(b)
		f.mu.Unlock()
	}
	out, err := f.record(dir, name, args)
	return []byte(out), err
}

func (f *fakeRunner) RunTo(_ context.Context, dir string, w io.Writer, name string, args ...string) error {
	out, err := f.record(dir, name, args)
	if err == nil {
		_, _ = io.Copy(w, bytes.NewBufferString(out))
	}
	return err
}

func (f *fakeRunner) cmds() []string {
	var s []string
	for _, c := range f.calls {
		s = append(s, c.cmd)
	}
	return s
}
