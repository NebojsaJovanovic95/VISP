package config

import "testing"

func TestLoadServerDefaults(t *testing.T) {
	setDatabaseEnv(t)
	t.Setenv("VISP_SERVER_ADDRESS", "")

	cfg, err := LoadServer()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}

	if cfg.Address != ":8080" {
		t.Errorf("expected server address %q, got %q", ":8080", cfg.Address)
	}
}

func TestLoadServerCustomAddress(t *testing.T) {
	setDatabaseEnv(t)
	t.Setenv("VISP_SERVER_ADDRESS", "localhost:9090")

	cfg, err := LoadServer()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}

	if cfg.Address != "localhost:9090" {
		t.Errorf("expected server address %q, got %q", "localhost:9090", cfg.Address)
	}
}

func TestLoadServerMissingDatabaseConfig(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("POSTGRES_DB", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_SSLMODE", "")

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

func setDatabaseEnv(t *testing.T) {
	t.Helper()

	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DB", "visp")
	t.Setenv("POSTGRES_USER", "visp")
	t.Setenv("POSTGRES_PASSWORD", "test-password")
	t.Setenv("POSTGRES_SSLMODE", "disable")
}
