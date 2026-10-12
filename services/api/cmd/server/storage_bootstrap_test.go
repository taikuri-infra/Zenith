package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEnsureBucket(t *testing.T) {
	calls := 0
	create := func(context.Context, string) error {
		calls++
		if calls < 3 {
			return errors.New("dial tcp storage:9000: connection refused")
		}
		return nil
	}
	if err := ensureBucket(context.Background(), create, "b", 5, time.Millisecond); err != nil || calls != 3 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}

	exists := func(context.Context, string) error {
		return errors.New("create S3 bucket b: BucketAlreadyOwnedByYou: Your previous request to create the named bucket succeeded")
	}
	if err := ensureBucket(context.Background(), exists, "b", 2, time.Millisecond); err != nil {
		t.Fatalf("already-owned must count as success: %v", err)
	}

	never := func(context.Context, string) error { return errors.New("connection refused") }
	if err := ensureBucket(context.Background(), never, "b", 2, time.Millisecond); err == nil {
		t.Fatal("want error after attempts are exhausted")
	}
}
