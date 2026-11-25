package config

import (
	"fmt"
	"net/url"
	"os"
	"time"
)

// ServerConfig holds configuration required by the VISP Server.
type ServerConfig struct {
	Address                     string
	Database                    DatabaseConfig
	ObjectStorage               ObjectStorageConfig
	ImageRetentionPeriod        time.Duration
	ImageRetentionCheckInterval time.Duration
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

// ObjectStorageConfig holds S3-compatible object storage settings.
type ObjectStorageConfig struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

// rawConfig holds the complete application configuration internally.
type rawConfig struct {
	ServerAddress               string
	ServerURL                   string
	Database                    DatabaseConfig
	ObjectStorage               ObjectStorageConfig
	ImageRetentionPeriod        string
	ImageRetentionCheckInterval string
}

// load reads the complete application configuration from environment variables.
func load() *rawConfig {
	return &rawConfig{
		ServerAddress:               getEnv("VISP_SERVER_ADDRESS", ":8080"),
		ServerURL:                   getEnv("VISP_SERVER_URL", "http://localhost:8080"),
		ImageRetentionPeriod:        getEnv("IMAGE_RETENTION_PERIOD", "168h"),
		ImageRetentionCheckInterval: getEnv("IMAGE_RETENTION_CHECK_INTERVAL", "1h"),
		Database: DatabaseConfig{
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			Name:     os.Getenv("POSTGRES_DB"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			SSLMode:  os.Getenv("POSTGRES_SSLMODE"),
		},
		ObjectStorage: ObjectStorageConfig{
			Endpoint:  os.Getenv("S3_ENDPOINT"),
			Region:    os.Getenv("S3_REGION"),
			Bucket:    os.Getenv("S3_BUCKET"),
			AccessKey: os.Getenv("S3_ACCESS_KEY"),
			SecretKey: os.Getenv("S3_SECRET_KEY"),
		},
	}
}

// LoadServer loads and validates the VISP Server configuration.
func LoadServer() (*ServerConfig, error) {
	cfg := load()

	if err := validateDatabase(cfg.Database); err != nil {
		return nil, err
	}

	if err := validateObjectStorage(cfg.ObjectStorage); err != nil {
		return nil, err
	}

	retentionPeriod, err := time.ParseDuration(cfg.ImageRetentionPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid IMAGE_RETENTION_PERIOD: %w", err)
	}
	if retentionPeriod <= 0 {
		return nil, fmt.Errorf("IMAGE_RETENTION_PERIOD must be positive")
	}

	retentionCheckInterval, err := time.ParseDuration(cfg.ImageRetentionCheckInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid IMAGE_RETENTION_CHECK_INTERVAL: %w", err)
	}
	if retentionCheckInterval <= 0 {
		return nil, fmt.Errorf("IMAGE_RETENTION_CHECK_INTERVAL must be positive")
	}

	return &ServerConfig{
		Address:                     cfg.ServerAddress,
		Database:                    cfg.Database,
		ObjectStorage:               cfg.ObjectStorage,
		ImageRetentionPeriod:        retentionPeriod,
		ImageRetentionCheckInterval: retentionCheckInterval,
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

func validateObjectStorage(cfg ObjectStorageConfig) error {
	if cfg.Endpoint == "" ||
		cfg.Region == "" ||
		cfg.Bucket == "" ||
		cfg.AccessKey == "" ||
		cfg.SecretKey == "" {
		return fmt.Errorf("missing required S3 object storage environment variables")
	}

	endpoint, err := url.ParseRequestURI(cfg.Endpoint)
	if err != nil || endpoint.Host == "" ||
		(endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return fmt.Errorf("invalid S3_ENDPOINT: %q", cfg.Endpoint)
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
