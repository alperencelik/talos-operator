package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps the AWS S3 client for etcd backup operations
type S3Client struct {
	client *s3.Client
	bucket string
}

// S3Config contains configuration for S3 client
type S3Config struct {
	Bucket             string
	Region             string
	Endpoint           string
	AccessKeyID        string
	SecretAccessKey    string
	InsecureSkipVerify bool
}

// NewS3Client creates a new S3 client with the provided configuration
func NewS3Client(ctx context.Context, cfg *S3Config) (*S3Client, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("region is required")
	}

	// Build AWS config options
	var opts []func(*config.LoadOptions) error
	opts = append(opts, config.WithRegion(cfg.Region))

	// Set credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Build S3 client options
	s3Opts := []func(*s3.Options){}

	// Set custom endpoint if provided (for S3-compatible storage)
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // Required for most S3-compatible storage
		})
	}

	// Handle InsecureSkipTLSVerify
	if cfg.InsecureSkipVerify {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.HTTPClient = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		})
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg, s3Opts...)

	return &S3Client{
		client: s3Client,
		bucket: cfg.Bucket,
	}, nil
}

// Upload uploads data from a reader to S3 with the specified key
// This method streams data directly to S3 without buffering in memory or disk
func (s *S3Client) Upload(ctx context.Context, key string, reader io.Reader) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	// Use PutObject for streaming upload
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// GenerateBackupKey generates a standardized key for etcd backups
func GenerateBackupKey(clusterName string) string {
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	return fmt.Sprintf("etcd-backups/%s/etcd-snapshot-%s.db", clusterName, timestamp)
}
