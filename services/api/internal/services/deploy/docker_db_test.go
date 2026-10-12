//go:build docker

// services/api/internal/services/deploy/docker_db_test.go
package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestDockerDBBackend_RedisLifecycle(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("no docker")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	const net = "zenith-dbtest"
	_, _ = cli.NetworkCreate(ctx, net, network.CreateOptions{})
	defer cli.NetworkRemove(context.Background(), net)

	b := NewDockerDBBackend(cli, net)
	db := &entities.UserDatabase{ID: "deadbeef-0000-0000-0000-000000000000", Engine: entities.DatabaseEngineRedis, DBName: "db_x", DBUser: "u_x"}
	host, port, err := b.Provision(ctx, db, "pw123456789012345678")
	if err != nil {
		t.Fatal(err)
	}
	defer b.Delete(context.Background(), db, true)
	if host != "zenith-db-deadbeef" || port != 6379 {
		t.Errorf("host/port = %s:%d", host, port)
	}
	if err := b.WaitReady(ctx, db); err != nil {
		t.Fatalf("not ready: %v", err)
	}
	insp, err := cli.ContainerInspect(ctx, host)
	if err != nil {
		t.Fatal(err)
	}
	if len(insp.HostConfig.PortBindings) != 0 {
		t.Errorf("database must not publish host ports: %v", insp.HostConfig.PortBindings)
	}
	if insp.HostConfig.RestartPolicy.Name != "unless-stopped" {
		t.Errorf("restart = %q", insp.HostConfig.RestartPolicy.Name)
	}
}
