package storage

import (
	"context"
	"fmt"

	"github.com/NebojsaJovanovic95/VISP/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Storage manages the connection to S3-compatible object storage.
type Storage struct {
	client *s3.Client
	bucket string
}

// New creates an S3-compatible storage client and verifies connectivity.
func New(ctx context.Context, cfg config.ObjectStorageConfig) (*Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load S3 configuration: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = true
	})

	storage := &Storage{
		client: client,
		bucket: cfg.Bucket,
	}

	if err := storage.CheckConnection(ctx); err != nil {
		return nil, err
	}

	return storage, nil
}

// CheckConnection verifies access to the configured storage bucket.
func (s *Storage) CheckConnection(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("check S3 bucket %q: %w", s.bucket, err)
	}

	return nil
}
