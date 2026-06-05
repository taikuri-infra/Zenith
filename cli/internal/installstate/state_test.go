package installstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-state.yaml")

	s := &State{
		Domain:            "example.com",
		ServerIP:          "1.2.3.4",
		MissionControlURL: "https://mission.example.com",
		CloudURL:          "https://cloud.example.com",
		AdminUser:         "admin",
		AdminPassword:     "secret123",
		Provider:          "hetzner",
		Region:            "fsn1",
		ServerID:          42,
		InstalledAt:       "2026-06-05T12:00:00Z",
	}

	if err := SaveTo(s, path); err != nil {
		t.Fatalf("SaveTo error: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom error: %v", err)
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
	if loaded.ServerID != s.ServerID {
		t.Errorf("ServerID: got %d, want %d", loaded.ServerID, s.ServerID)
	}
	if loaded.MissionControlURL != s.MissionControlURL {
		t.Errorf("MissionControlURL: got %q, want %q", loaded.MissionControlURL, s.MissionControlURL)
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
		ServerID:          100,
		SSHKeyID:          200,
		InstalledAt:       "2026-06-05T12:00:00Z",
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
		t.Errorf("SSHKeyID: got %d, want %d", loaded.SSHKeyID, s.SSHKeyID)
	}
}
