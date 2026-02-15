package hetzner

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("test-token")
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if !c.IsConfigured() {
		t.Error("Expected client to be configured with token")
	}
}

func TestNewClientEmpty(t *testing.T) {
	c := NewClient("")
	if c.IsConfigured() {
		t.Error("Expected client to not be configured without token")
	}
}

func TestCreateVolumeNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateVolume(context.Background(), "test-vol", 10, "fsn1")
	if err == nil {
		t.Error("Expected error when creating volume without token")
	}
}

func TestCreateVolumeWithToken(t *testing.T) {
	c := NewClient("test-token")
	vol, err := c.CreateVolume(context.Background(), "test-vol", 20, "fsn1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if vol.Name != "test-vol" {
		t.Errorf("Expected name 'test-vol', got '%s'", vol.Name)
	}
	if vol.SizeGB != 20 {
		t.Errorf("Expected size 20, got %d", vol.SizeGB)
	}
}

func TestDeleteVolumeNoToken(t *testing.T) {
	c := NewClient("")
	err := c.DeleteVolume(context.Background(), 123)
	if err == nil {
		t.Error("Expected error when deleting volume without token")
	}
}

func TestCreateServerNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateServer(context.Background(), "test", "cx22", "ubuntu-22.04", "fsn1")
	if err == nil {
		t.Error("Expected error when creating server without token")
	}
}

func TestCreateServerWithToken(t *testing.T) {
	c := NewClient("test-token")
	srv, err := c.CreateServer(context.Background(), "test", "cx22", "ubuntu-22.04", "fsn1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if srv.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", srv.Name)
	}
}

func TestCreateDNSRecordNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateDNSRecord(context.Background(), "zone1", "A", "app", "1.2.3.4", 300)
	if err == nil {
		t.Error("Expected error when creating DNS record without token")
	}
}

func TestCreateDNSRecordWithToken(t *testing.T) {
	c := NewClient("test-token")
	rec, err := c.CreateDNSRecord(context.Background(), "zone1", "A", "app", "1.2.3.4", 300)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if rec.Type != "A" {
		t.Errorf("Expected type 'A', got '%s'", rec.Type)
	}
}

func TestCreateBucketNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateBucket(context.Background(), "my-bucket", "fsn1")
	if err == nil {
		t.Error("Expected error when creating bucket without token")
	}
}

func TestCreateBucketWithToken(t *testing.T) {
	c := NewClient("test-token")
	bucket, err := c.CreateBucket(context.Background(), "my-bucket", "fsn1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if bucket.Name != "my-bucket" {
		t.Errorf("Expected name 'my-bucket', got '%s'", bucket.Name)
	}
	if bucket.Endpoint == "" {
		t.Error("Expected non-empty endpoint")
	}
}
