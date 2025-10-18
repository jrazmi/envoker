package postgresdb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jrazmi/envoker/sdk/environment"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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

// Options represents the exportable database configuration
type Options struct {
	DatabaseURL string        `env:"PG_DATABASE_URL" default:"postgres://postgres:password@localhost:5432/postgres?sslmode=disable"`
	MaxConns    int           `env:"PG_DATABASE_MAX_CONNS" default:"25"`
	MinConns    int           `env:"PG_DATABASE_MIN_CONNS" default:"5"`
	MaxLifetime time.Duration `env:"PG_DATABASE_MAX_LIFETIME" default:"1h"`
	MaxIdleTime time.Duration `env:"PG_DATABASE_MAX_IDLE_TIME" default:"30m"`
	HealthCheck time.Duration `env:"PG_DATABASE_HEALTH_CHECK" default:"1m"`
}

// options holds the internal runtime configuration
type options struct {
	databaseURL    string
	maxConns       int
	minConns       int
	maxLifetime    time.Duration
	maxIdleTime    time.Duration
	healthCheck    time.Duration
	logger         *slog.Logger
	tracer         pgx.QueryTracer
	connectTimeout time.Duration
	logQueries     bool
}

// Option is a function that configures the database options
type Option func(*options)

// WithLogger sets a custom logger for the database
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithTracer sets a custom query tracer
func WithTracer(tracer pgx.QueryTracer) Option {
	return func(o *options) {
		o.tracer = tracer
	}
}

// WithDatabaseURL overrides the database URL
func WithDatabaseURL(url string) Option {
	return func(o *options) {
		o.databaseURL = url
	}
}

// WithMaxConns sets the maximum number of connections
func WithMaxConns(max int) Option {
	return func(o *options) {
		o.maxConns = max
	}
}

// WithMinConns sets the minimum number of connections
func WithMinConns(min int) Option {
	return func(o *options) {
		o.minConns = min
	}
}

// WithMaxLifetime sets the maximum connection lifetime
func WithMaxLifetime(lifetime time.Duration) Option {
	return func(o *options) {
		o.maxLifetime = lifetime
	}
}

// WithMaxIdleTime sets the maximum idle time for connections
func WithMaxIdleTime(idleTime time.Duration) Option {
	return func(o *options) {
		o.maxIdleTime = idleTime
	}
}

// WithHealthCheck sets the health check period
func WithHealthCheck(period time.Duration) Option {
	return func(o *options) {
		o.healthCheck = period
	}
}

// WithConnectTimeout sets the connection timeout
func WithConnectTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.connectTimeout = timeout
	}
}

// WithLogQueries enables or disables query logging
func WithLogQueries(enable bool) Option {
	return func(o *options) {
		o.logQueries = enable
	}
}

// NewFromEnv creates a new database connection using environment variables
func NewFromEnv(prefix string, opts ...Option) (*pgxpool.Pool, error) {
	var cfg Options
	if err := environment.ParseEnvTags(prefix, &cfg); err != nil {
		return nil, fmt.Errorf("parsing database config: %w", err)
	}
	return newDatabase(cfg, opts...)
}

// NewTestDB creates a test database connection
func NewTestDB(conn string, opts ...Option) (*pgxpool.Pool, error) {
	cfg := Options{
		DatabaseURL: conn,
		MaxConns:    25,
		MinConns:    5,
		MaxLifetime: time.Hour,
		MaxIdleTime: time.Hour,
		HealthCheck: time.Hour,
	}
	return newDatabase(cfg, opts...)
}

// newDatabase creates a new database connection with given config and applies options
func newDatabase(cfg Options, opts ...Option) (*pgxpool.Pool, error) {
	// Start with config-based options
	internalOpts := &options{
		databaseURL:    cfg.DatabaseURL,
		maxConns:       cfg.MaxConns,
		minConns:       cfg.MinConns,
		maxLifetime:    cfg.MaxLifetime,
		maxIdleTime:    cfg.MaxIdleTime,
		healthCheck:    cfg.HealthCheck,
		connectTimeout: 10 * time.Second, // default
		logQueries:     false,            // default
	}

	// Apply functional options to override config
	for _, opt := range opts {
		opt(internalOpts)
	}

	// Set up default logger if none provided
	if internalOpts.logger == nil {
		internalOpts.logger = slog.Default()
	}

	// Set up default tracer if needed
	if internalOpts.tracer == nil && internalOpts.logQueries {
		internalOpts.tracer = NewMultiQueryTracer(
			NewLoggingQueryTracer(internalOpts.logger),
		)
	}

	return openDatabase(internalOpts)
}

// openDatabase creates the actual database connection
func openDatabase(opts *options) (*pgxpool.Pool, error) {
	// Configure the connection pool
	poolConfig, err := pgxpool.ParseConfig(opts.databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	// Set pool configuration
	poolConfig.MaxConns = int32(opts.maxConns)
	poolConfig.MinConns = int32(opts.minConns)
	poolConfig.MaxConnLifetime = opts.maxLifetime
	poolConfig.MaxConnIdleTime = opts.maxIdleTime
	poolConfig.HealthCheckPeriod = opts.healthCheck

	// Set tracer if provided
	if opts.tracer != nil {
		poolConfig.ConnConfig.Tracer = opts.tracer
	}

	// Create the connection pool with timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.connectTimeout)
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

// Example usage:
//
// Basic usage with environment variables:
//   pool, err := postgresdb.NewFromEnv("DB_")
//
// With additional options:
//   pool, err := postgresdb.NewFromEnv("DB_",
//       postgresdb.WithLogger(myLogger),
//       postgresdb.WithMaxConns(50),
//       postgresdb.WithLogQueries(true),
//   )
//
// Test database:
//   pool, err := postgresdb.NewTestDB(connString,
//       postgresdb.WithLogger(testLogger),
//   )
//
// Default configuration with options:
//   pool, err := postgresdb.NewDefault(
//       postgresdb.WithDatabaseURL("postgres://..."),
//       postgresdb.WithTracer(customTracer),
//   )
