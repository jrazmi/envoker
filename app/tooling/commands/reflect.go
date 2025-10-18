// app/tooling/commands/reflect.go
package commands

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jrazmi/envoker/schema/reflector"
)

// ReflectSchema reflects the current database schema and generates JSON/SQL artifacts
// This creates a Postgres-specific store and injects it into the reflector
func ReflectSchema(ctx context.Context, log *slog.Logger, args []string, pool *pgxpool.Pool) error {
	fs := flag.NewFlagSet("reflect-schema", flag.ExitOnError)

	// Schema reflection flags
	schema := fs.String("schema", "public", "Schema to reflect (default: public)")
	outputDir := fs.String("output", "schema/reflector/output", "Output directory for generated files")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	log.InfoContext(ctx, "reflect-schema started",
		"schema", *schema,
		"output", *outputDir,
	)

	// Create the Postgres store (knows how to query PostgreSQL)
	store := reflector.NewPostgresStore(ctx, pool, pool.Config().ConnConfig.Database)

	// Create reflector with injected store
	ref := reflector.NewReflector(store)

	log.InfoContext(ctx, "reflecting schema", "database", store.GetDatabaseName(), "schema", *schema)

	// Perform reflection
	reflectedSchema, err := ref.Reflect(*schema)
	if err != nil {
		return fmt.Errorf("reflect schema: %w", err)
	}

	log.InfoContext(ctx, "discovered tables", "count", len(reflectedSchema.Tables))

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Write JSON output
	jsonPath := filepath.Join(*outputDir, *schema+".json")
	if err := reflector.WriteJSON(reflectedSchema, jsonPath); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	log.InfoContext(ctx, "generated JSON", "path", jsonPath)

	// Write SQL output
	sqlPath := filepath.Join(*outputDir, *schema+".sql")
	if err := reflector.WriteSQL(reflectedSchema, sqlPath); err != nil {
		return fmt.Errorf("write SQL: %w", err)
	}
	log.InfoContext(ctx, "generated SQL", "path", sqlPath)

	log.InfoContext(ctx, "reflect-schema completed successfully")
	return nil
}
