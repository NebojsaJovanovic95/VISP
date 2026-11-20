package config

import (
	"fmt"
	"net/url"
	"os"
)

// ServerConfig holds configuration required by the VISP Server.
type ServerConfig struct {
	Address  string
	Database DatabaseConfig
}

// AcquisitionConfig holds configuration required by Camera Acquisition.
type AcquisitionConfig struct {
	ServerURL string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

// rawConfig holds the complete application configuration internally.
type rawConfig struct {
	ServerAddress string
	ServerURL     string
	Database      DatabaseConfig
}

// load reads the complete application configuration from environment variables.
func load() *rawConfig {
	return &rawConfig{
		ServerAddress: getEnv("VISP_SERVER_ADDRESS", ":8080"),
		ServerURL:     getEnv("VISP_SERVER_URL", "http://localhost:8080"),
		Database: DatabaseConfig{
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			Name:     os.Getenv("POSTGRES_DB"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			SSLMode:  os.Getenv("POSTGRES_SSLMODE"),
		},
	}
}

// LoadServer loads and validates the VISP Server configuration.
func LoadServer() (*ServerConfig, error) {
	cfg := load()

	if err := validateDatabase(cfg.Database); err != nil {
		return nil, err
	}

	return &ServerConfig{
		Address:  cfg.ServerAddress,
		Database: cfg.Database,
	}, nil
}

// LoadAcquisition loads and validates Camera Acquisition configuration.
func LoadAcquisition() (*AcquisitionConfig, error) {
	cfg := load()

	serverURL, err := url.ParseRequestURI(cfg.ServerURL)
	if err != nil || serverURL.Host == "" ||
		(serverURL.Scheme != "http" && serverURL.Scheme != "https") {
		return nil, fmt.Errorf("invalid VISP_SERVER_URL: %q", cfg.ServerURL)
	}

	return &AcquisitionConfig{
		ServerURL: cfg.ServerURL,
	}, nil
}

func validateDatabase(cfg DatabaseConfig) error {
	if cfg.Host == "" ||
		cfg.Port == "" ||
		cfg.Name == "" ||
		cfg.User == "" ||
		cfg.Password == "" ||
		cfg.SSLMode == "" {
		return fmt.Errorf("missing required PostgreSQL environment variables")
	}

	return nil
}

// getEnv returns an environment variable or its fallback value.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
