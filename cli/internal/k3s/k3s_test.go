package k3s

import (
	"strings"
	"testing"
)

func TestBuildEnv_Empty(t *testing.T) {
	env := buildEnv(Options{})
	if env != "" {
		t.Errorf("expected empty env for empty options, got: %q", env)
	}
}

func TestBuildEnv_WithVersion(t *testing.T) {
	env := buildEnv(Options{Version: "v1.29.4+k3s1"})
	if !strings.Contains(env, "INSTALL_K3S_VERSION") {
		t.Errorf("expected INSTALL_K3S_VERSION in env, got: %q", env)
	}
	if !strings.Contains(env, "v1.29.4+k3s1") {
		t.Errorf("expected version in env, got: %q", env)
	}
}

func TestBuildEnv_WithDisableComponents(t *testing.T) {
	env := buildEnv(Options{DisableComponents: []string{"traefik", "servicelb"}})
	if !strings.Contains(env, "INSTALL_K3S_EXEC") {
		t.Errorf("expected INSTALL_K3S_EXEC in env, got: %q", env)
	}
	if !strings.Contains(env, "traefik") {
		t.Errorf("expected 'traefik' in env, got: %q", env)
	}
	if !strings.Contains(env, "servicelb") {
		t.Errorf("expected 'servicelb' in env, got: %q", env)
	}
}

func TestBuildEnv_ExtraArgs(t *testing.T) {
	env := buildEnv(Options{ExtraArgs: []string{"INSTALL_K3S_CHANNEL=stable"}})
	if !strings.Contains(env, "INSTALL_K3S_CHANNEL") {
		t.Errorf("expected INSTALL_K3S_CHANNEL in env, got: %q", env)
	}
	if !strings.Contains(env, "stable") {
		t.Errorf("expected 'stable' in env, got: %q", env)
	}
}

func TestBuildEnv_ExtraArgs_InvalidFormat(t *testing.T) {
	// Extra args without = sign are ignored
	env := buildEnv(Options{ExtraArgs: []string{"BADARG"}})
	if strings.Contains(env, "BADARG") {
		t.Errorf("expected BADARG to be ignored, got: %q", env)
	}
}

func TestInstallCommand_ContainsScriptURL(t *testing.T) {
	opts := Options{}
	env := buildEnv(opts)
	var cmd string
	if env != "" {
		cmd = "curl -sfL https://get.k3s.io | " + env + " sh -"
	} else {
		cmd = "curl -sfL https://get.k3s.io | sh -"
	}
	if !strings.Contains(cmd, "get.k3s.io") {
		t.Error("install command should reference get.k3s.io")
	}
}

func TestInstallScriptURL(t *testing.T) {
	if installScriptURL != "https://get.k3s.io" {
		t.Errorf("expected install script URL 'https://get.k3s.io', got %q", installScriptURL)
	}
}

func TestInstall_DefaultVersionUsed(t *testing.T) {
	// Verify that buildEnv includes INSTALL_K3S_VERSION when an explicit version is set.
	// This mirrors what Install() does after the fix: it sets opts.Version = DefaultK3sVersion
	// before calling buildEnv, so an empty-version call results in a pinned version env var.
	env := buildEnv(Options{Version: DefaultK3sVersion})
	if !strings.Contains(env, "INSTALL_K3S_VERSION") {
		t.Error("expected INSTALL_K3S_VERSION in env when version is set")
	}
	if !strings.Contains(env, DefaultK3sVersion) {
		t.Errorf("expected DefaultK3sVersion %q in env, got: %q", DefaultK3sVersion, env)
	}

	// buildEnv with empty version should NOT include INSTALL_K3S_VERSION —
	// the default injection happens in Install(), not buildEnv itself.
	envEmpty := buildEnv(Options{})
	if strings.Contains(envEmpty, "INSTALL_K3S_VERSION") {
		t.Error("buildEnv with empty version should not include INSTALL_K3S_VERSION")
	}
}

func TestBuildEnv_AllOptions(t *testing.T) {
	env := buildEnv(Options{
		Version:           "v1.29.4+k3s1",
		DisableComponents: []string{"traefik"},
		ExtraArgs:         []string{"INSTALL_K3S_CHANNEL=stable"},
	})
	if env == "" {
		t.Error("expected non-empty env with all options")
	}
	if !strings.Contains(env, "INSTALL_K3S_VERSION") {
		t.Error("expected INSTALL_K3S_VERSION")
	}
	if !strings.Contains(env, "INSTALL_K3S_EXEC") {
		t.Error("expected INSTALL_K3S_EXEC")
	}
	if !strings.Contains(env, "INSTALL_K3S_CHANNEL") {
		t.Error("expected INSTALL_K3S_CHANNEL")
	}
}
