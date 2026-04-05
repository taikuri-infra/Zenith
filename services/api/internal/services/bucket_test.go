package services

import (
	"context"
	"strings"
	"testing"
)

// --- GenerateS3BucketName tests ---

func TestGenerateS3BucketName_Basic(t *testing.T) {
	name := GenerateS3BucketName("user123", "mybucket")
	if name != "zenith-user123-mybucket" {
		t.Errorf("Expected 'zenith-user123-mybucket', got '%s'", name)
	}
}

func TestGenerateS3BucketName_LongUserID(t *testing.T) {
	name := GenerateS3BucketName("abcdefghijklmnopqrstuvwxyz", "mybucket")
	// Should truncate userID to 12 chars
	if !strings.HasPrefix(name, "zenith-abcdefghijkl-mybucket") {
		t.Errorf("Expected userID to be truncated to 12 chars, got '%s'", name)
	}
}

func TestGenerateS3BucketName_MaxLength(t *testing.T) {
	longName := strings.Repeat("x", 100)
	name := GenerateS3BucketName("user123", longName)
	if len(name) > 63 {
		t.Errorf("Expected bucket name <= 63 chars, got %d: '%s'", len(name), name)
	}
}

func TestGenerateS3BucketName_Lowercase(t *testing.T) {
	name := GenerateS3BucketName("USER123", "MyBucket")
	if name != strings.ToLower(name) {
		t.Errorf("Expected lowercase bucket name, got '%s'", name)
	}
}

func TestGenerateS3BucketName_SpecialChars(t *testing.T) {
	name := GenerateS3BucketName("user@#$123", "mybucket")
	// Should sanitize special chars from userID
	if strings.ContainsAny(name, "@#$") {
		t.Errorf("Expected special chars to be removed, got '%s'", name)
	}
}

// --- sanitizeBucketSegment tests ---

func TestSanitizeBucketSegment(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"Hello-World", "hello-world"},
		{"user@123", "user123"},
		{"a_b_c", "abc"},
		{"", ""},
		{"ABC-123", "abc-123"},
		{"special!@#$chars", "specialchars"},
	}

	for _, tc := range cases {
		got := sanitizeBucketSegment(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeBucketSegment(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// --- BucketService tests ---

func TestBucketService_CreateAndDeleteBucket(t *testing.T) {
	s3 := newMockObjectStorage()
	bs := NewBucketService(s3)
	ctx := context.Background()

	err := bs.CreateRealBucket(ctx, "test-bucket")
	if err != nil {
		t.Fatalf("CreateRealBucket failed: %v", err)
	}

	// Put some objects
	s3.PutObject(ctx, "test-bucket", "a.txt", "text/plain", strings.NewReader("a"), 1)
	s3.PutObject(ctx, "test-bucket", "b.txt", "text/plain", strings.NewReader("b"), 1)

	err = bs.DeleteRealBucket(ctx, "test-bucket")
	if err != nil {
		t.Fatalf("DeleteRealBucket failed: %v", err)
	}

	// Bucket should be deleted
	if _, ok := s3.objects["test-bucket"]; ok {
		t.Error("Expected bucket to be deleted")
	}
}

func TestBucketService_DeleteEmptyBucket(t *testing.T) {
	s3 := newMockObjectStorage()
	bs := NewBucketService(s3)
	ctx := context.Background()

	err := bs.CreateRealBucket(ctx, "empty-bucket")
	if err != nil {
		t.Fatalf("CreateRealBucket failed: %v", err)
	}

	err = bs.DeleteRealBucket(ctx, "empty-bucket")
	if err != nil {
		t.Fatalf("DeleteRealBucket (empty) failed: %v", err)
	}
}
