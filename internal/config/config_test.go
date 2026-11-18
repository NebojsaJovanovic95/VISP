package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	setDatabaseEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	if cfg.Server.Address != ":8080" {
		t.Errorf("expected server address %q, got %q", ":8080", cfg.Server.Address)
	}
}

func TestLoadCustomServerAddress(t *testing.T) {
	setDatabaseEnv(t)
	t.Setenv("VISP_SERVER_ADDRESS", "localhost:9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	if cfg.Server.Address != "localhost:9090" {
		t.Errorf("expected server address %q, got %q", "localhost:9090", cfg.Server.Address)
	}
}

func TestLoadMissingDatabaseConfig(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("POSTGRES_DB", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_SSLMODE", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected configuration error, got nil")
	}
}

func setDatabaseEnv(t *testing.T) {
	t.Helper()

	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DB", "visp")
	t.Setenv("POSTGRES_USER", "visp")
	t.Setenv("POSTGRES_PASSWORD", "test-password")
	t.Setenv("POSTGRES_SSLMODE", "disable")
}
