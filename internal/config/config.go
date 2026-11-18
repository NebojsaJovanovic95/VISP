package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Address string
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

// Load reads application configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Address: getEnv("VISP_SERVER_ADDRESS", ":8080"),
		},
		Database: DatabaseConfig{
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			Name:     os.Getenv("POSTGRES_DB"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			SSLMode:  os.Getenv("POSTGRES_SSLMODE"),
		},
	}

	if cfg.Database.Host == "" ||
		cfg.Database.Port == "" ||
		cfg.Database.Name == "" ||
		cfg.Database.User == "" ||
		cfg.Database.Password == "" ||
		cfg.Database.SSLMode == "" {
		return nil, fmt.Errorf("missing required PostgreSQL environment variables")
	}

	return cfg, nil
}

// getEnv returns an environment variable or its fallback value.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
