package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ensureBucket creates the platform bucket, retrying while object storage
// starts. An existing bucket is success. Replaces the old one-shot
// storage-init job (minio/mc), whose image is no longer published.
func ensureBucket(ctx context.Context, create func(context.Context, string) error, bucket string, attempts int, wait time.Duration) error {
	var last error
	for i := 0; i < attempts; i++ {
		err := create(ctx, bucket)
		if err == nil {
			return nil
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "alreadyownedbyyou") || strings.Contains(msg, "alreadyexists") || strings.Contains(msg, "already exists") {
			return nil
		}
		last = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return fmt.Errorf("ensure bucket %s: %w", bucket, last)
}
