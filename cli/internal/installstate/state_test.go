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

	if err := MarkStepComplete(s, "Provision server"); err != nil {
		t.Fatalf("Second MarkStepComplete failed: %v", err)
	}
	if len(s.CompletedSteps) != 1 {
		t.Errorf("Expected 1 completed step, got %d", len(s.CompletedSteps))
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := LoadFrom("/tmp/nonexistent-zen-state-xyz-abc/install-state.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "install-state.yaml")

	s := &State{Domain: "test.com"}
	if err := SaveTo(s, path); err != nil {
		t.Fatalf("SaveTo error: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected state file to be created")
	}
}

func TestSave_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-state.yaml")

	s := &State{Domain: "test.com", AdminPassword: "secret"}
	if err := SaveTo(s, path); err != nil {
		t.Fatalf("SaveTo error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected file permissions 0600, got %o", info.Mode().Perm())
	}
}

func TestRoundtrip_AllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-state.yaml")

	s := &State{
		Domain:            "example.com",
		ServerIP:          "10.0.0.1",
		MissionControlURL: "https://mc.example.com",
		CloudURL:          "https://cloud.example.com",
		AdminUser:         "admin",
		AdminPassword:     "pass",
		SSHKeyPath:        "/home/user/.zen/keys/id_rsa",
		Provider:          "hetzner",
		Region:            "fsn1",
		ServerID:          "100",
		SSHKeyID:          "200",
		InstalledAt:       time.Now().UTC().Truncate(time.Second),
	}

	if err := SaveTo(s, path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SSHKeyPath != s.SSHKeyPath {
		t.Errorf("SSHKeyPath: got %q, want %q", loaded.SSHKeyPath, s.SSHKeyPath)
	}
	if loaded.SSHKeyID != s.SSHKeyID {
		t.Errorf("SSHKeyID: got %q, want %q", loaded.SSHKeyID, s.SSHKeyID)
	}
	if loaded.ServerID != s.ServerID {
		t.Errorf("ServerID: got %q, want %q", loaded.ServerID, s.ServerID)
	}
}

func TestState_NewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.yaml")

	s := &State{
		Domain:        "example.com",
		ZenithVersion: "1.2.3",
	}
	if err := SaveTo(s, path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}
	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if loaded.ZenithVersion != "1.2.3" {
		t.Errorf("ZenithVersion: got %q, want %q", loaded.ZenithVersion, "1.2.3")
	}
}

func TestSaveAndLoad_ServerHostKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-state.yaml")

	s := &State{
		Domain:        "example.com",
		ServerHostKey: "AAAAB3NzaC1yc2EAAAADAQAB...", // fake base64 key
	}
	if err := SaveTo(s, path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if loaded.ServerHostKey != s.ServerHostKey {
		t.Errorf("ServerHostKey: got %q, want %q", loaded.ServerHostKey, s.ServerHostKey)
	}
}
