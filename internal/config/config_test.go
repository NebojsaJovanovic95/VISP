package config

import (
	"testing"
	"time"
)

func setDatabaseEnv(t *testing.T) {
	t.Helper()

	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DB", "visp")
	t.Setenv("POSTGRES_USER", "visp")
	t.Setenv("POSTGRES_PASSWORD", "test-password")
	t.Setenv("POSTGRES_SSLMODE", "disable")
}

func setObjectStorageEnv(t *testing.T) {
	t.Helper()

	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_REGION", "us-east-1")
	t.Setenv("S3_BUCKET", "visp")
	t.Setenv("S3_ACCESS_KEY", "test-access-key")
	t.Setenv("S3_SECRET_KEY", "test-secret-key")
}

func clearObjectStorageEnv(t *testing.T) {
	t.Helper()

	t.Setenv("S3_ENDPOINT", "")
	t.Setenv("S3_REGION", "")
	t.Setenv("S3_BUCKET", "")
	t.Setenv("S3_ACCESS_KEY", "")
	t.Setenv("S3_SECRET_KEY", "")
}

func TestLoadServerDefaults(t *testing.T) {
	setDatabaseEnv(t)
	setObjectStorageEnv(t)
	t.Setenv("VISP_SERVER_ADDRESS", "")
	t.Setenv("IMAGE_RETENTION_PERIOD", "")
	t.Setenv("IMAGE_RETENTION_CHECK_INTERVAL", "")

	cfg, err := LoadServer()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}

	if cfg.Address != ":8080" {
		t.Errorf("expected server address %q, got %q", ":8080", cfg.Address)
	}

	if cfg.ImageRetentionPeriod != 168*time.Hour {
		t.Errorf("expected retention period %v, got %v",
			168*time.Hour, cfg.ImageRetentionPeriod)
	}

	if cfg.ImageRetentionCheckInterval != time.Hour {
		t.Errorf("expected check interval %v, got %v",
			time.Hour, cfg.ImageRetentionCheckInterval)
	}
}

func TestLoadServerCustomAddress(t *testing.T) {
	setDatabaseEnv(t)
	setObjectStorageEnv(t)
	t.Setenv("VISP_SERVER_ADDRESS", "localhost:9090")

	cfg, err := LoadServer()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}

	if cfg.Address != "localhost:9090" {
		t.Errorf("expected server address %q, got %q", "localhost:9090", cfg.Address)
	}
}

func TestLoadServerObjectStorage(t *testing.T) {
	setDatabaseEnv(t)
	setObjectStorageEnv(t)

	cfg, err := LoadServer()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}

	storage := cfg.ObjectStorage

	if storage.Endpoint != "http://localhost:9000" {
		t.Errorf("expected endpoint %q, got %q", "http://localhost:9000", storage.Endpoint)
	}

	if storage.Region != "us-east-1" {
		t.Errorf("expected region %q, got %q", "us-east-1", storage.Region)
	}

	if storage.Bucket != "visp" {
		t.Errorf("expected bucket %q, got %q", "visp", storage.Bucket)
	}

	if storage.AccessKey != "test-access-key" {
		t.Errorf("expected access key %q, got %q", "test-access-key", storage.AccessKey)
	}

	if storage.SecretKey != "test-secret-key" {
		t.Errorf("expected secret key %q, got %q", "test-secret-key", storage.SecretKey)
	}
}

func TestLoadServerMissingObjectStorageConfig(t *testing.T) {
	setDatabaseEnv(t)
	clearObjectStorageEnv(t)

	_, err := LoadServer()
	if err == nil {
		t.Fatal("expected configuration error, got nil")
	}
}

func TestLoadServerInvalidObjectStorageEndpoint(t *testing.T) {
	setDatabaseEnv(t)
	setObjectStorageEnv(t)
	t.Setenv("S3_ENDPOINT", "not-a-valid-url")

	_, err := LoadServer()
	if err == nil {
		t.Fatal("expected configuration error, got nil")
	}
}

func TestLoadServerMissingDatabaseConfig(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("POSTGRES_DB", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_SSLMODE", "")
	setObjectStorageEnv(t)

	_, err := LoadServer()
	if err == nil {
		t.Fatal("expected configuration error, got nil")
	}
}

func TestLoadAcquisitionDefaults(t *testing.T) {
	t.Setenv("VISP_SERVER_URL", "")

	cfg, err := LoadAcquisition()
	if err != nil {
		t.Fatalf("load acquisition configuration: %v", err)
	}

	if cfg.ServerURL != "http://localhost:8080" {
		t.Errorf("expected server URL %q, got %q", "http://localhost:8080", cfg.ServerURL)
	}
}

func TestLoadAcquisitionCustomServerURL(t *testing.T) {
	t.Setenv("VISP_SERVER_URL", "http://localhost:9090")

	cfg, err := LoadAcquisition()
	if err != nil {
		t.Fatalf("load acquisition configuration: %v", err)
	}

	if cfg.ServerURL != "http://localhost:9090" {
		t.Errorf("expected server URL %q, got %q", "http://localhost:9090", cfg.ServerURL)
	}
}

func TestLoadAcquisitionInvalidServerURL(t *testing.T) {
	t.Setenv("VISP_SERVER_URL", "not-a-valid-url")

	_, err := LoadAcquisition()
	if err == nil {
		t.Fatal("expected configuration error, got nil")
	}
}

func TestLoadAcquisitionDoesNotRequireDatabase(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("POSTGRES_DB", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_SSLMODE", "")

	_, err := LoadAcquisition()
	if err != nil {
		t.Fatalf("acquisition configuration should not require database settings: %v", err)
	}
}

func TestLoadServerCustomRetentionConfig(t *testing.T) {
	setDatabaseEnv(t)
	setObjectStorageEnv(t)

	t.Setenv("IMAGE_RETENTION_PERIOD", "72h")
	t.Setenv("IMAGE_RETENTION_CHECK_INTERVAL", "30m")

	cfg, err := LoadServer()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}

	if cfg.ImageRetentionPeriod != 72*time.Hour {
		t.Errorf("expected retention period %v, got %v",
			72*time.Hour, cfg.ImageRetentionPeriod)
	}

	if cfg.ImageRetentionCheckInterval != 30*time.Minute {
		t.Errorf("expected check interval %v, got %v",
			30*time.Minute, cfg.ImageRetentionCheckInterval)
	}
}

func TestLoadServerInvalidRetentionConfig(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		value string
	}{
		{
			name:  "invalid retention period",
			env:   "IMAGE_RETENTION_PERIOD",
			value: "invalid",
		},
		{
			name:  "zero retention period",
			env:   "IMAGE_RETENTION_PERIOD",
			value: "0s",
		},
		{
			name:  "negative retention period",
			env:   "IMAGE_RETENTION_PERIOD",
			value: "-1h",
		},
		{
			name:  "invalid check interval",
			env:   "IMAGE_RETENTION_CHECK_INTERVAL",
			value: "invalid",
		},
		{
			name:  "zero check interval",
			env:   "IMAGE_RETENTION_CHECK_INTERVAL",
			value: "0s",
		},
		{
			name:  "negative check interval",
			env:   "IMAGE_RETENTION_CHECK_INTERVAL",
			value: "-1m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setDatabaseEnv(t)
			setObjectStorageEnv(t)

			t.Setenv("IMAGE_RETENTION_PERIOD", "168h")
			t.Setenv("IMAGE_RETENTION_CHECK_INTERVAL", "1h")
			t.Setenv(tt.env, tt.value)

			_, err := LoadServer()
			if err == nil {
				t.Fatal("expected configuration error, got nil")
			}
		})
	}
}
