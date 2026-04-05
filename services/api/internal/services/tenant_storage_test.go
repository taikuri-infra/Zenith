package services

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// mockObjectStorage is a minimal in-memory S3 implementation for testing TenantStorage.
type mockObjectStorage struct {
	objects map[string]map[string]*mockS3Object // bucket -> key -> object
}

type mockS3Object struct {
	data        []byte
	contentType string
}

func newMockObjectStorage() *mockObjectStorage {
	return &mockObjectStorage{
		objects: make(map[string]map[string]*mockS3Object),
	}
}

func (m *mockObjectStorage) CreateBucket(_ context.Context, bucketName string) error {
	if _, ok := m.objects[bucketName]; !ok {
		m.objects[bucketName] = make(map[string]*mockS3Object)
	}
	return nil
}

func (m *mockObjectStorage) DeleteBucket(_ context.Context, bucketName string) error {
	delete(m.objects, bucketName)
	return nil
}

func (m *mockObjectStorage) PutObject(_ context.Context, bucket, key, contentType string, body io.Reader, size int64) error {
	if _, ok := m.objects[bucket]; !ok {
		m.objects[bucket] = make(map[string]*mockS3Object)
	}
	data, _ := io.ReadAll(body)
	m.objects[bucket][key] = &mockS3Object{data: data, contentType: contentType}
	return nil
}

func (m *mockObjectStorage) GetObject(_ context.Context, bucket, key string) (io.ReadCloser, string, int64, error) {
	if bkt, ok := m.objects[bucket]; ok {
		if obj, ok := bkt[key]; ok {
			return io.NopCloser(bytes.NewReader(obj.data)), obj.contentType, int64(len(obj.data)), nil
		}
	}
	return nil, "", 0, io.ErrUnexpectedEOF
}

func (m *mockObjectStorage) DeleteObject(_ context.Context, bucket, key string) error {
	if bkt, ok := m.objects[bucket]; ok {
		delete(bkt, key)
	}
	return nil
}

func (m *mockObjectStorage) ListObjects(_ context.Context, bucket, prefix, delimiter string, maxKeys int) (*ports.ObjectListResult, error) {
	result := &ports.ObjectListResult{Prefix: prefix}
	if bkt, ok := m.objects[bucket]; ok {
		for key, obj := range bkt {
			if strings.HasPrefix(key, prefix) {
				result.Objects = append(result.Objects, ports.ObjectInfo{
					Key:  key,
					Size: int64(len(obj.data)),
				})
			}
		}
	}
	return result, nil
}

func (m *mockObjectStorage) GeneratePresignedUploadURL(_ context.Context, bucket, key, contentType string, expiry time.Duration) (string, error) {
	return "https://presigned-upload/" + bucket + "/" + key, nil
}

func (m *mockObjectStorage) GeneratePresignedDownloadURL(_ context.Context, bucket, key string, expiry time.Duration) (string, error) {
	return "https://presigned-download/" + bucket + "/" + key, nil
}

func (m *mockObjectStorage) CreateFolder(_ context.Context, bucket, prefix string) error {
	if _, ok := m.objects[bucket]; !ok {
		m.objects[bucket] = make(map[string]*mockS3Object)
	}
	m.objects[bucket][prefix] = &mockS3Object{data: nil, contentType: ""}
	return nil
}

// --- validateKey tests ---

func TestValidateKey_Empty(t *testing.T) {
	err := validateKey("")
	if err == nil {
		t.Error("Expected error for empty key")
	}
}

func TestValidateKey_PathTraversal(t *testing.T) {
	err := validateKey("../../etc/passwd")
	if err == nil {
		t.Error("Expected error for path traversal")
	}
}

func TestValidateKey_AbsolutePath(t *testing.T) {
	err := validateKey("/etc/passwd")
	if err == nil {
		t.Error("Expected error for absolute path")
	}
}

func TestValidateKey_ValidKey(t *testing.T) {
	err := validateKey("images/photo.jpg")
	if err != nil {
		t.Errorf("Expected nil error for valid key, got: %v", err)
	}
}

func TestValidateKey_ValidNestedKey(t *testing.T) {
	err := validateKey("folder/subfolder/file.txt")
	if err != nil {
		t.Errorf("Expected nil error for valid nested key, got: %v", err)
	}
}

// --- TenantStorage with real S3 bucket tests ---

func TestTenantStorage_PutAndGetObject_RealBucket(t *testing.T) {
	s3 := newMockObjectStorage()
	ts := NewTenantStorage(s3, "shared-platform-bucket")
	ctx := context.Background()

	bucket := &entities.UserBucket{
		S3BucketName: "user-real-bucket",
		S3Prefix:     "u/user1/mybucket/",
	}

	err := ts.PutObject(ctx, bucket, "test.txt", "text/plain", strings.NewReader("hello world"), 11)
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	reader, contentType, size, err := ts.GetObject(ctx, bucket, "test.txt")
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	defer reader.Close()

	if contentType != "text/plain" {
		t.Errorf("Expected content type text/plain, got %s", contentType)
	}
	if size != 11 {
		t.Errorf("Expected size 11, got %d", size)
	}

	data, _ := io.ReadAll(reader)
	if string(data) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(data))
	}
}

// --- TenantStorage with shared bucket prefix isolation ---

func TestTenantStorage_PutAndGetObject_SharedBucket(t *testing.T) {
	s3 := newMockObjectStorage()
	ts := NewTenantStorage(s3, "shared-platform-bucket")
	ctx := context.Background()

	bucket := &entities.UserBucket{
		S3BucketName: "", // no real bucket — shared
		S3Prefix:     "u/user1/mybucket/",
	}

	err := ts.PutObject(ctx, bucket, "test.txt", "text/plain", strings.NewReader("shared data"), 11)
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	// Verify it's stored under the prefixed key in the shared bucket
	if _, ok := s3.objects["shared-platform-bucket"]["u/user1/mybucket/test.txt"]; !ok {
		t.Error("Expected object at prefixed key in shared bucket")
	}
}

func TestTenantStorage_ListObjects_SharedBucket(t *testing.T) {
	s3 := newMockObjectStorage()
	ts := NewTenantStorage(s3, "shared-platform-bucket")
	ctx := context.Background()

	bucket := &entities.UserBucket{
		S3BucketName: "",
		S3Prefix:     "u/user1/mybucket/",
	}

	// Put some objects
	ts.PutObject(ctx, bucket, "a.txt", "text/plain", strings.NewReader("a"), 1)
	ts.PutObject(ctx, bucket, "b.txt", "text/plain", strings.NewReader("b"), 1)

	result, err := ts.ListObjects(ctx, bucket, "", "", 100)
	if err != nil {
		t.Fatalf("ListObjects failed: %v", err)
	}

	// Keys should be stripped of prefix
	for _, obj := range result.Objects {
		if strings.HasPrefix(obj.Key, "u/") {
			t.Errorf("Expected prefix to be stripped from key, got: %s", obj.Key)
		}
	}
	if len(result.Objects) != 2 {
		t.Errorf("Expected 2 objects, got %d", len(result.Objects))
	}
}

func TestTenantStorage_DeleteObject(t *testing.T) {
	s3 := newMockObjectStorage()
	ts := NewTenantStorage(s3, "shared-platform-bucket")
	ctx := context.Background()

	bucket := &entities.UserBucket{
		S3BucketName: "user-bucket",
		S3Prefix:     "",
	}

	ts.PutObject(ctx, bucket, "delete-me.txt", "text/plain", strings.NewReader("data"), 4)

	err := ts.DeleteObject(ctx, bucket, "delete-me.txt")
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}

	_, _, _, err = ts.GetObject(ctx, bucket, "delete-me.txt")
	if err == nil {
		t.Error("Expected error after deleting object")
	}
}

func TestTenantStorage_CreateFolder(t *testing.T) {
	s3 := newMockObjectStorage()
	ts := NewTenantStorage(s3, "shared-platform-bucket")
	ctx := context.Background()

	bucket := &entities.UserBucket{
		S3BucketName: "user-bucket",
		S3Prefix:     "",
	}

	err := ts.CreateFolder(ctx, bucket, "myfolder/")
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}
}

func TestTenantStorage_PresignedURLs(t *testing.T) {
	s3 := newMockObjectStorage()
	ts := NewTenantStorage(s3, "shared-platform-bucket")
	ctx := context.Background()

	bucket := &entities.UserBucket{
		S3BucketName: "user-bucket",
		S3Prefix:     "",
	}

	uploadURL, err := ts.GeneratePresignedUploadURL(ctx, bucket, "upload.txt", "text/plain", time.Minute)
	if err != nil {
		t.Fatalf("GeneratePresignedUploadURL failed: %v", err)
	}
	if uploadURL == "" {
		t.Error("Expected non-empty presigned upload URL")
	}

	downloadURL, err := ts.GeneratePresignedDownloadURL(ctx, bucket, "download.txt", time.Minute)
	if err != nil {
		t.Fatalf("GeneratePresignedDownloadURL failed: %v", err)
	}
	if downloadURL == "" {
		t.Error("Expected non-empty presigned download URL")
	}
}

func TestTenantStorage_ValidationErrors(t *testing.T) {
	s3 := newMockObjectStorage()
	ts := NewTenantStorage(s3, "shared-platform-bucket")
	ctx := context.Background()

	bucket := &entities.UserBucket{
		S3BucketName: "user-bucket",
		S3Prefix:     "",
	}

	// PutObject with invalid key
	err := ts.PutObject(ctx, bucket, "", "text/plain", strings.NewReader("data"), 4)
	if err == nil {
		t.Error("Expected error for empty key in PutObject")
	}

	err = ts.PutObject(ctx, bucket, "../escape", "text/plain", strings.NewReader("data"), 4)
	if err == nil {
		t.Error("Expected error for path traversal in PutObject")
	}

	// GetObject with invalid key
	_, _, _, err = ts.GetObject(ctx, bucket, "")
	if err == nil {
		t.Error("Expected error for empty key in GetObject")
	}

	// DeleteObject with invalid key
	err = ts.DeleteObject(ctx, bucket, "/absolute")
	if err == nil {
		t.Error("Expected error for absolute path in DeleteObject")
	}
}

func TestTenantStorage_DeleteAllForBucket(t *testing.T) {
	s3 := newMockObjectStorage()
	ts := NewTenantStorage(s3, "shared-platform-bucket")
	ctx := context.Background()

	bucket := &entities.UserBucket{
		S3BucketName: "cleanup-bucket",
		S3Prefix:     "",
	}

	ts.PutObject(ctx, bucket, "a.txt", "text/plain", strings.NewReader("a"), 1)
	ts.PutObject(ctx, bucket, "b.txt", "text/plain", strings.NewReader("b"), 1)
	ts.PutObject(ctx, bucket, "c.txt", "text/plain", strings.NewReader("c"), 1)

	err := ts.DeleteAllForBucket(ctx, bucket)
	if err != nil {
		t.Fatalf("DeleteAllForBucket failed: %v", err)
	}

	result, _ := ts.ListObjects(ctx, bucket, "", "", 100)
	if len(result.Objects) != 0 {
		t.Errorf("Expected 0 objects after cleanup, got %d", len(result.Objects))
	}
}

func TestTenantStorage_DeleteAllForBucket_SharedPrefix(t *testing.T) {
	s3 := newMockObjectStorage()
	ts := NewTenantStorage(s3, "shared-platform-bucket")
	ctx := context.Background()

	bucket := &entities.UserBucket{
		S3BucketName: "",
		S3Prefix:     "u/user1/mybucket/",
	}

	ts.PutObject(ctx, bucket, "x.txt", "text/plain", strings.NewReader("x"), 1)

	err := ts.DeleteAllForBucket(ctx, bucket)
	if err != nil {
		t.Fatalf("DeleteAllForBucket (shared) failed: %v", err)
	}
}
