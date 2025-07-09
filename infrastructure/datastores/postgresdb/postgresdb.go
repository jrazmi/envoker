package postgresdb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jrazmi/envoker/sdk/environment"
)

// PostgreSQL error codes
const (
	uniqueViolation = "23505"
	undefinedTable  = "42P01"
)

// Set of error variables for CRUD operations.
var (
	ErrDBNotFound        = pgx.ErrNoRows
	ErrDBDuplicatedEntry = errors.New("duplicated entry")
	ErrUndefinedTable    = errors.New("undefined table")
)

// Status enums
var (
	StatusActive   = "active"
	StatusArchived = "archived"
)

type Pool = pgxpool.Pool

// Config represents the database environment
type Config struct {
	DatabaseURL string        `env:"DATABASE_URL" default:"postgres://postgres:password@localhost:5432/postgres?sslmode=disable"`
	MaxConns    int           `env:"DATABASE_MAX_CONNS" default:"25"`
	MinConns    int           `env:"DATABASE_MIN_CONNS" default:"5"`
	MaxLifetime time.Duration `env:"DATABASE_MAX_LIFETIME" default:"1h"`
	MaxIdleTime time.Duration `env:"DATABASE_MAX_IDLE_TIME" default:"30m"`
	HealthCheck time.Duration `env:"DATABASE_HEALTH_CHECK" default:"1m"`
	// LogQueries  bool          `env:"DATABASE_LOG_QUERIES" default:"false"`
}

// NewDatabaseFromEnv creates a new database connection using environment variables
func NewDatabaseFromEnv(prefix string) (*pgxpool.Pool, error) {
	var cfg Config

	if err := environment.ParseEnvTags(prefix, &cfg); err != nil {
		return nil, fmt.Errorf("parsing database config: %w", err)
	}

	return Open(cfg)
}

// Open creates a new database connection with the given config
func Open(cfg Config) (*pgxpool.Pool, error) {
	// Configure the connection pool
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	// Set pool environment
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = cfg.MaxLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheck

	// Create the connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return pool, nil
}

// StatusCheck returns nil if it can successfully talk to the database
func StatusCheck(ctx context.Context, pool *pgxpool.Pool) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second)
		defer cancel()
	}

	return pool.Ping(ctx)
}

// HandlePgError converts PostgreSQL errors to application errors
func HandlePgError(err error) error {
	if err == nil {
		return nil
	}

	var pqerr *pgconn.PgError
	if errors.As(err, &pqerr) {
		switch pqerr.Code {
		case undefinedTable:
			return ErrUndefinedTable
		case uniqueViolation:
			return ErrDBDuplicatedEntry
		}
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDBNotFound
	}

	return err
}
