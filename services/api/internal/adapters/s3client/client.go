package s3client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// S3API is an alias for ports.ObjectStorage. Kept for backward compatibility.
type S3API = ports.ObjectStorage

// Compile-time checks.
var _ ports.ObjectStorage = (*Client)(nil)
var _ ports.ObjectStorage = (*MemoryS3Client)(nil)

// Client implements S3API using AWS SDK v2 (Hetzner S3-compatible).
type Client struct {
	s3Client *s3.Client
}

// NewClient creates an S3 client targeting Hetzner Object Storage.
func NewClient(endpoint, accessKey, secretKey, region string) *Client {
	s3Client := s3.New(s3.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       region,
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		UsePathStyle: true,
	})
	return &Client{s3Client: s3Client}
}

// CreateBucket creates an S3 bucket for a tenant.
func (c *Client) CreateBucket(ctx context.Context, bucketName string) error {
	_, err := c.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("create S3 bucket %s: %w", bucketName, err)
	}
	return nil
}

// DeleteBucket deletes an S3 bucket.
func (c *Client) DeleteBucket(ctx context.Context, bucketName string) error {
	_, err := c.s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("delete S3 bucket %s: %w", bucketName, err)
	}
	return nil
}

// ListObjects lists objects in a bucket with optional prefix/delimiter filtering.
func (c *Client) ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int) (*ports.ObjectListResult, error) {
	if maxKeys <= 0 {
		maxKeys = 1000
	}
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(int32(maxKeys)),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}
	if delimiter != "" {
		input.Delimiter = aws.String(delimiter)
	}

	out, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("list objects in %s: %w", bucket, err)
	}

	result := &ports.ObjectListResult{
		Prefix:      prefix,
		IsTruncated: aws.ToBool(out.IsTruncated),
	}

	for _, obj := range out.Contents {
		result.Objects = append(result.Objects, ports.ObjectInfo{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
			ETag:         aws.ToString(obj.ETag),
		})
	}

	for _, cp := range out.CommonPrefixes {
		result.CommonPrefixes = append(result.CommonPrefixes, aws.ToString(cp.Prefix))
	}

	return result, nil
}

// DeleteObject deletes an object from a bucket.
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object %s/%s: %w", bucket, key, err)
	}
	return nil
}

// GeneratePresignedUploadURL creates a presigned PUT URL for uploading an object.
func (c *Client) GeneratePresignedUploadURL(ctx context.Context, bucket, key, contentType string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c.s3Client)
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	out, err := presignClient.PresignPutObject(ctx, input, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign upload %s/%s: %w", bucket, key, err)
	}
	return out.URL, nil
}

// GeneratePresignedDownloadURL creates a presigned GET URL for downloading an object.
func (c *Client) GeneratePresignedDownloadURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c.s3Client)
	out, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign download %s/%s: %w", bucket, key, err)
	}
	return out.URL, nil
}

// CreateFolder creates a zero-byte object representing a folder.
func (c *Client) CreateFolder(ctx context.Context, bucket, prefix string) error {
	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(prefix),
		Body:          bytes.NewReader([]byte{}),
		ContentLength: aws.Int64(0),
	})
	if err != nil {
		return fmt.Errorf("create folder %s/%s: %w", bucket, prefix, err)
	}
	return nil
}

// PutObject uploads an object to a bucket.
func (c *Client) PutObject(ctx context.Context, bucket, key, contentType string, body io.Reader, size int64) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}
	_, err := c.s3Client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("put object %s/%s: %w", bucket, key, err)
	}
	return nil
}

// GetObject downloads an object from a bucket. Returns body, contentType, size.
func (c *Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, string, int64, error) {
	out, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", 0, fmt.Errorf("get object %s/%s: %w", bucket, key, err)
	}
	contentType := ""
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return out.Body, contentType, size, nil
}

// MemoryS3Client is a no-op implementation for dev/test.
type MemoryS3Client struct{}

func NewMemoryClient() *MemoryS3Client { return &MemoryS3Client{} }

func (m *MemoryS3Client) CreateBucket(_ context.Context, _ string) error { return nil }
func (m *MemoryS3Client) DeleteBucket(_ context.Context, _ string) error { return nil }

func (m *MemoryS3Client) ListObjects(_ context.Context, _, prefix, _ string, _ int) (*ports.ObjectListResult, error) {
	// Return sample objects for dev/test
	result := &ports.ObjectListResult{
		Prefix: prefix,
		Objects: []ports.ObjectInfo{
			{Key: prefix + "readme.txt", Size: 1024, LastModified: time.Now().Add(-24 * time.Hour), ETag: "\"abc123\""},
			{Key: prefix + "data.json", Size: 4096, LastModified: time.Now().Add(-2 * time.Hour), ETag: "\"def456\""},
		},
		CommonPrefixes: []string{prefix + "images/", prefix + "docs/"},
	}
	return result, nil
}

func (m *MemoryS3Client) DeleteObject(_ context.Context, _, _ string) error { return nil }

func (m *MemoryS3Client) GeneratePresignedUploadURL(_ context.Context, bucket, key, _ string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("https://%s.s3.zenith.local/%s?X-Amz-Expires=%.0f", bucket, key, expiry.Seconds()), nil
}

func (m *MemoryS3Client) GeneratePresignedDownloadURL(_ context.Context, bucket, key string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("https://%s.s3.zenith.local/%s?X-Amz-Expires=%.0f", bucket, key, expiry.Seconds()), nil
}

func (m *MemoryS3Client) PutObject(_ context.Context, _, _, _ string, _ io.Reader, _ int64) error {
	return nil
}

func (m *MemoryS3Client) GetObject(_ context.Context, _, key string) (io.ReadCloser, string, int64, error) {
	content := []byte("sample file content for " + key)
	return io.NopCloser(bytes.NewReader(content)), "application/octet-stream", int64(len(content)), nil
}

func (m *MemoryS3Client) CreateFolder(_ context.Context, _, _ string) error { return nil }
