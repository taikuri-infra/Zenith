// services/api/internal/services/deploy/docker_db.go
package deploy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/services"
)

var _ services.DBBackend = (*DockerDBBackend)(nil)

// DockerDBBackend runs each managed database as its own container on the
// app network. Data lives in a named volume; the engine port is published on 127.0.0.1 only.
type DockerDBBackend struct {
	cli     *client.Client
	network string
}

func NewDockerDBBackend(cli *client.Client, network string) *DockerDBBackend {
	return &DockerDBBackend{cli: cli, network: network}
}

func dbShortID(db *entities.UserDatabase) string {
	id := db.ID
	if len(id) > 8 {
		id = id[:8]
	}
	return id
}

func dbContainerName(db *entities.UserDatabase) string { return "zenith-db-" + dbShortID(db) }

type dbSpec struct {
	image string
	port  int
	env   []string
	cmd   []string
	data  string
	test  []string
}

func specFor(db *entities.UserDatabase, pw string) (dbSpec, error) {
	switch db.Engine {
	case entities.DatabaseEnginePostgres:
		return dbSpec{image: "postgres:16-alpine", port: 5432, data: "/var/lib/postgresql/data",
			env:  []string{"POSTGRES_DB=" + db.DBName, "POSTGRES_USER=" + db.DBUser, "POSTGRES_PASSWORD=" + pw},
			test: []string{"CMD-SHELL", "pg_isready -U " + db.DBUser + " -d " + db.DBName + " -h 127.0.0.1"}}, nil
	case entities.DatabaseEngineMySQL:
		return dbSpec{image: "mysql:8.4", port: 3306, data: "/var/lib/mysql",
			env:  []string{"MYSQL_DATABASE=" + db.DBName, "MYSQL_USER=" + db.DBUser, "MYSQL_PASSWORD=" + pw, "MYSQL_RANDOM_ROOT_PASSWORD=yes"},
			test: []string{"CMD-SHELL", "mysqladmin ping -h 127.0.0.1 -u" + db.DBUser + " -p$MYSQL_PASSWORD --silent"}}, nil
	case entities.DatabaseEngineRedis:
		return dbSpec{image: "redis:7-alpine", port: 6379, data: "/data",
			cmd:  []string{"sh", "-c", `exec docker-entrypoint.sh redis-server --requirepass "$REDIS_PASSWORD" --appendonly yes`},
			test: []string{"CMD-SHELL", "redis-cli -a \"$REDIS_PASSWORD\" --no-auth-warning ping | grep -q PONG"},
			env:  []string{"REDIS_PASSWORD=" + pw}}, nil
	}
	return dbSpec{}, fmt.Errorf("engine %s not supported by the Docker runtime", db.Engine)
}

// Provision pulls the engine image and starts the container. Idempotent: an
// existing container with the same name is replaced, the volume is kept.
func (b *DockerDBBackend) Provision(ctx context.Context, db *entities.UserDatabase, pw string) (string, int, error) {
	spec, err := specFor(db, pw)
	if err != nil {
		return "", 0, err
	}
	rc, err := b.cli.ImagePull(ctx, spec.image, image.PullOptions{})
	if err != nil {
		return "", 0, fmt.Errorf("pull %s: %w", spec.image, err)
	}
	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()

	name := dbContainerName(db)
	exposed, bindings := loopbackBinding(spec.port)
	_ = b.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})
	resp, err := b.cli.ContainerCreate(ctx,
		&container.Config{
			Image:        spec.image,
			ExposedPorts: exposed,
			Env:          spec.env,
			Cmd:          spec.cmd,
			Labels:       map[string]string{"zenith.db.id": db.ID, "zenith.db.engine": string(db.Engine)},
			Healthcheck: &container.HealthConfig{
				Test: spec.test, Interval: 3 * time.Second, Timeout: 3 * time.Second, Retries: 40, StartPeriod: 5 * time.Second,
			},
		},
		&container.HostConfig{
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			PortBindings:  bindings,
			Mounts:        []mount.Mount{{Type: mount.TypeVolume, Source: name, Target: spec.data}},
		},
		&network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{b.network: {Aliases: []string{name}}}},
		nil, name)
	if err != nil {
		return "", 0, fmt.Errorf("create %s: %w", name, err)
	}
	if err := b.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", 0, fmt.Errorf("start %s: %w", name, err)
	}
	return name, spec.port, nil
}

// loopbackBinding publishes the engine port on 127.0.0.1 only, with a port
// Docker picks. Loopback isn't reachable from the internet; it is reachable
// through an SSH tunnel (zen db connect --tunnel).
func loopbackBinding(port int) (nat.PortSet, nat.PortMap) {
	p := nat.Port(fmt.Sprintf("%d/tcp", port))
	return nat.PortSet{p: struct{}{}}, nat.PortMap{p: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: ""}}}
}

// ErrNoLoopbackPort means the database container has no 127.0.0.1 binding
// (it was created before loopback publishing existed).
var ErrNoLoopbackPort = errors.New("database container has no loopback port")

// HostPort is the loopback port Docker assigned to the engine port.
func (b *DockerDBBackend) HostPort(ctx context.Context, db *entities.UserDatabase) (int, error) {
	spec, err := specFor(db, "")
	if err != nil {
		return 0, err
	}
	insp, err := b.cli.ContainerInspect(ctx, dbContainerName(db))
	if err != nil {
		return 0, err
	}
	if insp.NetworkSettings == nil {
		return 0, fmt.Errorf("no network settings")
	}
	for _, pb := range insp.NetworkSettings.Ports[nat.Port(fmt.Sprintf("%d/tcp", spec.port))] {
		if pb.HostIP == "127.0.0.1" {
			return strconv.Atoi(pb.HostPort)
		}
	}
	return 0, fmt.Errorf("%s: %w", dbContainerName(db), ErrNoLoopbackPort)
}

// WaitReady polls the container health until healthy, unhealthy, or ctx ends.
func (b *DockerDBBackend) WaitReady(ctx context.Context, db *entities.UserDatabase) error {
	name := dbContainerName(db)
	for {
		insp, err := b.cli.ContainerInspect(ctx, name)
		if err != nil {
			return fmt.Errorf("inspect %s: %w", name, err)
		}
		if insp.State != nil && insp.State.Health != nil {
			switch insp.State.Health.Status {
			case "healthy":
				return nil
			case "unhealthy":
				return fmt.Errorf("%s is unhealthy", name)
			}
		}
		if insp.State != nil && !insp.State.Running && insp.State.ExitCode != 0 {
			return fmt.Errorf("%s exited with code %d", name, insp.State.ExitCode)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s not ready: %w", name, ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}
}

// Delete removes the container; purge also removes its data volume.
func (b *DockerDBBackend) Delete(ctx context.Context, db *entities.UserDatabase, purge bool) error {
	name := dbContainerName(db)
	if err := b.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true}); err != nil && !client.IsErrNotFound(err) {
		return fmt.Errorf("remove %s: %w", name, err)
	}
	if purge {
		if err := b.cli.VolumeRemove(ctx, name, true); err != nil && !client.IsErrNotFound(err) {
			return fmt.Errorf("remove volume %s: %w", name, err)
		}
	}
	return nil
}

// ContainerRunning reports whether the database container exists and runs.
func (b *DockerDBBackend) ContainerRunning(ctx context.Context, db *entities.UserDatabase) (bool, error) {
	insp, err := b.cli.ContainerInspect(ctx, dbContainerName(db))
	if client.IsErrNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return insp.State != nil && (insp.State.Running || insp.State.Restarting), nil
}
