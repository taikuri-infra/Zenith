package entities

import "testing"

func TestDeployedImage(t *testing.T) {
	if got := (&Deployment{ImageTag: "a:1", GitSHA: "b:2"}).DeployedImage(); got != "a:1" {
		t.Errorf("got %q", got)
	}
	if got := (&Deployment{GitSHA: "docker.io/traefik/whoami:latest"}).DeployedImage(); got != "docker.io/traefik/whoami:latest" {
		t.Errorf("got %q", got)
	}
}
