package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
)

func TestS3StorageCheckConnection(t *testing.T) {
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

			storage, err := NewS3Storage(config.ObjectStorageConfig{
				Endpoint:  server.URL,
				Region:    "us-east-1",
				Bucket:    "visp",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
			})
			if err != nil {
				t.Fatalf("initialize S3 storage: %v", err)
			}

			err = storage.CheckConnection(context.Background())

			if tt.wantErr && err == nil {
				t.Fatal("expected connection error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected connection error: %v", err)
			}
		})
	}
}
