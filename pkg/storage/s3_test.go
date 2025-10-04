package storage

import (
	"context"
	"testing"
)

func TestNewS3Client_ValidConfig(t *testing.T) {
	ctx := context.Background()
	cfg := &S3Config{
		Bucket:          "test-bucket",
		Region:          "us-west-2",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
	}

	client, err := NewS3Client(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	if client.bucket != "test-bucket" {
		t.Errorf("Expected bucket to be 'test-bucket', got '%s'", client.bucket)
	}
}

func TestNewS3Client_MissingBucket(t *testing.T) {
	ctx := context.Background()
	cfg := &S3Config{
		Region:          "us-west-2",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
	}

	_, err := NewS3Client(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for missing bucket, got nil")
	}

	expectedError := "bucket name is required"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestNewS3Client_MissingRegion(t *testing.T) {
	ctx := context.Background()
	cfg := &S3Config{
		Bucket:          "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
	}

	_, err := NewS3Client(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for missing region, got nil")
	}

	expectedError := "region is required"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGenerateBackupKey(t *testing.T) {
	clusterName := "test-cluster"
	key := GenerateBackupKey(clusterName)

	if key == "" {
		t.Fatal("Expected non-empty key")
	}

	// Key should start with the etcd-backups prefix
	expectedPrefix := "etcd-backups/test-cluster/etcd-snapshot-"
	if len(key) < len(expectedPrefix) {
		t.Fatalf("Key too short: %s", key)
	}

	actualPrefix := key[:len(expectedPrefix)]
	if actualPrefix != expectedPrefix {
		t.Errorf("Expected key to start with '%s', got '%s'", expectedPrefix, actualPrefix)
	}

	// Key should end with .db
	expectedSuffix := ".db"
	if len(key) < len(expectedSuffix) {
		t.Fatalf("Key too short: %s", key)
	}

	actualSuffix := key[len(key)-len(expectedSuffix):]
	if actualSuffix != expectedSuffix {
		t.Errorf("Expected key to end with '%s', got '%s'", expectedSuffix, actualSuffix)
	}
}

func TestNewS3Client_WithEndpoint(t *testing.T) {
	ctx := context.Background()
	cfg := &S3Config{
		Bucket:          "test-bucket",
		Region:          "us-west-2",
		Endpoint:        "https://minio.example.com",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
	}

	client, err := NewS3Client(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}
}

func TestNewS3Client_WithInsecureSkipVerify(t *testing.T) {
	ctx := context.Background()
	cfg := &S3Config{
		Bucket:             "test-bucket",
		Region:             "us-west-2",
		Endpoint:           "https://minio.example.com",
		AccessKeyID:        "test-key",
		SecretAccessKey:    "test-secret",
		InsecureSkipVerify: true,
	}

	client, err := NewS3Client(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}
}
