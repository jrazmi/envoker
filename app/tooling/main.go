package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jrazmi/envoker/app/tooling/commands"
	"github.com/jrazmi/envoker/infrastructure/postgresdb"
	"github.com/jrazmi/envoker/sdk/environment"
	"github.com/jrazmi/envoker/sdk/logger"
)

var build = "develop"
var appName = "TOOLING"

func processCommands(ctx context.Context, log *logger.Logger, command string, args []string, pg *pgxpool.Pool) error {
	switch command {
	case "migrate":
		log.InfoContext(ctx, "running migration")
		if err := postgresdb.Migrate(ctx, pg); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		log.InfoContext(ctx, "migration completed successfully")
		return nil

	case "reflect-schema":
		log.InfoContext(ctx, "running schema reflection")
		if err := commands.ReflectSchema(ctx, log.Logger, args, pg); err != nil {
			return fmt.Errorf("reflect schema failed: %w", err)
		}
		return nil

	default:
		printHelp()
		return nil
	}

}
func printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  migrate        - create the schema in the database")
	fmt.Println("  reflect-schema - reflect current database schema to JSON/SQL files")
	fmt.Println()
	fmt.Println("Use 'go run app/tooling/main.go <command> --help' for command-specific help.")
}

func run(ctx context.Context, log *logger.Logger) error {
	log.InfoContext(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))
	// DATA INFRASTRUCTURE
	// ==============================================================================
	// Parse command from arguments
	var command string
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	// Show help and exit early if requested
	if command == "help" || command == "--help" || command == "-h" {
		printHelp()
		return nil
	}
	pg, err := postgresdb.NewFromEnv(appName, postgresdb.WithTracer(postgresdb.NewLoggingQueryTracer(log.Logger)))
	if err != nil {
		return fmt.Errorf("configuring postgres support: %w", err)
	}
	defer func() {
		log.InfoContext(ctx, "shutdown", "status", "closing database connection")
		pg.Close()
	}()
	log.InfoContext(ctx, "init", "service", "postgres")

	// Setup signal handling for graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Process commands in a goroutine to allow for graceful shutdown
	done := make(chan error, 1)
	go func() {
		// Pass remaining args (everything after the command)
		args := []string{}
		if len(os.Args) > 2 {
			args = os.Args[2:]
		}
		done <- processCommands(ctx, log, command, args, pg)
	}()

	// Handle shutdown
	select {
	case err := <-done:
		return err

	case sig := <-shutdown:
		log.InfoContext(ctx, "shutdown", "status", "shutdown started", "signal", sig)

		// Give a short time for commands to complete
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Wait for command to complete or timeout
		select {
		case err := <-done:
			return err
		case <-shutdownCtx.Done():
			return fmt.Errorf("shutdown timeout: %w", shutdownCtx.Err())
		}
	}

}

func main() {
	environment.LoadEnv()

	log, err := logger.NewFromEnv(appName)
	if err != nil {
		fmt.Println("oh no we couldn't even get logging going.")
		os.Exit(1)
	}
	ctx := context.Background()

	if err = run(ctx, log); err != nil {
		log.ErrorContext(ctx, "startup", "err", err)
		os.Exit(1)
	}
}
