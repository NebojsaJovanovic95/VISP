package database

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Database manages the PostgreSQL connection pool.
type Database struct {
	pool *pgxpool.Pool
}

// CameraConfig represents the configuration of a registered camera.
type CameraConfig struct {
	CameraID                   string
	CameraName                 string
	RTSPURL                    string
	Location                   *string
	Status                     string
	AcquisitionIntervalSeconds float64
}

// New creates a database connection pool and verifies connectivity.
func New(ctx context.Context, cfg config.DatabaseConfig) (*Database, error) {
	connURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   cfg.Name,
	}

	query := url.Values{}
	query.Set("sslmode", cfg.SSLMode)
	connURL.RawQuery = query.Encode()

	pool, err := pgxpool.New(ctx, connURL.String())
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return &Database{
		pool: pool,
	}, nil
}

// Close releases all connections in the pool.
func (db *Database) Close() {
	db.pool.Close()
}

// ListCameras retrieves all registered cameras.
func (db *Database) ListCameras(ctx context.Context) ([]CameraConfig, error) {
	rows, err := db.pool.Query(ctx, `
        SELECT
            camera_id::text,
            camera_name,
            rtsp_url,
            location,
            status,
            EXTRACT(EPOCH FROM acquisition_interval)
        FROM cameras
        ORDER BY camera_name;
    `)
	if err != nil {
		return nil, fmt.Errorf("query cameras: %w", err)
	}
	defer rows.Close()

	cameras := []CameraConfig{}

	for rows.Next() {
		var camera CameraConfig

		if err := rows.Scan(
			&camera.CameraID,
			&camera.CameraName,
			&camera.RTSPURL,
			&camera.Location,
			&camera.Status,
			&camera.AcquisitionIntervalSeconds,
		); err != nil {
			return nil, fmt.Errorf("scan camera: %w", err)
		}

		cameras = append(cameras, camera)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cameras: %w", err)
	}

	return cameras, nil
}

// CameraExists checks whether a camera is registered.
func (db *Database) CameraExists(ctx context.Context, cameraID string) (bool, error) {
	var exists bool

	err := db.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM cameras
			WHERE camera_id = $1
		);
	`, cameraID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check camera existence: %w", err)
	}

	return exists, nil
}
