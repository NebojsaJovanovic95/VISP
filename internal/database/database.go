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
