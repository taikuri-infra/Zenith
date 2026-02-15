package hetzner

import (
	"context"
	"math"
	"strings"
	"testing"
)

// ============================================================================
// NewClient Tests
// ============================================================================

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

func TestNewClient_WithToken_HcloudClientNotNil(t *testing.T) {
	c := NewClient("hc_test_token_123")
	if c.hcloud == nil {
		t.Error("Expected hcloud client to be non-nil when token is provided")
	}
	if c.token != "hc_test_token_123" {
		t.Errorf("Expected token 'hc_test_token_123', got '%s'", c.token)
	}
}

func TestNewClient_EmptyToken_HcloudClientNil(t *testing.T) {
	c := NewClient("")
	if c.hcloud != nil {
		t.Error("Expected hcloud client to be nil when token is empty")
	}
	if c.token != "" {
		t.Errorf("Expected empty token, got '%s'", c.token)
	}
}

// ============================================================================
// IsConfigured Tests
// ============================================================================

func TestIsConfigured_WithToken(t *testing.T) {
	c := NewClient("valid-token")
	if !c.IsConfigured() {
		t.Error("Expected IsConfigured to return true with valid token")
	}
}

func TestIsConfigured_WithoutToken(t *testing.T) {
	c := NewClient("")
	if c.IsConfigured() {
		t.Error("Expected IsConfigured to return false without token")
	}
}

func TestIsConfigured_ManuallyNilHcloud(t *testing.T) {
	// Test the edge case where token is set but hcloud is nil
	c := &Client{token: "some-token", hcloud: nil}
	if c.IsConfigured() {
		t.Error("Expected IsConfigured to return false when hcloud is nil even with token")
	}
}

// ============================================================================
// Volume Operation Error Tests
// ============================================================================

func TestCreateVolumeNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateVolume(context.Background(), "test-vol", 10, "fsn1")
	if err == nil {
		t.Error("Expected error when creating volume without token")
	}
}

func TestCreateVolumeNoToken_ErrorMessage(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateVolume(context.Background(), "test-vol", 10, "fsn1")
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "hetzner token not configured") {
		t.Errorf("Expected error to contain 'hetzner token not configured', got '%s'", err.Error())
	}
}

func TestDeleteVolumeNoToken(t *testing.T) {
	c := NewClient("")
	err := c.DeleteVolume(context.Background(), 123)
	if err == nil {
		t.Error("Expected error when deleting volume without token")
	}
}

func TestDeleteVolumeNoToken_ErrorMessage(t *testing.T) {
	c := NewClient("")
	err := c.DeleteVolume(context.Background(), 123)
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "hetzner token not configured") {
		t.Errorf("Expected error to contain 'hetzner token not configured', got '%s'", err.Error())
	}
}

func TestGetVolumeNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.GetVolume(context.Background(), 123)
	if err == nil {
		t.Error("Expected error when getting volume without token")
	}
}

func TestGetVolumeNoToken_ErrorMessage(t *testing.T) {
	c := NewClient("")
	_, err := c.GetVolume(context.Background(), 123)
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "hetzner token not configured") {
		t.Errorf("Expected error to contain 'hetzner token not configured', got '%s'", err.Error())
	}
}

func TestResizeVolumeNoToken(t *testing.T) {
	c := NewClient("")
	err := c.ResizeVolume(context.Background(), 123, 50)
	if err == nil {
		t.Error("Expected error when resizing volume without token")
	}
}

func TestResizeVolumeNoToken_ErrorMessage(t *testing.T) {
	c := NewClient("")
	err := c.ResizeVolume(context.Background(), 123, 50)
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "hetzner token not configured") {
		t.Errorf("Expected error to contain 'hetzner token not configured', got '%s'", err.Error())
	}
}

// ============================================================================
// Server Operation Error Tests
// ============================================================================

func TestCreateServerNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateServer(context.Background(), "test", "cx22", "ubuntu-22.04", "fsn1")
	if err == nil {
		t.Error("Expected error when creating server without token")
	}
}

func TestCreateServerNoToken_ErrorMessage(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateServer(context.Background(), "test", "cx22", "ubuntu-22.04", "fsn1")
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "hetzner token not configured") {
		t.Errorf("Expected error to contain 'hetzner token not configured', got '%s'", err.Error())
	}
}

func TestDeleteServerNoToken(t *testing.T) {
	c := NewClient("")
	err := c.DeleteServer(context.Background(), 123)
	if err == nil {
		t.Error("Expected error when deleting server without token")
	}
}

func TestDeleteServerNoToken_ErrorMessage(t *testing.T) {
	c := NewClient("")
	err := c.DeleteServer(context.Background(), 123)
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "hetzner token not configured") {
		t.Errorf("Expected error to contain 'hetzner token not configured', got '%s'", err.Error())
	}
}

// ============================================================================
// DNS Operation Error Tests
// ============================================================================

func TestCreateDNSRecordNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateDNSRecord(context.Background(), "zone1", "A", "app", "1.2.3.4", 300)
	if err == nil {
		t.Error("Expected error when creating DNS record without token")
	}
}

func TestCreateDNSRecordNoToken_ErrorMessage(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateDNSRecord(context.Background(), "zone1", "A", "app", "1.2.3.4", 300)
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "hetzner token not configured") {
		t.Errorf("Expected error to contain 'hetzner token not configured', got '%s'", err.Error())
	}
}

func TestDeleteDNSRecordNoToken(t *testing.T) {
	c := NewClient("")
	err := c.DeleteDNSRecord(context.Background(), "record-123")
	if err == nil {
		t.Error("Expected error when deleting DNS record without token")
	}
}

func TestDeleteDNSRecordNoToken_ErrorMessage(t *testing.T) {
	c := NewClient("")
	err := c.DeleteDNSRecord(context.Background(), "record-123")
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "hetzner token not configured") {
		t.Errorf("Expected error to contain 'hetzner token not configured', got '%s'", err.Error())
	}
}

// ============================================================================
// Bucket Operation Error Tests
// ============================================================================

func TestCreateBucketNoToken(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateBucket(context.Background(), "my-bucket", "fsn1")
	if err == nil {
		t.Error("Expected error when creating bucket without token")
	}
}

func TestCreateBucketNoToken_ErrorMessage(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateBucket(context.Background(), "my-bucket", "fsn1")
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "hetzner token not configured") {
		t.Errorf("Expected error to contain 'hetzner token not configured', got '%s'", err.Error())
	}
}

func TestDeleteBucketNoToken(t *testing.T) {
	c := NewClient("")
	err := c.DeleteBucket(context.Background(), "my-bucket")
	if err == nil {
		t.Error("Expected error when deleting bucket without token")
	}
}

func TestDeleteBucketNoToken_ErrorMessage(t *testing.T) {
	c := NewClient("")
	err := c.DeleteBucket(context.Background(), "my-bucket")
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "hetzner token not configured") {
		t.Errorf("Expected error to contain 'hetzner token not configured', got '%s'", err.Error())
	}
}

// ============================================================================
// All Operations Return Nil Results on Error
// ============================================================================

func TestAllOperationsReturnNilOnUnconfigured(t *testing.T) {
	c := NewClient("")

	vol, err := c.CreateVolume(context.Background(), "v", 10, "fsn1")
	if vol != nil {
		t.Error("Expected nil volume on error")
	}
	if err == nil {
		t.Error("Expected error")
	}

	gotVol, err := c.GetVolume(context.Background(), 1)
	if gotVol != nil {
		t.Error("Expected nil volume on error")
	}
	if err == nil {
		t.Error("Expected error")
	}

	srv, err := c.CreateServer(context.Background(), "s", "cx22", "ubuntu", "fsn1")
	if srv != nil {
		t.Error("Expected nil server on error")
	}
	if err == nil {
		t.Error("Expected error")
	}

	dns, err := c.CreateDNSRecord(context.Background(), "z", "A", "n", "v", 300)
	if dns != nil {
		t.Error("Expected nil DNS record on error")
	}
	if err == nil {
		t.Error("Expected error")
	}

	bucket, err := c.CreateBucket(context.Background(), "b", "fsn1")
	if bucket != nil {
		t.Error("Expected nil bucket on error")
	}
	if err == nil {
		t.Error("Expected error")
	}
}

// ============================================================================
// ParseVolumeID Tests
// ============================================================================

func TestParseVolumeID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"valid positive", "123", 123, false},
		{"zero", "0", 0, false},
		{"large number", "999999", 999999, false},
		{"empty string", "", 0, true},
		{"alpha string", "abc", 0, true},
		{"decimal number", "12.5", 0, true},
		{"negative number", "-1", -1, false},
		{"max int64", "9223372036854775807", math.MaxInt64, false},
		{"whitespace", " 123 ", 0, true},
		{"hex prefix", "0x1F", 0, true},
		{"leading zero", "0123", 123, false},
		{"just whitespace", "   ", 0, true},
		{"mixed alpha-numeric", "123abc", 0, true},
		{"special characters", "12!3", 0, true},
		{"newline in string", "12\n3", 0, true},
		{"plus sign", "+42", 42, false},
		{"very large number", "99999999999999999999", math.MaxInt64, true}, // overflow returns max with error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVolumeID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVolumeID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseVolumeID(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// ============================================================================
// Struct Field Tests
// ============================================================================

func TestVolumeStruct(t *testing.T) {
	v := Volume{
		ID:       42,
		Name:     "test-vol",
		SizeGB:   100,
		Location: "fsn1",
		Status:   "available",
	}

	if v.ID != 42 {
		t.Errorf("Expected ID 42, got %d", v.ID)
	}
	if v.Name != "test-vol" {
		t.Errorf("Expected Name 'test-vol', got '%s'", v.Name)
	}
	if v.SizeGB != 100 {
		t.Errorf("Expected SizeGB 100, got %d", v.SizeGB)
	}
	if v.Location != "fsn1" {
		t.Errorf("Expected Location 'fsn1', got '%s'", v.Location)
	}
	if v.Status != "available" {
		t.Errorf("Expected Status 'available', got '%s'", v.Status)
	}
}

func TestServerStruct(t *testing.T) {
	s := Server{
		ID:         99,
		Name:       "web-01",
		ServerType: "cx22",
		Status:     "running",
		PublicIPv4: "1.2.3.4",
		PrivateIP:  "10.0.0.1",
	}

	if s.ID != 99 {
		t.Errorf("Expected ID 99, got %d", s.ID)
	}
	if s.Name != "web-01" {
		t.Errorf("Expected Name 'web-01', got '%s'", s.Name)
	}
	if s.ServerType != "cx22" {
		t.Errorf("Expected ServerType 'cx22', got '%s'", s.ServerType)
	}
	if s.PublicIPv4 != "1.2.3.4" {
		t.Errorf("Expected PublicIPv4 '1.2.3.4', got '%s'", s.PublicIPv4)
	}
}

func TestDNSRecordStruct(t *testing.T) {
	r := DNSRecord{
		ID:     "dns-123",
		Type:   "A",
		Name:   "app",
		Value:  "1.2.3.4",
		TTL:    300,
		ZoneID: "zone-abc",
	}

	if r.ID != "dns-123" {
		t.Errorf("Expected ID 'dns-123', got '%s'", r.ID)
	}
	if r.Type != "A" {
		t.Errorf("Expected Type 'A', got '%s'", r.Type)
	}
	if r.TTL != 300 {
		t.Errorf("Expected TTL 300, got %d", r.TTL)
	}
}

func TestObjectStoreBucketStruct(t *testing.T) {
	b := ObjectStoreBucket{
		Name:      "my-bucket",
		Endpoint:  "https://my-bucket.fsn1.your-objectstorage.com",
		Region:    "fsn1",
		AccessKey: "access-key-123",
		SecretKey: "secret-key-456",
	}

	if b.Name != "my-bucket" {
		t.Errorf("Expected Name 'my-bucket', got '%s'", b.Name)
	}
	if b.Endpoint != "https://my-bucket.fsn1.your-objectstorage.com" {
		t.Errorf("Expected Endpoint to be set, got '%s'", b.Endpoint)
	}
	if b.Region != "fsn1" {
		t.Errorf("Expected Region 'fsn1', got '%s'", b.Region)
	}
}

// ============================================================================
// Consistency Tests - Multiple Clients
// ============================================================================

func TestMultipleClients_Independent(t *testing.T) {
	c1 := NewClient("token-1")
	c2 := NewClient("token-2")
	c3 := NewClient("")

	if !c1.IsConfigured() {
		t.Error("Expected c1 to be configured")
	}
	if !c2.IsConfigured() {
		t.Error("Expected c2 to be configured")
	}
	if c3.IsConfigured() {
		t.Error("Expected c3 to not be configured")
	}

	// Verify they are independent instances
	if c1 == c2 {
		t.Error("Expected different client instances")
	}
	if c1.token == c2.token {
		t.Error("Expected different tokens")
	}
}

func TestNewClient_VariousTokenFormats(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		configured bool
	}{
		{"empty", "", false},
		{"simple", "abc", true},
		{"hetzner format", "hc_abcdef123456", true},
		{"with special chars", "token-with-dashes_and_underscores", true},
		{"single char", "x", true},
		{"very long", strings.Repeat("a", 1000), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.token)
			if c == nil {
				t.Fatal("NewClient returned nil")
			}
			if c.IsConfigured() != tt.configured {
				t.Errorf("Expected IsConfigured=%v for token '%s'", tt.configured, tt.token)
			}
		})
	}
}
