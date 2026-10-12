package deploy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// DockerLogSource reads app logs from the app's container (stdout+stderr).
type DockerLogSource struct{ cli *client.Client }

func NewDockerLogSource(cli *client.Client) *DockerLogSource { return &DockerLogSource{cli: cli} }

func appContainer(app *entities.App) string { return "zenith-app-" + sanitizeName(app.Subdomain) }

func (s *DockerLogSource) open(ctx context.Context, app *entities.App, opts container.LogsOptions) (io.ReadCloser, error) {
	opts.ShowStdout, opts.ShowStderr, opts.Timestamps = true, true, true
	rc, err := s.cli.ContainerLogs(ctx, appContainer(app), opts)
	if err != nil {
		return nil, fmt.Errorf("container logs %s: %w", appContainer(app), err)
	}
	pr, pw := io.Pipe()
	go func() {
		_, err := stdcopy.StdCopy(pw, pw, rc)
		_ = rc.Close()
		_ = pw.CloseWithError(err)
	}()
	return pr, nil
}

func parseDockerLine(line string) dto.MonitoringLogEntry {
	ts, rest, ok := strings.Cut(line, " ")
	if t, err := time.Parse(time.RFC3339Nano, ts); ok && err == nil {
		return dto.MonitoringLogEntry{Timestamp: t, Line: rest}
	}
	return dto.MonitoringLogEntry{Timestamp: time.Now(), Line: line}
}

func (s *DockerLogSource) Tail(ctx context.Context, app *entities.App, since time.Duration, limit int) ([]dto.MonitoringLogEntry, error) {
	r, err := s.open(ctx, app, container.LogsOptions{Since: time.Now().Add(-since).Format(time.RFC3339), Tail: fmt.Sprintf("%d", limit)})
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var out []dto.MonitoringLogEntry
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		out = append(out, parseDockerLine(sc.Text()))
	}
	return out, nil
}

func (s *DockerLogSource) Stream(ctx context.Context, app *entities.App, out chan<- dto.MonitoringLogEntry) error {
	r, err := s.open(ctx, app, container.LogsOptions{Follow: true, Tail: "50"})
	if err != nil {
		return err
	}
	defer r.Close()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		select {
		case out <- parseDockerLine(sc.Text()):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
