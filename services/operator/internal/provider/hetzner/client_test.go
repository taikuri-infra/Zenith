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

func TestDeleteVolumeNoToken(t *testing.T) {
	c := NewClient("")
	err := c.DeleteVolume(context.Background(), 123)
	if err == nil {
		t.Error("Expected error when deleting volume without token")
	}
}

func TestGetVolumeNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.GetVolume(context.Background(), 123)
	if err == nil {
		t.Error("Expected error when getting volume without token")
	}
}

func TestResizeVolumeNoToken(t *testing.T) {
	c := NewClient("")
	err := c.ResizeVolume(context.Background(), 123, 50)
	if err == nil {
		t.Error("Expected error when resizing volume without token")
	}
}

func TestCreateServerNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateServer(context.Background(), "test", "cx22", "ubuntu-22.04", "fsn1")
	if err == nil {
		t.Error("Expected error when creating server without token")
	}
}

func TestDeleteServerNoToken(t *testing.T) {
	c := NewClient("")
	err := c.DeleteServer(context.Background(), 123)
	if err == nil {
		t.Error("Expected error when deleting server without token")
	}
}

func TestCreateDNSRecordNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateDNSRecord(context.Background(), "zone1", "A", "app", "1.2.3.4", 300)
	if err == nil {
		t.Error("Expected error when creating DNS record without token")
	}
}

func TestDeleteDNSRecordNoToken(t *testing.T) {
	c := NewClient("")
	err := c.DeleteDNSRecord(context.Background(), "record-123")
	if err == nil {
		t.Error("Expected error when deleting DNS record without token")
	}
}

func TestCreateBucketNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateBucket(context.Background(), "my-bucket", "fsn1")
	if err == nil {
		t.Error("Expected error when creating bucket without token")
	}
}

func TestDeleteBucketNoToken(t *testing.T) {
	c := NewClient("")
	err := c.DeleteBucket(context.Background(), "my-bucket")
	if err == nil {
		t.Error("Expected error when deleting bucket without token")
	}
}

func TestParseVolumeID(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"123", 123, false},
		{"0", 0, false},
		{"999999", 999999, false},
		{"", 0, true},
		{"abc", 0, true},
		{"12.5", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseVolumeID(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseVolumeID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseVolumeID(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
