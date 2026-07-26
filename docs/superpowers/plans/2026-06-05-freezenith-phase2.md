# FreeZenith Phase 2: Production-Ready Installer + Upgrade

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 4 critical bugs that make `zen install` non-functional (state never saved, SSH key never saved, admin password never reaches Helm, Helm chart never installed), fix `zen upgrade` to use the correct chart reference, and add the spec's remaining requirements (version compatibility, pre-flight checks, helm-diff dry-run, `createFirstCluster` login).

**Architecture:**
- All Helm operations run on the **remote server via SSH** — no local `helm` binary required.
- Admin password generated **before** install steps, passed to Helm via a base64-encoded temp values file, cleaned up after install.
- `installstate.State` stores SSH key path, version, and admin token so `zen upgrade` and `zen status` work without re-login.

**Tech Stack:** Go, Cobra CLI, SSH (`golang.org/x/crypto/ssh`), k3s, Helm OCI (`oci://ghcr.io/dotechhq/zenith/charts/zenith`), CloudNativePG

---

## File Map

| File | Change |
|------|--------|
| `cli/internal/install/installer.go` | Fix `SaveTo(state,"")` bug, save SSH key, add `dialSSH`, add `installZenithChart`, add step, fix `BuildResult` password, export `GeneratePassword` |
| `cli/internal/install/installer_test.go` | Tests for new functions and step count |
| `cli/internal/installstate/state.go` | Add `ZenithVersion`, `AdminToken` fields |
| `cli/internal/installstate/state_test.go` | New — round-trip save/load tests |
| `cli/internal/semver/semver.go` | New — version parsing + compatibility check |
| `cli/internal/semver/semver_test.go` | New — semver tests |
| `cli/internal/api/client.go` | Add `Login` method |
| `cli/internal/api/client_test.go` | Test Login method |
| `cli/cmd/install/install.go` | Generate password before step loop |
| `cli/cmd/upgrade/upgrade.go` | Fix chart ref, add pre-flight step, version compat, helm-diff dry-run |

---

### Task 1: Fix State Persistence Bug + Save SSH Private Key

**Files:**
- Modify: `cli/internal/install/installer.go`
- Modify: `cli/internal/installstate/state.go`
- Create: `cli/internal/installstate/state_test.go`
- Modify: `cli/internal/install/installer_test.go`

Bug 1 (`installer.go:279`): `BuildResult` calls `installstate.SaveTo(state, "")` — the empty string path makes `filepath.Dir("")` return `"."` and `os.WriteFile("", ...)` fail silently due to `_ =`.
Bug 2: `provisionHetznerServer` sets `cfg.GeneratedSSHPrivateKey` but never writes it to disk. `zen upgrade` cannot SSH back in without it.
Refactor: extract private `dialSSH(cfg *Config)` used by both `installPlatform` and `verifyExistingServer` to remove duplicated SSH setup code.

- [ ] **Step 1: Write failing test — state persistence**

Create `cli/internal/installstate/state_test.go`:
```go
package installstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-state.yaml")

	s := &State{
		Domain:            "example.com",
		ServerIP:          "1.2.3.4",
		MissionControlURL: "https://mission.example.com",
		CloudURL:          "https://cloud.example.com",
		AdminUser:         "admin",
		AdminPassword:     "secret123",
		SSHKeyPath:        "/home/user/.zen/install-key.pem",
		Provider:          "hetzner",
		Region:            "fsn1",
		InstalledAt:       time.Now().UTC().Truncate(time.Second),
		CompletedSteps:    []string{"Provision server", "Install platform"},
	}

	if err := SaveTo(s, path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if loaded.Domain != s.Domain {
		t.Errorf("Domain: got %q, want %q", loaded.Domain, s.Domain)
	}
	if loaded.ServerIP != s.ServerIP {
		t.Errorf("ServerIP: got %q, want %q", loaded.ServerIP, s.ServerIP)
	}
	if loaded.AdminPassword != s.AdminPassword {
		t.Errorf("AdminPassword: got %q, want %q", loaded.AdminPassword, s.AdminPassword)
	}
	if loaded.SSHKeyPath != s.SSHKeyPath {
		t.Errorf("SSHKeyPath: got %q, want %q", loaded.SSHKeyPath, s.SSHKeyPath)
	}
	if len(loaded.CompletedSteps) != 2 {
		t.Errorf("CompletedSteps: got %d, want 2", len(loaded.CompletedSteps))
	}
}

func TestSave_DefaultPath(t *testing.T) {
	// Override home dir by setting HOME env var
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	s := &State{Domain: "test.example.com", ServerIP: "5.6.7.8"}
	if err := Save(s); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	expectedPath := filepath.Join(dir, ".zen", "install-state.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected state file at %s, does not exist", expectedPath)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Domain != "test.example.com" {
		t.Errorf("Domain: got %q, want %q", loaded.Domain, "test.example.com")
	}
}

func TestMarkStepComplete(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	s := &State{Domain: "example.com"}

	if err := MarkStepComplete(s, "Provision server"); err != nil {
		t.Fatalf("MarkStepComplete failed: %v", err)
	}
	if !IsStepComplete(s, "Provision server") {
		t.Error("Expected step to be complete")
	}
	if IsStepComplete(s, "Install platform") {
		t.Error("Expected unregistered step to not be complete")
	}

	// Double mark — no duplicate
	if err := MarkStepComplete(s, "Provision server"); err != nil {
		t.Fatalf("Second MarkStepComplete failed: %v", err)
	}
	if len(s.CompletedSteps) != 1 {
		t.Errorf("Expected 1 completed step, got %d", len(s.CompletedSteps))
	}
}
```

- [ ] **Step 2: Run failing tests**

```bash
cd cli && go test ./internal/installstate/... -v 2>&1 | tail -20
```
Expected: FAIL — `state_test.go` doesn't compile yet (new file, no import issues; should pass actually since the functions exist — but `TestSave_DefaultPath` will reveal the `Save` default path behaviour).

- [ ] **Step 3: Write failing test — BuildResult persists state**

Add to `cli/internal/install/installer_test.go`:
```go
func TestBuildResult_PersistsStateToDefaultPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &Config{
		MCProvider:    ProviderHetzner,
		Domain:        "example.com",
		SSHHost:       "10.0.0.1",
		AdminPassword: "pre-generated-pass",
	}

	BuildResult(cfg)

	statePath := filepath.Join(dir, ".zen", "install-state.yaml")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Errorf("Expected install-state.yaml at %s to exist after BuildResult", statePath)
	}
}
```

Add required imports at the top of the test file:
```go
import (
	"os"
	"path/filepath"
	"testing"
	// ...existing imports...
)
```

- [ ] **Step 4: Run to verify it fails**

```bash
cd cli && go test ./internal/install/... -run TestBuildResult_PersistsStateToDefaultPath -v
```
Expected: FAIL — state file does not exist (the `SaveTo(state, "")` bug).

- [ ] **Step 5: Fix `BuildResult` — `SaveTo(state, "")` → `Save(state)`**

In `cli/internal/install/installer.go`, find the line (around line 279):
```go
	_ = installstate.SaveTo(&installstate.State{
```

Replace the entire `SaveTo` block:
```go
	// Persist installation state to ~/.zen/install-state.yaml
	state := &installstate.State{
		Domain:            cfg.Domain,
		ServerIP:          ip,
		MissionControlURL: result.MissionControlURL,
		CloudURL:          result.CloudURL,
		AdminUser:         result.AdminUser,
		AdminPassword:     result.AdminPassword,
		Provider:          string(cfg.MCProvider),
		Region:            cfg.Region,
		ServerID:          fmt.Sprintf("%d", cfg.ProvisionedServerID),
		SSHKeyID:          fmt.Sprintf("%d", cfg.HetznerSSHKeyID),
		SSHKeyPath:        cfg.SSHKeyPath,
		InstalledAt:       time.Now().UTC(),
	}
	if cfg.AdminToken != "" {
		state.AdminToken = cfg.AdminToken
	}
	_ = installstate.Save(state)
```

- [ ] **Step 6: Add `AdminToken` and `SSHKeyPath` fields to State (if not already present)**

In `cli/internal/installstate/state.go`, verify `SSHKeyPath` exists (it does). Add `AdminToken` field:
```go
type State struct {
	Domain            string    `yaml:"domain"`
	ServerIP          string    `yaml:"server_ip"`
	MissionControlURL string    `yaml:"mission_control_url"`
	CloudURL          string    `yaml:"cloud_url"`
	AdminUser         string    `yaml:"admin_user"`
	AdminPassword     string    `yaml:"admin_password"`
	AdminToken        string    `yaml:"admin_token,omitempty"`
	ZenithVersion     string    `yaml:"zenith_version,omitempty"`
	SSHKeyPath        string    `yaml:"ssh_key_path"`
	Provider          string    `yaml:"provider"`
	Region            string    `yaml:"region"`
	ServerID          string    `yaml:"server_id"`
	SSHKeyID          string    `yaml:"ssh_key_id"`
	InstalledAt       time.Time `yaml:"installed_at"`
	CompletedSteps    []string  `yaml:"completed_steps,omitempty"`
}
```

- [ ] **Step 7: Add `AdminToken string` field to `install.Config`**

In `cli/internal/install/installer.go`, find the `Config` struct and add after `AdminPassword`:
```go
	AdminToken   string // JWT token obtained after createFirstCluster login
```

- [ ] **Step 8: Fix `provisionHetznerServer` — save SSH key to disk**

In `cli/internal/install/installer.go`, after the line that sets `cfg.GeneratedSSHPrivateKey` (inside `provisionHetznerServer`), add:

```go
	cfg.GeneratedSSHPrivateKey = kp.PrivateKeyPEM

	// Save key to disk so zen upgrade can SSH back in
	if home, err := os.UserHomeDir(); err == nil {
		keyPath := filepath.Join(home, ".zen", "install-key.pem")
		if mkErr := os.MkdirAll(filepath.Dir(keyPath), 0o700); mkErr == nil {
			if writeErr := os.WriteFile(keyPath, kp.PrivateKeyPEM, 0o600); writeErr == nil {
				cfg.SSHKeyPath = keyPath
			}
		}
	}
```

Add required imports at the top of `installer.go` (if not already present):
```go
import (
    "os"
    "path/filepath"
    // ...existing imports...
)
```

- [ ] **Step 9: Add `dialSSH` helper to eliminate duplicated SSH setup**

In `cli/internal/install/installer.go`, add before `provisionHetznerServer`:

```go
// dialSSH creates an SSH client from the current install config.
func dialSSH(cfg *Config) (*sshclient.Client, error) {
	user := cfg.SSHUser
	if user == "" {
		user = "root"
	}
	sshCfg := sshclient.Config{
		Host:    cfg.SSHHost,
		Port:    22,
		User:    user,
		Timeout: 30 * time.Second,
	}
	if len(cfg.GeneratedSSHPrivateKey) > 0 {
		sshCfg.PrivateKey = cfg.GeneratedSSHPrivateKey
	}
	return sshclient.DialWithRetry(sshCfg, 10, 15*time.Second)
}
```

Refactor `verifyExistingServer` and `installPlatform` to call `dialSSH(cfg)` instead of building the `sshclient.Config` inline.

- [ ] **Step 10: Run tests to verify all pass**

```bash
cd cli && go test ./internal/install/... ./internal/installstate/... -v 2>&1 | tail -30
```
Expected: all PASS.

- [ ] **Step 11: Commit**

```bash
git add cli/internal/install/installer.go cli/internal/installstate/state.go cli/internal/installstate/state_test.go cli/internal/install/installer_test.go
git commit -m "fix(cli): fix state persistence bug, save SSH key to disk, add dialSSH helper"
```

---

### Task 2: Admin Password Lifecycle

**Files:**
- Modify: `cli/internal/install/installer.go`
- Modify: `cli/cmd/install/install.go`
- Modify: `cli/internal/install/installer_test.go`

The admin password is currently generated inside `BuildResult` (runs **after** all install steps complete). The Helm chart install needs it **during** the "Install Zenith chart" step. Fix: generate before the step loop, store in `cfg.AdminPassword`, and have `BuildResult` use it if already set.

- [ ] **Step 1: Write failing test — password present before steps run**

Add to `cli/internal/install/installer_test.go`:
```go
func TestGetInstallSteps_AdminPasswordSetBeforeExecution(t *testing.T) {
	cfg := &Config{
		MCProvider:    ProviderHetzner,
		HetznerToken:  "test-token-1234567890",
		Domain:        "example.com",
		DNSProvider:   DNSManual,
		DryRun:        true,
		AdminPassword: "pre-set-password-16c",
	}

	steps := GetInstallSteps(cfg)

	// Execute all steps — the "Install Zenith chart" step must see a non-empty AdminPassword
	for _, step := range steps {
		if err := step.Action(cfg); err != nil {
			t.Errorf("Step %q failed: %v", step.Name, err)
		}
	}

	// Password should be preserved (not regenerated)
	if cfg.AdminPassword != "pre-set-password-16c" {
		t.Errorf("Expected AdminPassword to be preserved, got %q", cfg.AdminPassword)
	}
}

func TestGeneratePassword_Length(t *testing.T) {
	for _, n := range []int{8, 16, 24, 32} {
		p := GeneratePassword(n)
		if len(p) != n {
			t.Errorf("GeneratePassword(%d) returned length %d", n, len(p))
		}
	}
}

func TestGeneratePassword_Uniqueness(t *testing.T) {
	p1 := GeneratePassword(16)
	p2 := GeneratePassword(16)
	if p1 == p2 {
		t.Error("Two GeneratePassword calls returned identical passwords")
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
cd cli && go test ./internal/install/... -run "TestGeneratePassword" -v
```
Expected: FAIL — `GeneratePassword` not exported.

- [ ] **Step 3: Export `generatePassword` → `GeneratePassword`**

In `cli/internal/install/installer.go`, rename:
```go
func GeneratePassword(length int) string {
```

Update the single call inside `BuildResult`:
```go
AdminPassword: GeneratePassword(16),
```

And update `BuildResult` to use pre-set password if available:
```go
func BuildResult(cfg *Config) *InstallResult {
	ip := cfg.SSHHost
	if ip == "" {
		ip = "203.0.113.42"
	}

	adminPassword := cfg.AdminPassword
	if adminPassword == "" {
		adminPassword = GeneratePassword(16)
	}

	result := &InstallResult{
		ServerIP:          ip,
		Domain:            cfg.Domain,
		MissionControlURL: fmt.Sprintf("https://mission.%s", cfg.Domain),
		CloudURL:          fmt.Sprintf("https://cloud.%s", cfg.Domain),
		AdminUser:         "admin",
		AdminPassword:     adminPassword,
	}
	// ...rest unchanged
```

- [ ] **Step 4: Generate password before step loop in `install.go`**

In `cli/cmd/install/install.go`, inside `runSteps`, before the `steps := install.GetInstallSteps(cfg)` line:
```go
	// Generate admin password before steps run so installZenithChart can pass it to Helm.
	if cfg.AdminPassword == "" {
		cfg.AdminPassword = install.GeneratePassword(16)
	}

	steps := install.GetInstallSteps(cfg)
```

- [ ] **Step 5: Run tests to verify pass**

```bash
cd cli && go test ./internal/install/... -run "TestGeneratePassword|TestBuildResult" -v
```
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add cli/internal/install/installer.go cli/cmd/install/install.go cli/internal/install/installer_test.go
git commit -m "fix(cli): generate admin password before install steps, export GeneratePassword"
```

---

### Task 3: Wire Helm Chart Install into `installPlatform`

**Files:**
- Modify: `cli/internal/install/installer.go`
- Modify: `cli/internal/install/installer_test.go`

`installPlatform` installs k3s but never deploys the Zenith Helm chart. Add a new `installZenithChart` function that: installs helm on the remote server, writes a temp values file (via base64 to handle special chars in passwords), runs `helm upgrade --install zenith oci://...`, then cleans up. Add a "Install Zenith chart" step to `GetInstallSteps` between "Install platform" and "Configure DNS".

- [ ] **Step 1: Write failing test — step count increases**

Update `TestGetInstallSteps` in `cli/internal/install/installer_test.go`:
```go
func TestGetInstallSteps(t *testing.T) {
	cfg := &Config{
		MCProvider:    ProviderHetzner,
		HetznerToken:  "test-token-1234567890",
		ServerType:    "cx22",
		Region:        "fsn1",
		Domain:        "example.com",
		DNSProvider:   DNSManual,
		DryRun:        true,
		AdminPassword: "test-password-here1",
	}

	steps := GetInstallSteps(cfg)
	if len(steps) != 6 {
		t.Errorf("Expected 6 install steps (no cluster), got %d", len(steps))
	}

	for i, step := range steps {
		if err := step.Action(cfg); err != nil {
			t.Errorf("Step %d (%s) failed: %v", i, step.Name, err)
		}
	}
}

func TestGetInstallSteps_WithCluster(t *testing.T) {
	cfg := &Config{
		MCProvider:        ProviderHetzner,
		HetznerToken:      "test-token-1234567890",
		ServerType:        "cx22",
		Region:            "fsn1",
		Domain:            "example.com",
		DNSProvider:       DNSManual,
		WithCluster:       true,
		ClusterProvider:   ProviderHetzner,
		ClusterServerType: "cx22",
		ClusterRegion:     "fsn1",
		DryRun:            true,
		AdminPassword:     "test-password-here1",
	}

	steps := GetInstallSteps(cfg)
	if len(steps) != 7 {
		t.Errorf("Expected 7 install steps (with cluster), got %d", len(steps))
	}

	lastStep := steps[len(steps)-1]
	if lastStep.Name != "Create first cluster" {
		t.Errorf("Expected last step 'Create first cluster', got '%s'", lastStep.Name)
	}
}

func TestGetInstallSteps_StepNames(t *testing.T) {
	cfg := &Config{
		MCProvider:    ProviderHetzner,
		HetznerToken:  "test-token-1234567890",
		ServerType:    "cx22",
		Region:        "fsn1",
		Domain:        "example.com",
		DNSProvider:   DNSCloudflare,
		DryRun:        true,
		AdminPassword: "test-password-here1",
	}

	steps := GetInstallSteps(cfg)

	expectedNames := []string{
		"Provision server",
		"Install platform",
		"Install Zenith chart",
		"Configure DNS",
		"Issue SSL certificates",
		"Wait for Mission Control",
	}

	for i, expected := range expectedNames {
		if i >= len(steps) {
			t.Fatalf("Expected at least %d steps", i+1)
		}
		if steps[i].Name != expected {
			t.Errorf("Step %d: expected name %q, got %q", i, expected, steps[i].Name)
		}
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
cd cli && go test ./internal/install/... -run "TestGetInstallSteps" -v 2>&1 | tail -20
```
Expected: FAIL — step count is 5, expected 6.

- [ ] **Step 3: Add `AdminEmail` to `Config`**

In `cli/internal/install/installer.go`, in the `Config` struct, after `AdminPassword`:
```go
	AdminEmail   string // defaults to "admin@<domain>" if empty
```

- [ ] **Step 4: Add `installZenithChart` function**

In `cli/internal/install/installer.go`, add after `installPlatform`:

```go
// installZenithChart installs helm on the remote server, writes a temp values
// file via base64, runs helm upgrade --install for the Zenith chart, then cleans up.
func installZenithChart(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}

	sshCli, err := dialSSH(cfg)
	if err != nil {
		return fmt.Errorf("ssh connect: %w", err)
	}
	defer sshCli.Close()

	// Install helm if not already present
	out, err := sshCli.Run("which helm >/dev/null 2>&1 || (curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash 2>&1)")
	if err != nil {
		return fmt.Errorf("install helm: %w\nOutput: %s", err, out)
	}

	adminEmail := cfg.AdminEmail
	if adminEmail == "" {
		adminEmail = "admin@" + cfg.Domain
	}
	hetznerToken := cfg.HetznerToken
	if hetznerToken == "" {
		hetznerToken = "none"
	}

	valuesYAML := fmt.Sprintf("global:\n  domain: %s\n  hetznerToken: %s\nsecrets:\n  adminEmail: %s\n  adminPassword: %s\n",
		cfg.Domain, hetznerToken, adminEmail, cfg.AdminPassword)

	// Write via base64 to handle special characters in passwords
	encoded := base64.StdEncoding.EncodeToString([]byte(valuesYAML))
	writeCmd := fmt.Sprintf("echo '%s' | base64 -d > /tmp/zenith-install-values.yaml", encoded)
	if _, err := sshCli.Run(writeCmd); err != nil {
		return fmt.Errorf("write values file: %w", err)
	}

	helmCmd := "KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm upgrade --install zenith " +
		"oci://ghcr.io/dotechhq/zenith/charts/zenith " +
		"--namespace zenith-system --create-namespace " +
		"-f /tmp/zenith-install-values.yaml " +
		"--wait --timeout 10m 2>&1"
	if out, err := sshCli.Run(helmCmd); err != nil {
		sshCli.Run("rm -f /tmp/zenith-install-values.yaml") //nolint:errcheck
		return fmt.Errorf("helm install: %w\nOutput: %s", err, out)
	}

	sshCli.Run("rm -f /tmp/zenith-install-values.yaml") //nolint:errcheck
	return nil
}
```

Add `"encoding/base64"` to the import block.

- [ ] **Step 5: Add "Install Zenith chart" step to `GetInstallSteps`**

In `cli/internal/install/installer.go`, in `GetInstallSteps`, insert this step **between** "Install platform" and "Configure DNS":

```go
		{
			Name:        "Install Zenith chart",
			Description: fmt.Sprintf("Installing Zenith via Helm on %s...", cfg.SSHHost),
			Duration:    2 * time.Minute,
			Action: func(cfg *Config) error {
				return installZenithChart(cfg)
			},
		},
```

- [ ] **Step 6: Run tests to verify pass**

```bash
cd cli && go test ./internal/install/... -run "TestGetInstallSteps" -v
```
Expected: all PASS.

- [ ] **Step 7: Run full install test suite**

```bash
cd cli && go test ./internal/install/... -v 2>&1 | tail -20
```
Expected: all PASS (all dry-run-based tests skip the SSH call).

- [ ] **Step 8: Commit**

```bash
git add cli/internal/install/installer.go cli/internal/install/installer_test.go
git commit -m "feat(cli): wire Helm chart install into zen install via SSH, add AdminEmail to Config"
```

---

### Task 4: Fix `zen upgrade` Release Name + Chart Reference

**Files:**
- Modify: `cli/cmd/upgrade/upgrade.go`

The upgrade command uses `helm upgrade zenith-platform zenith/zenith-platform` — both the release name and chart reference are wrong. The installer uses release name `zenith` and chart `oci://ghcr.io/dotechhq/zenith/charts/zenith`. Fix these plus rollback and dry-run display.

- [ ] **Step 1: Write failing test**

Create `cli/cmd/upgrade/upgrade_test.go`:
```go
package upgrade

import (
	"strings"
	"testing"
)

func TestHelmUpgradeCommand_ReleaseName(t *testing.T) {
	// helmUpgrade builds a command string — verify it uses the correct release name and chart
	// We test the command by running it with --dry-run=client which prints but doesn't execute
	cmd := buildHelmUpgradeCmd("1.2.3")
	if !strings.Contains(cmd, "zenith ") {
		t.Errorf("Expected release name 'zenith' in command, got: %s", cmd)
	}
	if strings.Contains(cmd, "zenith-platform") {
		t.Errorf("Expected 'zenith-platform' to be gone, got: %s", cmd)
	}
	if !strings.Contains(cmd, "oci://ghcr.io/dotechhq/zenith/charts/zenith") {
		t.Errorf("Expected OCI chart ref in command, got: %s", cmd)
	}
}

func TestHelmRollbackCommand_ReleaseName(t *testing.T) {
	cmd := buildHelmRollbackCmd()
	if !strings.Contains(cmd, "zenith ") && !strings.HasSuffix(cmd, "zenith") {
		t.Errorf("Expected release name 'zenith' in rollback command, got: %s", cmd)
	}
	if strings.Contains(cmd, "zenith-platform") {
		t.Errorf("Expected 'zenith-platform' to be gone from rollback, got: %s", cmd)
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
cd cli && go test ./cmd/upgrade/... -v 2>&1 | tail -20
```
Expected: FAIL — `buildHelmUpgradeCmd` and `buildHelmRollbackCmd` don't exist yet.

- [ ] **Step 3: Extract `buildHelmUpgradeCmd` and `buildHelmRollbackCmd` helpers + fix values**

In `cli/cmd/upgrade/upgrade.go`, replace the `helmUpgrade` and `helmRollback` functions:

```go
// buildHelmUpgradeCmd returns the helm upgrade command string for the given version.
func buildHelmUpgradeCmd(version string) string {
	cmd := "KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm upgrade zenith " +
		"oci://ghcr.io/dotechhq/zenith/charts/zenith " +
		"-n zenith-system --wait --timeout=10m"
	if version != "" {
		cmd += " --version " + version
	}
	return cmd + " 2>&1"
}

// buildHelmRollbackCmd returns the helm rollback command string.
func buildHelmRollbackCmd() string {
	return "KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm rollback zenith -n zenith-system --wait --timeout=5m 2>&1"
}

// helmUpgrade runs helm upgrade for the zenith chart.
func helmUpgrade(cli *sshclient.Client, version string) error {
	_, err := cli.Run(buildHelmUpgradeCmd(version))
	return err
}

// helmRollback rolls back the zenith Helm release.
func helmRollback(cli *sshclient.Client) error {
	_, err := cli.Run(buildHelmRollbackCmd())
	return err
}
```

Also fix `waitForRollout` to use correct release name context (namespace stays `zenith-system`):
```go
func waitForRollout(cli *sshclient.Client) error {
	cmd := "KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl rollout status deployment -n zenith-system --timeout=10m 2>&1 && " +
		"KUBECONFIG=/etc/rancher/k3s/k3s.yaml kubectl rollout status statefulset -n zenith-system --timeout=10m 2>&1"
	_, err := cli.Run(cmd)
	return err
}
```

Also fix the description text in `Cmd.Long` (line starting "2. Run helm upgrade for zenith-platform") and in `printDryRun` (replace `zenith-platform@%s` with `zenith@%s`).

- [ ] **Step 4: Run tests to verify pass**

```bash
cd cli && go test ./cmd/upgrade/... -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cli/cmd/upgrade/upgrade.go cli/cmd/upgrade/upgrade_test.go
git commit -m "fix(cli): correct zen upgrade helm release name (zenith-platform→zenith) and chart ref to OCI"
```

---

### Task 5: Version Tracking + Compatibility Check

**Files:**
- Create: `cli/internal/semver/semver.go`
- Create: `cli/internal/semver/semver_test.go`
- Modify: `cli/internal/install/installer.go` (save version in BuildResult)
- Modify: `cli/cmd/upgrade/upgrade.go` (read version, compare, save after upgrade)

The spec requires blocking upgrades that skip more than one minor version (e.g. v1.0 → v1.3 must go via v1.1, v1.2). Implement a semver comparison package and wire it into upgrade.

- [ ] **Step 1: Write semver tests first**

Create `cli/internal/semver/semver_test.go`:
```go
package semver

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input         string
		wantMajor     int
		wantMinor     int
		wantPatch     int
		wantErr       bool
	}{
		{"1.2.3", 1, 2, 3, false},
		{"v1.2.3", 1, 2, 3, false},
		{"zenith-1.2.3", 1, 2, 3, false},
		{"0.9.0", 0, 9, 0, false},
		{"latest", 0, 0, 0, true},
		{"", 0, 0, 0, true},
		{"1.2", 0, 0, 0, true},
		{"abc", 0, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, patch, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if major != tt.wantMajor || minor != tt.wantMinor || patch != tt.wantPatch {
				t.Errorf("ParseVersion(%q) = %d.%d.%d, want %d.%d.%d",
					tt.input, major, minor, patch, tt.wantMajor, tt.wantMinor, tt.wantPatch)
			}
		})
	}
}

func TestIsSafeUpgrade(t *testing.T) {
	tests := []struct {
		current  string
		target   string
		wantSafe bool
	}{
		{"1.0.0", "1.1.0", true},  // one minor version up: OK
		{"1.0.0", "1.0.5", true},  // patch only: OK
		{"1.0.0", "1.2.0", false}, // two minor versions: blocked
		{"1.0.0", "1.5.0", false}, // five minor versions: blocked
		{"1.1.0", "2.0.0", true},  // major bump, same minor rule applies per spec (major counts as 1 step)
		{"0.9.0", "1.0.0", true},  // cross-major, one step: OK
		{"0.9.0", "1.1.0", false}, // cross-major + minor: blocked
		{"1.2.3", "1.3.0", true},  // minor +1: OK
		{"1.2.3", "1.2.10", true}, // patch only: OK
	}
	for _, tt := range tests {
		t.Run(tt.current+"→"+tt.target, func(t *testing.T) {
			err := IsSafeUpgrade(tt.current, tt.target)
			isSafe := (err == nil)
			if isSafe != tt.wantSafe {
				t.Errorf("IsSafeUpgrade(%q, %q) safe=%v, want %v (err: %v)",
					tt.current, tt.target, isSafe, tt.wantSafe, err)
			}
		})
	}
}
```

- [ ] **Step 2: Run to verify FAIL (package doesn't exist)**

```bash
cd cli && go test ./internal/semver/... 2>&1 | head -5
```
Expected: error "no Go files in .../semver".

- [ ] **Step 3: Implement `semver.go`**

Create `cli/internal/semver/semver.go`:
```go
package semver

import (
	"fmt"
	"regexp"
	"strconv"
)

var versionRe = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// ParseVersion extracts major, minor, patch from strings like "1.2.3", "v1.2.3", "zenith-1.2.3".
func ParseVersion(v string) (major, minor, patch int, err error) {
	m := versionRe.FindStringSubmatch(v)
	if m == nil {
		return 0, 0, 0, fmt.Errorf("cannot parse version from %q", v)
	}
	major, _ = strconv.Atoi(m[1])
	minor, _ = strconv.Atoi(m[2])
	patch, _ = strconv.Atoi(m[3])
	return major, minor, patch, nil
}

// IsSafeUpgrade returns nil if current→target is a safe upgrade (≤1 minor version jump).
// "Minor version jump" is computed as (targetMajor*1000+targetMinor) - (currentMajor*1000+currentMinor).
func IsSafeUpgrade(current, target string) error {
	curMaj, curMin, _, err := ParseVersion(current)
	if err != nil {
		return fmt.Errorf("current version: %w", err)
	}
	tgtMaj, tgtMin, _, err := ParseVersion(target)
	if err != nil {
		return fmt.Errorf("target version: %w", err)
	}

	curStep := curMaj*1000 + curMin
	tgtStep := tgtMaj*1000 + tgtMin

	if tgtStep-curStep > 1 {
		return fmt.Errorf(
			"cannot upgrade from v%d.%d to v%d.%d: must upgrade one minor version at a time (next: v%d.%d)",
			curMaj, curMin, tgtMaj, tgtMin, curMaj, curMin+1,
		)
	}
	return nil
}
```

- [ ] **Step 4: Run semver tests to verify pass**

```bash
cd cli && go test ./internal/semver/... -v
```
Expected: all PASS.

- [ ] **Step 5: Add version reading helper to upgrade.go**

In `cli/cmd/upgrade/upgrade.go`, add:
```go
// currentInstalledVersion reads the chart version of the zenith release via helm list.
// Returns "" if the version cannot be determined (non-fatal).
func currentInstalledVersion(cli *sshclient.Client) string {
	out, err := cli.Run(
		"KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm list -n zenith-system --filter '^zenith$' -o json 2>/dev/null",
	)
	if err != nil || out == "" {
		return ""
	}
	// Parse JSON: [{"name":"zenith","chart":"zenith-1.2.3",...}]
	type release struct {
		Chart string `json:"chart"`
	}
	var releases []release
	if err := json.Unmarshal([]byte(out), &releases); err != nil || len(releases) == 0 {
		return ""
	}
	// Chart field is like "zenith-1.2.3" — strip prefix
	chart := releases[0].Chart
	if idx := strings.LastIndex(chart, "-"); idx >= 0 {
		return chart[idx+1:]
	}
	return chart
}
```

Add `"encoding/json"` to the imports of `upgrade.go`.

- [ ] **Step 6: Add version compatibility check to `buildSteps`**

In `buildSteps`, after reading the SSH client, add a pre-check for version compatibility when a specific target version is requested. Add a new first step (before backup):

```go
	if version != "" {
		steps = append(steps, stepFunc{
			name: "Version compatibility",
			desc: "Checking semver compatibility...",
			fn: func(cli *sshclient.Client) error {
				current := currentInstalledVersion(cli)
				if current == "" {
					return nil // cannot determine — allow upgrade
				}
				return semverPkg.IsSafeUpgrade(current, version)
			},
		})
	}
```

Add import alias at top of `upgrade.go`:
```go
import (
    // ...existing...
    semverPkg "github.com/dotechhq/zenith/cli/internal/semver"
)
```

- [ ] **Step 7: Save version to state after successful upgrade**

In `runUpgrade`, after the step loop completes successfully:
```go
	// Save new installed version to state
	if version != "" {
		if state, loadErr := installstate.Load(); loadErr == nil {
			state.ZenithVersion = version
			installstate.Save(state)
		}
	} else {
		// Read the version that was actually installed
		if state, loadErr := installstate.Load(); loadErr == nil {
			state.ZenithVersion = currentInstalledVersion(sshCli)
			installstate.Save(state)
		}
	}
```

- [ ] **Step 8: Run all tests**

```bash
cd cli && go test ./... 2>&1 | tail -20
```
Expected: all PASS.

- [ ] **Step 9: Commit**

```bash
git add cli/internal/semver/ cli/cmd/upgrade/upgrade.go cli/internal/install/installer.go
git commit -m "feat(cli): add semver package, version compatibility check in zen upgrade, version tracking in state"
```

---

### Task 6: Upgrade Pre-flight Checks

**Files:**
- Modify: `cli/cmd/upgrade/upgrade.go`
- Modify: `cli/cmd/upgrade/upgrade_test.go`

Before upgrading, show the user their current version and the target, then verify disk space (≥5GB free). This gives confidence that the upgrade is safe and prevents mid-upgrade disk failures.

- [ ] **Step 1: Write tests for pre-flight helpers**

Add to `cli/cmd/upgrade/upgrade_test.go`:
```go
func TestParseDiskSpaceFreeGB(t *testing.T) {
	tests := []struct {
		input   string
		wantGB  float64
		wantOK  bool
	}{
		{"10G", 10.0, true},
		{"5.5G", 5.5, true},
		{"512M", 0.5, true},
		{"2048M", 2.0, true},
		{"100K", 0.0001, true},
		{"", 0, false},
		{"garbage", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gb, ok := parseDiskSpaceFreeGB(tt.input)
			if ok != tt.wantOK {
				t.Errorf("parseDiskSpaceFreeGB(%q) ok=%v, want %v", tt.input, ok, tt.wantOK)
				return
			}
			if ok && (gb < tt.wantGB*0.99 || gb > tt.wantGB*1.01) {
				t.Errorf("parseDiskSpaceFreeGB(%q) = %.3f, want %.3f", tt.input, gb, tt.wantGB)
			}
		})
	}
}

func TestBuildSteps_IncludesPreflightFirst(t *testing.T) {
	// buildSteps is not exported but we verify through the step list length
	// A non-dry-run with no --skip-backup produces: preflight + backup + upgrade + rollout + health = 5
	// (we can't test this without SSH, but we verify the total step count via the exported function)
	// This test verifies that adding --skip-backup removes 1 step, not the preflight
	stepsWithBackup := len(buildStepsForTest(false))
	stepsNoBackup := len(buildStepsForTest(true))
	if stepsNoBackup != stepsWithBackup-1 {
		t.Errorf("--skip-backup should remove exactly 1 step: got %d vs %d", stepsNoBackup, stepsWithBackup)
	}
}
```

Add a test helper (unexported, test-only) at the bottom of `upgrade_test.go`:
```go
func buildStepsForTest(skipBackup bool) []stepFunc {
	return buildSteps(nil, &installstate.State{Domain: "test.example.com"}, "", skipBackup)
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
cd cli && go test ./cmd/upgrade/... -run "TestParseDiskSpace|TestBuildSteps" -v 2>&1 | tail -10
```
Expected: FAIL — `parseDiskSpaceFreeGB` not defined.

- [ ] **Step 3: Implement `parseDiskSpaceFreeGB`**

In `cli/cmd/upgrade/upgrade.go`:
```go
// parseDiskSpaceFreeGB parses a string like "10G", "512M", "2048K" and returns GB as float64.
func parseDiskSpaceFreeGB(s string) (float64, bool) {
	if len(s) < 2 {
		return 0, false
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, false
	}
	switch unit {
	case 'G', 'g':
		return val, true
	case 'M', 'm':
		return val / 1024, true
	case 'K', 'k':
		return val / (1024 * 1024), true
	case 'T', 't':
		return val * 1024, true
	}
	return 0, false
}
```

Add `"strconv"` to imports.

- [ ] **Step 4: Add pre-flight step to `buildSteps`**

In `buildSteps`, insert as the **first** step (before the version-compat check from Task 5):

```go
	steps = append(steps, stepFunc{
		name: "Pre-flight check",
		desc: "Checking disk space and cluster reachability...",
		fn: func(cli *sshclient.Client) error {
			// Disk space: require ≥5GB free on /
			freeStr, err := cli.Run("df -h / | awk 'NR==2{print $4}'")
			if err != nil {
				return fmt.Errorf("disk check failed: %w", err)
			}
			freeStr = strings.TrimSpace(freeStr)
			if gb, ok := parseDiskSpaceFreeGB(freeStr); ok && gb < 5.0 {
				return fmt.Errorf("insufficient disk space: %.1fGB free (need ≥5GB)", gb)
			}
			return nil
		},
	})
```

- [ ] **Step 5: Display current→target version header before steps run**

In `runUpgrade`, after connecting SSH, add before the steps loop:
```go
	// Show current → target version
	current := currentInstalledVersion(sshCli)
	tgt := flagVersion
	if tgt == "" {
		tgt = "latest"
	}
	if current != "" {
		fmt.Println(muted.Render(fmt.Sprintf("  %s → %s", current, tgt)))
	} else {
		fmt.Println(muted.Render(fmt.Sprintf("  current version unknown → %s", tgt)))
	}
	fmt.Println()
```

- [ ] **Step 6: Run tests to verify pass**

```bash
cd cli && go test ./cmd/upgrade/... -v 2>&1 | tail -20
```
Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add cli/cmd/upgrade/upgrade.go cli/cmd/upgrade/upgrade_test.go
git commit -m "feat(cli): add upgrade pre-flight (disk space check, version display)"
```

---

### Task 7: Upgrade `--dry-run` with `helm diff`

**Files:**
- Modify: `cli/cmd/upgrade/upgrade.go`

The spec says `--dry-run` should run `helm diff upgrade` to show what would change, not just print a static table. This uses the `helm-diff` plugin. The function installs the plugin on the remote server if not present, then runs `helm diff upgrade`.

- [ ] **Step 1: Write test for the dry-run diff command builder**

Add to `cli/cmd/upgrade/upgrade_test.go`:
```go
func TestBuildHelmDiffCmd(t *testing.T) {
	cmd := buildHelmDiffCmd("1.2.3")
	if !strings.Contains(cmd, "helm diff upgrade") {
		t.Errorf("Expected 'helm diff upgrade' in cmd, got: %s", cmd)
	}
	if !strings.Contains(cmd, "zenith") {
		t.Errorf("Expected release name 'zenith' in cmd, got: %s", cmd)
	}
	if !strings.Contains(cmd, "oci://ghcr.io/dotechhq/zenith/charts/zenith") {
		t.Errorf("Expected OCI chart ref in diff cmd, got: %s", cmd)
	}
	if !strings.Contains(cmd, "1.2.3") {
		t.Errorf("Expected version 1.2.3 in diff cmd, got: %s", cmd)
	}
}

func TestBuildHelmDiffCmd_NoVersion(t *testing.T) {
	cmd := buildHelmDiffCmd("")
	if strings.Contains(cmd, "--version") {
		t.Errorf("Expected no --version flag when version is empty, got: %s", cmd)
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
cd cli && go test ./cmd/upgrade/... -run "TestBuildHelmDiffCmd" -v
```
Expected: FAIL — `buildHelmDiffCmd` not defined.

- [ ] **Step 3: Implement `buildHelmDiffCmd` and `ensureHelmDiff`**

In `cli/cmd/upgrade/upgrade.go`:
```go
// buildHelmDiffCmd returns the helm diff upgrade command for the given version.
func buildHelmDiffCmd(version string) string {
	cmd := "KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm diff upgrade zenith " +
		"oci://ghcr.io/dotechhq/zenith/charts/zenith " +
		"-n zenith-system"
	if version != "" {
		cmd += " --version " + version
	}
	return cmd + " 2>&1"
}

// ensureHelmDiff installs the helm-diff plugin on the remote server if not present.
func ensureHelmDiff(cli *sshclient.Client) error {
	out, _ := cli.Run("KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm plugin list 2>/dev/null | grep -c diff")
	if strings.TrimSpace(out) == "1" {
		return nil // already installed
	}
	installOut, err := cli.Run("helm plugin install https://github.com/databus23/helm-diff 2>&1")
	if err != nil {
		return fmt.Errorf("install helm-diff plugin: %w\n%s", err, installOut)
	}
	return nil
}
```

- [ ] **Step 4: Replace `printDryRun` with SSH-based helm diff**

Replace the `printDryRun` function entirely:
```go
// runDryRun connects via SSH and runs helm diff upgrade to show what would change.
// Falls back to printing a static plan if SSH connection fails.
func runDryRun(state *installstate.State, version string, skipBackup bool) {
	muted := lipgloss.NewStyle().Foreground(tui.ColorMuted).PaddingLeft(2)
	info := lipgloss.NewStyle().Foreground(tui.ColorText).PaddingLeft(2)
	bold := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).PaddingLeft(2)
	errStyle := lipgloss.NewStyle().Foreground(tui.ColorError).PaddingLeft(2)

	fmt.Println(bold.Render("Dry-run plan:"))
	fmt.Println()

	versionStr := "latest"
	if version != "" {
		versionStr = version
	}
	if !skipBackup {
		fmt.Println(info.Render("  1. Pre-upgrade backup — CNPG immediate annotation"))
	}
	fmt.Println(info.Render(fmt.Sprintf("  Helm upgrade: zenith@%s in zenith-system", versionStr)))
	fmt.Println()

	if state.ServerIP == "" {
		fmt.Println(muted.Render("No server IP in state — skipping helm diff."))
		return
	}

	fmt.Println(muted.Render("Connecting to server for helm diff..."))
	sshCfg := sshclient.Config{
		Host:    state.ServerIP,
		User:    "root",
		Timeout: 15 * time.Second,
	}
	if state.SSHKeyPath != "" {
		if keyData, err := os.ReadFile(state.SSHKeyPath); err == nil {
			sshCfg.PrivateKey = keyData
		}
	}
	cli, err := sshclient.DialWithRetry(sshCfg, 3, 5*time.Second)
	if err != nil {
		fmt.Println(errStyle.Render("  Could not connect: " + err.Error()))
		fmt.Println(muted.Render("  Showing static plan above only."))
		return
	}
	defer cli.Close()

	if err := ensureHelmDiff(cli); err != nil {
		fmt.Println(errStyle.Render("  helm-diff not available: " + err.Error()))
		return
	}

	fmt.Println(bold.Render("helm diff output:"))
	fmt.Println()
	out, _ := cli.Run(buildHelmDiffCmd(version))
	fmt.Println(out)
}
```

Update `runUpgrade` to call `runDryRun(state, flagVersion, flagSkipBackup)` instead of `printDryRun(state, flagVersion, flagSkipBackup)`.

Remove the old `printDryRun` function.

- [ ] **Step 5: Run tests to verify pass**

```bash
cd cli && go test ./cmd/upgrade/... -v 2>&1 | tail -15
```
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add cli/cmd/upgrade/upgrade.go cli/cmd/upgrade/upgrade_test.go
git commit -m "feat(cli): zen upgrade --dry-run uses helm diff upgrade for real diff output"
```

---

### Task 8: `createFirstCluster()` — MC Login + Token Persist

**Files:**
- Modify: `cli/internal/api/client.go`
- Modify: `cli/internal/api/client_test.go`
- Modify: `cli/internal/install/installer.go`

The `--with-cluster` flag triggers `createFirstCluster()` which is currently a placeholder. In CE standalone mode, the "cluster" is the k3s node the MC runs on — it's auto-configured by the Helm chart. What `createFirstCluster` *should* do is log in to Mission Control with the admin credentials, obtain a JWT, and save it to state — so `zen status` / `zen deploy` work immediately after install without requiring a separate `zen login`.

Auth endpoint: `POST /api/v1/auth/login`
Request body: `{"email": "...", "password": "..."}`
Response: `{"access_token": "...", "refresh_token": "...", "token_type": "bearer", "expires_in": ...}`

- [ ] **Step 1: Write failing test for `api.Client.Login`**

Add to `cli/internal/api/client_test.go`:
```go
func TestClient_Login_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/auth/login" {
			http.Error(w, "unexpected request", 404)
			return
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["email"] == "" || body["password"] == "" {
			http.Error(w, "missing fields", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "jwt.token.here",
			"refresh_token": "refresh.token.here",
			"token_type":    "bearer",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	token, err := c.Login("admin@example.com", "secret")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if token != "jwt.token.here" {
		t.Errorf("Expected access_token 'jwt.token.here', got %q", token)
	}
}

func TestClient_Login_InvalidCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"invalid credentials"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.Login("bad@example.com", "wrong")
	if err == nil {
		t.Error("Expected error for invalid credentials, got nil")
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
cd cli && go test ./internal/api/... -run "TestClient_Login" -v
```
Expected: FAIL — `Login` method not defined.

- [ ] **Step 3: Add `Login` to `cli/internal/api/client.go`**

```go
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Login authenticates with email+password and returns the access token.
func (c *Client) Login(email, password string) (string, error) {
	var resp loginTokenResponse
	if err := c.do("POST", "/api/v1/auth/login", loginRequest{email, password}, &resp); err != nil {
		return "", err
	}
	return resp.AccessToken, nil
}
```

- [ ] **Step 4: Run to verify PASS**

```bash
cd cli && go test ./internal/api/... -run "TestClient_Login" -v
```
Expected: PASS.

- [ ] **Step 5: Implement `createFirstCluster`**

In `cli/internal/install/installer.go`, replace the placeholder:

```go
// createFirstCluster logs in to Mission Control with admin credentials and saves
// the JWT token to cfg for state persistence. This enables zen status / zen deploy
// to work immediately after install without requiring a separate zen login.
func createFirstCluster(cfg *Config) error {
	if cfg.DryRun {
		return nil
	}
	if cfg.AdminPassword == "" {
		return fmt.Errorf("admin password not set — run zen install first")
	}

	mcURL := fmt.Sprintf("https://mission.%s", cfg.Domain)
	apiClient := cliapi.NewClient(mcURL, "")

	adminEmail := cfg.AdminEmail
	if adminEmail == "" {
		adminEmail = "admin@" + cfg.Domain
	}

	token, err := apiClient.Login(adminEmail, cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("login to Mission Control at %s: %w", mcURL, err)
	}

	cfg.AdminToken = token
	return nil
}
```

Add import:
```go
import (
    cliapi "github.com/dotechhq/zenith/cli/internal/api"
    // ...existing imports...
)
```

- [ ] **Step 6: Write test for `createFirstCluster` dry-run**

Add to `cli/internal/install/installer_test.go`:
```go
func TestCreateFirstCluster_DryRun(t *testing.T) {
	cfg := &Config{
		DryRun:        true,
		Domain:        "example.com",
		AdminPassword: "test-password",
	}

	steps := GetInstallSteps(cfg)

	// Find "Create first cluster" step in a WithCluster config
	cfgWithCluster := *cfg
	cfgWithCluster.WithCluster = true
	stepsWithCluster := GetInstallSteps(&cfgWithCluster)

	var clusterStep *Step
	for i := range stepsWithCluster {
		if stepsWithCluster[i].Name == "Create first cluster" {
			clusterStep = &stepsWithCluster[i]
			break
		}
	}
	if clusterStep == nil {
		t.Fatal("Expected 'Create first cluster' step with WithCluster=true")
	}

	// In dry-run mode, the step must not make real API calls and must return nil
	if err := clusterStep.Action(&cfgWithCluster); err != nil {
		t.Errorf("createFirstCluster dry-run failed: %v", err)
	}
	_ = steps
}
```

- [ ] **Step 7: Run all tests**

```bash
cd cli && go test ./... 2>&1 | tail -20
```
Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add cli/internal/api/client.go cli/internal/api/client_test.go cli/internal/install/installer.go cli/internal/install/installer_test.go
git commit -m "feat(cli): implement createFirstCluster — login to MC, persist JWT token to state"
```

---

## Self-Review

### Spec Coverage Check

| Spec Section | Task |
|---|---|
| §4 `zen upgrade` — pre-flight, backup, dry-run, upgrade, rollout, health, rollback | Tasks 4, 5, 6, 7 (rollback already in upgrade.go) |
| §4 version compatibility — block >1 minor version | Task 5 |
| §4 dry-run with `helm diff upgrade` | Task 7 |
| §7 `zen install --resume` | Already implemented — no change |
| `installPlatform` never installs Helm chart | Task 3 |
| Admin password never reaches Helm | Task 2 + 3 |
| State persistence bug | Task 1 |
| SSH key never saved | Task 1 |
| `createFirstCluster()` placeholder | Task 8 |

### Gaps Not Planned (YAGNI — Phase 3+)
- Installer reliability pass (10 clean runs): a process task, not a code task
- Documentation / README redesign: Phase 3
- `zen node add/remove`: Phase 5

### Type Consistency Check
- `Config.AdminPassword`: set in Task 2, used in Tasks 3 and 8 ✅
- `Config.AdminToken`: added in Task 1, set in Task 8 ✅
- `State.AdminToken`, `State.ZenithVersion`: added in Task 1, written in Tasks 7/5, used in Tasks 5/8 ✅
- `semver.ParseVersion`, `semver.IsSafeUpgrade`: defined in Task 5, called in Task 5 upgrade step ✅
- `buildHelmUpgradeCmd`, `buildHelmDiffCmd`: defined in Tasks 4 and 7, tested in same tasks ✅
- `api.Client.Login`: defined in Task 8, called in `createFirstCluster` same task ✅

---

**Plan complete and saved to `docs/superpowers/plans/2026-06-05-freezenith-phase2.md`.**

**Two execution options:**

**1. Subagent-Driven (recommended)** — Fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
