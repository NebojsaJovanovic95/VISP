package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Database manages the PostgreSQL connection pool.
type Database struct {
	pool *pgxpool.Pool
}

// New creates a database connection pool and verifies connectivity.
func New(ctx context.Context) (*Database, error) {
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	dbName := os.Getenv("POSTGRES_DB")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	sslMode := os.Getenv("POSTGRES_SSLMODE")

	if host == "" || port == "" || dbName == "" ||
		user == "" || password == "" || sslMode == "" {
		return nil, fmt.Errorf("missing required PostgreSQL environment variables")
	}

	connURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   dbName,
	}

	query := url.Values{}
	query.Set("sslmode", sslMode)
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
