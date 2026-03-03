package s3client

import (
	"context"
	"fmt"

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

// MemoryS3Client is a no-op implementation for dev/test.
type MemoryS3Client struct{}

func NewMemoryClient() *MemoryS3Client { return &MemoryS3Client{} }

func (m *MemoryS3Client) CreateBucket(_ context.Context, _ string) error { return nil }
func (m *MemoryS3Client) DeleteBucket(_ context.Context, _ string) error { return nil }
