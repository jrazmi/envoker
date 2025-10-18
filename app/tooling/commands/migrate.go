// app/tooling/commands/migrate.go
package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jrazmi/envoker/infrastructure/postgresdb"
)

// ErrHelp provides context that help was given.
var ErrHelp = errors.New("provided help")

// Migrate creates the schema in the database.
func Migrate(pool *pgxpool.Pool, log *slog.Logger) error {
	// Increase timeout for migrations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.InfoContext(ctx, "migration started", "step", "testing simple query")

	// Test a simple query first
	var result bool
	err := pool.QueryRow(ctx, "SELECT true").Scan(&result)
	if err != nil {
		return fmt.Errorf("simple query failed: %w", err)
	}

	log.InfoContext(ctx, "simple query successful", "step", "checking database status")

	// Check database status using the postgresdb package
	if err := postgresdb.StatusCheck(ctx, pool); err != nil {
		return fmt.Errorf("database status check failed: %w", err)
	}

	log.InfoContext(ctx, "database status check successful", "step", "running migrations")

	// Run the actual migration
	if err := postgresdb.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	log.InfoContext(ctx, "migrations completed successfully")
	return nil
}

// // Seed runs the seed document to populate test data.
// func Seed(pool *pgxpool.Pool, log *slog.Logger) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
// 	defer cancel()

// 	log.InfoContext(ctx, "seeding started")

// 	if err := migrate.Seed(ctx, pool); err != nil {
// 		return fmt.Errorf("seed database: %w", err)
// 	}

// 	log.InfoContext(ctx, "seeding completed successfully")
// 	fmt.Println("seeding complete")
// 	return nil
// }
