package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jrazmi/envoker/bridge/scaffolding/mid"
	"github.com/jrazmi/envoker/infrastructure/postgresdb"
	"github.com/jrazmi/envoker/infrastructure/web"
	"github.com/jrazmi/envoker/sdk/environment"
	"github.com/jrazmi/envoker/sdk/logger"
	"github.com/jrazmi/envoker/sdk/telemetry"
)

var build = "develop"
var appName = "ENVOKER"

type Repositories struct {
	// TaskRepository *tasksrepo.Repository
}

type APIConfig struct {
	Logger       *logger.Logger
	Repositories Repositories
}

// Create the API v1 route group
func setupAPIv1Routes(app *web.WebHandler, cfg APIConfig) *web.RouteGroup {
	// Create the base API v1 group
	api := app.Group("/api/v1")

	// tasksrepobridge.AddHttpRoutes(api, tasksrepobridge.Config{
	// 	Log:        cfg.Logger,
	// 	Repository: cfg.Repositories.TaskRepository,
	// })

	return api
}

func run(ctx context.Context, log *logger.Logger) error {
	// TELEMETRY
	// ==============================================================================

	telemetry := telemetry.NewTelemetry()
	log.InfoContext(ctx, "init", "service", "telemetry")

	// ==============================================================================

	// DATABASES
	// ==============================================================================

	pg, err := postgresdb.NewFromEnv(appName)
	if err != nil {
		return fmt.Errorf("configuring postgres support: %w", err)
	}
	defer func() {
		log.InfoContext(ctx, "shutdown", "status", "closing database connection")
		pg.Close()
	}()
	log.InfoContext(ctx, "init", "service", "postgres")

	// ==============================================================================

	// REPOSITORIES AND USE CASES
	// ==============================================================================

	repositories := Repositories{
		// TaskRepository: tasksrepo.NewRepository(log, taskspgxstore.NewStore(log, pg)),
	}

	// ==============================================================================

	// WEB APPLICATION / HANDLERS
	// ==============================================================================
	webHandler, err := web.NewWebHandlerFromEnv(
		appName,
		// web handler uses a language level logger for error logger - hence slog.Logger here
		web.WithLogging(log.Logger),
		web.WithTelemetry(telemetry),
		web.WithGlobalMiddleware(
			mid.Logger(log),
			mid.Errors(log),
			mid.Metrics(),
			mid.Panics(),
		),
	)
	if err != nil {
		return fmt.Errorf("web app: %v", err)
	}

	setupAPIv1Routes(webHandler, APIConfig{
		Logger:       log,
		Repositories: repositories,
	})

	// ==============================================================================

	// WEB SERVER
	// ==============================================================================
	httpServer, err := web.NewServerFromEnv(appName, web.WithHandler(webHandler))
	if err != nil {
		return fmt.Errorf("web server: %v", err)
	}
	log.InfoContext(ctx, "init", "service", "http server")
	serverErrors := make(chan error, 1)

	go func() {
		log.InfoContext(ctx, "startup", "status", "api router started", "host", httpServer.Config.Port)
		serverErrors <- httpServer.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.InfoContext(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.InfoContext(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, 30)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}

	}
	return nil
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
