package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/credentials"
)

type ImageStorage interface {
	SaveImage(ctx context.Context, r io.Reader, filename string) (string, error)
	LoadImage(ctx context.Context, filename string) (io.ReadCloser, error)
	DeleteImage(ctx context.Context, filename string) error
}

type LocalStorage struct {
	BaseDir string
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create image directory: %w", err)
	}

	return &LocalStorage{
		BaseDir: baseDir,
	}, nil
}

func (s *LocalStorage) SaveImage(
	ctx context.Context,
	r io.Reader,
	filename string,
) (string, error) {
	path := filepath.Join(s.BaseDir, filename)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("create image directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create image file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		return "", fmt.Errorf("write image: %w", err)
	}

	return filename, nil
}

func (s *LocalStorage) LoadImage(
	ctx context.Context,
	filename string,
) (io.ReadCloser, error) {
	path := filepath.Join(s.BaseDir, filename)

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("image not found: %w", err)
		}
		return nil, fmt.Errorf("open image: %w", err)
	}

	return file, nil
}

func (s *LocalStorage) DeleteImage(
	ctx context.Context,
	filename string,
) error {
	path := filepath.Join(s.BaseDir, filename)

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("delete image: %w", err)
	}

	return nil
}

// ------------------------------------------------------------
// MinIO
// ------------------------------------------------------------

type MinIOStorage struct {
	Client *minio.Client
	Bucket string
}

func NewMinIOStorage(
	endpoint string,
	accessKey string,
	secretKey string,
	bucket string,
) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, accessKey, secretKey, false)
	//		endpoint, &minio.Options{
	//		Creds: credentials.NewStaticV4(
	//			accessKey,``
	//			secretKey,
	//			"",
	//		),
	//		Secure: false,
	//	})
	if err != nil {
		return nil, fmt.Errorf("create MinIO client: %w", err)
	}

	ctx := context.Background()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("check MinIO bucket: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create MinIO bucket: %w", err)
		}
	}

	return &MinIOStorage{
		Client: client,
		Bucket: bucket,
	}, nil
}

func (s *MinIOStorage) SaveImage(
	ctx context.Context,
	r io.Reader,
	filename string,
) (string, error) {
	_, err := s.Client.PutObject(
		ctx,
		s.Bucket,
		filename,
		r,
		-1,
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		},
	)
	if err != nil {
		return "", fmt.Errorf("upload image to MinIO: %w", err)
	}

	return filename, nil
}

func (s *MinIOStorage) LoadImage(
	ctx context.Context,
	filename string,
) (io.ReadCloser, error) {
	object, err := s.Client.GetObject(
		ctx,
		s.Bucket,
		filename,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("get image from MinIO: %w", err)
	}

	// GetObject is lazy, so actually check that the object exists.
	if _, err := object.Stat(); err != nil {
		object.Close()
		return nil, fmt.Errorf("image not found: %w", err)
	}

	return object, nil
}

func (s *MinIOStorage) DeleteImage(
	ctx context.Context,
	filename string,
) error {
	err := s.Client.RemoveObject(
		ctx,
		s.Bucket,
		filename,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("delete image from MinIO: %w", err)
	}

	return nil
}

// Make sure both implementations satisfy the interface.
var _ ImageStorage = (*LocalStorage)(nil)
var _ ImageStorage = (*MinIOStorage)(nil)
