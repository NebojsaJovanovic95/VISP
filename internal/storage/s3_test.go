package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful connection",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "failed connection",
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodHead {
						t.Errorf("expected HEAD request, got %s", r.Method)
					}

					if r.URL.Path != "/visp" {
						t.Errorf("expected bucket path %q, got %q", "/visp", r.URL.Path)
					}

					w.WriteHeader(tt.statusCode)
				},
			))
			defer server.Close()

			_, err := New(
				context.Background(),
				config.ObjectStorageConfig{
					Endpoint:  server.URL,
					Region:    "us-east-1",
					Bucket:    "visp",
					AccessKey: "test-access-key",
					SecretKey: "test-secret-key",
				},
			)

			if tt.wantErr && err == nil {
				t.Fatal("expected connection error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected connection error: %v", err)
			}
		})
	}
}

func TestUpload(t *testing.T) {
	const (
		bucket      = "visp"
		objectKey   = "frames/test-image.png"
		contentType = "image/png"
		imageData   = "png-data"
	)

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful upload",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "failed upload",
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case http.MethodHead:
						if r.URL.Path != "/"+bucket {
							t.Errorf("expected bucket path %q, got %q", "/"+bucket, r.URL.Path)
						}

						w.WriteHeader(http.StatusOK)

					case http.MethodPut:
						if r.URL.Path != "/"+bucket+"/"+objectKey {
							t.Errorf(
								"expected object path %q, got %q",
								"/"+bucket+"/"+objectKey,
								r.URL.Path,
							)
						}

						if got := r.Header.Get("Content-Type"); got != contentType {
							t.Errorf("expected content type %q, got %q", contentType, got)
						}

						body, err := io.ReadAll(r.Body)
						if err != nil {
							t.Errorf("read upload body: %v", err)
						}

						if string(body) != imageData {
							t.Errorf("expected body %q, got %q", imageData, string(body))
						}

						w.WriteHeader(tt.statusCode)

					default:
						t.Errorf("unexpected HTTP method %s", r.Method)
						w.WriteHeader(http.StatusMethodNotAllowed)
					}
				},
			))
			defer server.Close()

			storage, err := New(
				context.Background(),
				config.ObjectStorageConfig{
					Endpoint:  server.URL,
					Region:    "us-east-1",
					Bucket:    bucket,
					AccessKey: "test-access-key",
					SecretKey: "test-secret-key",
				},
			)
			if err != nil {
				t.Fatalf("initialize storage: %v", err)
			}

			ref, err := storage.Upload(
				context.Background(),
				objectKey,
				bytes.NewReader([]byte(imageData)),
				int64(len(imageData)),
				contentType,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected upload error, got nil")
				}

				if ref != "" {
					t.Errorf("expected empty reference on failure, got %q", ref)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected upload error: %v", err)
			}

			if ref != objectKey {
				t.Errorf("expected reference %q, got %q", objectKey, ref)
			}
		})
	}
}

func TestDownload(t *testing.T) {
	const (
		bucket    = "visp"
		objectKey = "frames/test-image.png"
		imageData = "png-data"
	)

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful download",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "failed download",
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case http.MethodHead:
						if r.URL.Path != "/"+bucket {
							t.Errorf(
								"expected bucket path %q, got %q",
								"/"+bucket,
								r.URL.Path,
							)
						}

						w.WriteHeader(http.StatusOK)

					case http.MethodGet:
						if r.URL.Path != "/"+bucket+"/"+objectKey {
							t.Errorf(
								"expected object path %q, got %q",
								"/"+bucket+"/"+objectKey,
								r.URL.Path,
							)
						}

						w.WriteHeader(tt.statusCode)

						if tt.statusCode == http.StatusOK {
							_, err := w.Write([]byte(imageData))
							if err != nil {
								t.Errorf("write download response: %v", err)
							}
						}

					default:
						t.Errorf("unexpected HTTP method %s", r.Method)
						w.WriteHeader(http.StatusMethodNotAllowed)
					}
				},
			))
			defer server.Close()

			storage, err := New(
				context.Background(),
				config.ObjectStorageConfig{
					Endpoint:  server.URL,
					Region:    "us-east-1",
					Bucket:    bucket,
					AccessKey: "test-access-key",
					SecretKey: "test-secret-key",
				},
			)
			if err != nil {
				t.Fatalf("initialize storage: %v", err)
			}

			body, err := storage.Download(context.Background(), objectKey)

			if tt.wantErr {
				if err == nil {
					if body != nil {
						body.Close()
					}
					t.Fatal("expected download error, got nil")
				}

				if body != nil {
					body.Close()
					t.Fatal("expected nil body on failure")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected download error: %v", err)
			}
			defer body.Close()

			got, err := io.ReadAll(body)
			if err != nil {
				t.Fatalf("read downloaded image: %v", err)
			}

			if string(got) != imageData {
				t.Errorf("expected body %q, got %q", imageData, string(got))
			}
		})
	}
}

func TestS3Integration(t *testing.T) {
	if os.Getenv("VISP_S3_INTEGRATION_TEST") != "1" {
		t.Skip("S3 integration test is not enabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	objectStorage, err := New(ctx, cfg.ObjectStorage)
	if err != nil {
		t.Fatalf("initialize storage: %v", err)
	}

	objectKey := "integration-tests/" + uuid.NewString() + ".txt"
	imageData := []byte("VISP SeaweedFS integration test")

	_, err = objectStorage.Upload(
		ctx,
		objectKey,
		bytes.NewReader(imageData),
		int64(len(imageData)),
		"text/plain",
	)
	if err != nil {
		t.Fatalf("upload test object: %v", err)
	}

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		if err := objectStorage.Delete(cleanupCtx, objectKey); err != nil {
			t.Errorf("delete test object: %v", err)
		}
	}()

	body, err := objectStorage.Download(ctx, objectKey)
	if err != nil {
		t.Fatalf("download test object: %v", err)
	}
	defer body.Close()

	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read downloaded object: %v", err)
	}

	if !bytes.Equal(got, imageData) {
		t.Errorf("expected downloaded data %q, got %q", imageData, got)
	}
}

func TestDelete(t *testing.T) {
	const (
		bucket    = "visp"
		objectKey = "frames/test-image.png"
	)

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful deletion",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "failed deletion",
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case http.MethodHead:
						if r.URL.Path != "/"+bucket {
							t.Errorf(
								"expected bucket path %q, got %q",
								"/"+bucket,
								r.URL.Path,
							)
						}

						w.WriteHeader(http.StatusOK)

					case http.MethodDelete:
						if r.URL.Path != "/"+bucket+"/"+objectKey {
							t.Errorf(
								"expected object path %q, got %q",
								"/"+bucket+"/"+objectKey,
								r.URL.Path,
							)
						}

						w.WriteHeader(tt.statusCode)

					default:
						t.Errorf("unexpected HTTP method %s", r.Method)
						w.WriteHeader(http.StatusMethodNotAllowed)
					}
				},
			))
			defer server.Close()

			storage, err := New(
				context.Background(),
				config.ObjectStorageConfig{
					Endpoint:  server.URL,
					Region:    "us-east-1",
					Bucket:    bucket,
					AccessKey: "test-access-key",
					SecretKey: "test-secret-key",
				},
			)
			if err != nil {
				t.Fatalf("initialize storage: %v", err)
			}

			err = storage.Delete(context.Background(), objectKey)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected deletion error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected deletion error: %v", err)
			}
		})
	}
}
