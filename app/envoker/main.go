package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/jrazmi/envoker/app/envoker/admin"
	"github.com/jrazmi/envoker/app/envoker/api"
	"github.com/jrazmi/envoker/app/envoker/config"
	"github.com/jrazmi/envoker/bridge/scaffolding/mid"
	"github.com/jrazmi/envoker/infrastructure/web"
	"github.com/jrazmi/envoker/sdk/logger"
	"github.com/jrazmi/envoker/sdk/telemetry"
)

var build = "develop"
var appName = "ENVOKER"

func main() {
	godotenv.Load()
	ctx := context.Background()

	var log *logger.Logger
	events := logger.Events{
		Error: func(ctx context.Context, r logger.Record) {
			log.Info(ctx, "******* SEND ALERT *******")
		},
	}
	var telemetry telemetry.Telemetry
	traceIDFn := func(ctx context.Context) string {
		return telemetry.GetTraceID(ctx)
	}
	log = logger.NewWithEvents(os.Stdout, logger.LevelDebug, appName, traceIDFn, events)

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "err", err)

		os.Exit(1)
	}

}

func run(ctx context.Context, log *logger.Logger) error {
	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// :*: START DATABASES :*:
	// pg, err := postgresdb.NewDatabaseFromEnv(appName)
	// if err != nil {
	// 	return fmt.Errorf("configuring postgres support: %w", err)
	// }
	// defer func() {
	// 	log.Info(ctx, "shutdown", "status", "closing database connection")
	// 	pg.Close()
	// }()
	// END DATABASES //

	// REPOSITORIES //
	log.Info(ctx, "startup", "status", "initializing repository support")
	// repositories := repositories.NewPostgresRepositories(log, pg)
	// END REPOSITORIES //

	webCfg, err := web.LoadServerConfig(appName)
	if err != nil {
		return fmt.Errorf("webserver: %w", err)
	}

	siteCfg := config.Envoker{
		Build:        build,
		Logger:       log,
		Repositories: config.Repositories{
			// User: repositories.UserRepository,
		},
	}
	server := web.NewWebServer(webCfg, webHandler(siteCfg), logger.NewStdLogger(log, logger.LevelError))

	serverErrors := make(chan error, 1)
	go func() {
		log.Info(ctx, "startup", "status", "api router started", "host", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, 30)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			server.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}

	}

	return nil
}

func webHandler(cfg config.Envoker) http.Handler {

	// INITIALIZATION
	app := web.NewApp(cfg.Logger, cfg.Telemetry)

	// GLOBAL MIDDLEWARE
	app.AddGlobalMiddleware("cors", mid.PublicCORS())         // Default public CORS
	app.AddGlobalMiddleware("logger", mid.Logger(cfg.Logger)) // Request logging
	app.AddGlobalMiddleware("errors", mid.Errors(cfg.Logger)) // Error handling
	app.AddGlobalMiddleware("metrics", mid.Metrics())         // Metrics collection
	app.AddGlobalMiddleware("panics", mid.Panics())           // Panic recovery

	// API
	api.AddHandlers(app)

	// Admin section - different CORS, skip caching, replace rate limiting
	// app.AddPathGroupMiddlewareWithOverrides("/admin",
	// 	// []web.MiddlewareOverride{
	// 	// 	{Action: web.MiddlewareSkip, Target: "cache"},
	// 	// 	// ...
	// 	// },
	// 	// mid.AdminAuth(),
	// 	// mid.AdminLogging(),
	// )

	// ADMIN
	admin.AddHandlers(app)

	// // WEBHOOKS
	// cmsadmin.AddHandlers(app, cmsadmin.Config{URLPath: AdminRoute, Repositories: cfg.Repositories})

	return app
}
