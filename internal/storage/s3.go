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

// ObjectStorage defines the operations provided by VISP file storage.
type ObjectStorage interface {
	CheckConnection(ctx context.Context) error
}

// S3Storage implements ObjectStorage using an S3-compatible service.
type S3Storage struct {
	client *s3.Client
	bucket string
}

// NewS3Storage initializes an S3-compatible storage client.
func NewS3Storage(cfg config.ObjectStorageConfig) (*S3Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
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

	return &S3Storage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// CheckConnection verifies access to the configured storage bucket.
func (s *S3Storage) CheckConnection(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("check S3 bucket %q: %w", s.bucket, err)
	}

	return nil
}
