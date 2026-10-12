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

func TestDeployedImageLegacyGitSHA(t *testing.T) {
	cases := map[string]string{
		"project-deploy":  "",
		"abc123def":       "",
		"docker.io/x/y:1": "docker.io/x/y:1",
		"nginx:1":         "nginx:1",
		"":                "",
	}
	for sha, want := range cases {
		if got := (&Deployment{GitSHA: sha}).DeployedImage(); got != want {
			t.Errorf("GitSHA %q: got %q, want %q", sha, got, want)
		}
	}
}
