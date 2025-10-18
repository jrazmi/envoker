package postgresdb

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jrazmi/envoker/schema"
)

// Migrate runs all pending migrations from schema/pgmigrations/*.sql files.
// Migrations are applied in alphabetical order (use numeric prefixes: 001_xxx.sql, 002_xxx.sql).
// Already-applied migrations are tracked in the schema_migrations table.
// This is a forward-only migration system - no rollbacks.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if err := StatusCheck(ctx, pool); err != nil {
		return fmt.Errorf("status check database: %w", err)
	}

	fmt.Println("🚀 Running database migrations...")

	if err := runMigrations(ctx, pool, schema.MigrationsFS, "pgmigrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	fmt.Println("✨ Migrations complete!")
	return nil
}

// runMigrations is the internal migration runner
func runMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsFS embed.FS, migrationsDir string) error {
	// Ensure migrations tracking table exists
	if err := createMigrationsTable(ctx, pool); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Get list of migration files
	files, err := getMigrationFiles(migrationsFS, migrationsDir)
	if err != nil {
		return fmt.Errorf("get migration files: %w", err)
	}

	// Apply each migration if not already applied
	for _, file := range files {
		if err := applyMigration(ctx, pool, migrationsFS, filepath.Join(migrationsDir, file)); err != nil {
			return fmt.Errorf("apply migration %s: %w", file, err)
		}
	}

	return nil
}

// createMigrationsTable creates the tracking table if it doesn't exist
func createMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			checksum VARCHAR(64) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW(),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`
	_, err := pool.Exec(ctx, query)
	return err
}

// getMigrationFiles returns sorted list of .sql files from the migrations directory
func getMigrationFiles(migrationsFS embed.FS, migrationsDir string) ([]string, error) {
	var files []string

	err := fs.WalkDir(migrationsFS, migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".sql") {
			// Get just the filename, not the full path
			files = append(files, filepath.Base(path))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort alphabetically (001_xxx.sql comes before 002_xxx.sql)
	sort.Strings(files)
	return files, nil
}

// applyMigration applies a single migration if it hasn't been applied yet
func applyMigration(ctx context.Context, pool *pgxpool.Pool, migrationsFS embed.FS, filePath string) error {
	version := filepath.Base(filePath)

	// Read migration file
	content, err := fs.ReadFile(migrationsFS, filePath)
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	// Calculate checksum of migration content
	checksum := fmt.Sprintf("%x", sha256.Sum256(content))

	// Check if already applied and verify checksum
	var existingChecksum string
	err = pool.QueryRow(ctx, "SELECT checksum FROM schema_migrations WHERE version = $1", version).Scan(&existingChecksum)
	if err == nil {
		// Migration was already applied - verify checksum matches
		if existingChecksum != checksum {
			return fmt.Errorf("CHECKSUM MISMATCH: migration %s has been modified after being applied (expected: %s, got: %s)",
				version, existingChecksum, checksum)
		}
		fmt.Printf("  ⏭️  %s (already applied, checksum verified)\n", version)
		return nil
	}

	// Migration not applied yet (ignore "no rows" error)
	// Execute migration in a transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute the migration SQL
	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	// Record migration as applied with checksum
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)", version, checksum); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	fmt.Printf("  ✅ %s (applied, checksum: %.8s...)\n", version, checksum)
	return nil
}
