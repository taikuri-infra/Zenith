//go:build docker

package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestDockerLogSource_Tail(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("no docker")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	const name = "zenith-app-logtest"
	_ = cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})
	created, err := cli.ContainerCreate(ctx, &container.Config{Image: "alpine:3.20", Cmd: []string{"sh", "-c", "echo hello-zenith; sleep 30"}}, nil, nil, nil, name)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.ContainerRemove(context.Background(), created.ID, container.RemoveOptions{Force: true})
	if err := cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond)

	entries, err := NewDockerLogSource(cli).Tail(ctx, &entities.App{Subdomain: "logtest"}, time.Hour, 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Line == "hello-zenith" {
			return
		}
	}
	t.Errorf("hello-zenith not in %+v", entries)
}
