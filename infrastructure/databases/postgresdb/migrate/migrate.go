package migrate

// import (
// 	"context"
// 	_ "embed"
// 	"fmt"

// 	"github.com/jrazmi/envoker/infrastructure/databases/postgresdb"

// 	"github.com/ardanlabs/darwin/v3"
// 	"github.com/ardanlabs/darwin/v3/dialects/postgres"
// 	"github.com/ardanlabs/darwin/v3/drivers/generic"
// 	"github.com/jackc/pgx/v5/stdlib"
// )

// var (
// 	//go:embed sql/migrate.sql
// 	migrateDoc string
// )

// // Migrate attempts to bring the database up to date with the migrations
// // defined in this package.
// func Migrate(ctx context.Context, pool *postgresdb.Pool) error {
// 	if err := postgresdb.StatusCheck(ctx, pool); err != nil {
// 		return fmt.Errorf("status check database %w", err)
// 	}

// 	// Get the underlying database/sql.DB from the pgx pool
// 	// This allows us to use darwin which expects database/sql interface
// 	sqlDB := stdlib.OpenDBFromPool(pool)
// 	defer sqlDB.Close()

// 	driver, err := generic.New(sqlDB, postgres.Dialect{})
// 	if err != nil {
// 		return fmt.Errorf("construct darwin driver: %w", err)
// 	}

// 	d := darwin.New(driver, darwin.ParseMigrations(migrateDoc))
// 	return d.Migrate()
// }
