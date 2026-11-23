package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
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
