package storage

import (
	"context"
	"fmt"
	"io"

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

// Upload stores an object and returns its object key as the reference.
func (s *Storage) Upload(
	ctx context.Context,
	key string,
	body io.Reader,
	contentLength int64,
	contentType string,
) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(contentLength),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf(
			"upload object %q to S3 bucket %q: %w",
			key,
			s.bucket,
			err,
		)
	}

	return key, nil
}

// Download retrieves an object from storage and returns its contents.
func (s *Storage) Download(
	ctx context.Context,
	key string,
) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf(
			"download object %q from S3 bucket %q: %w",
			key,
			s.bucket,
			err,
		)
	}

	return result.Body, nil
}
