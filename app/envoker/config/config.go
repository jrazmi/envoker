package config

import (
	"github.com/jrazmi/envoker/sdk/logger"
	"github.com/jrazmi/envoker/sdk/telemetry"
)

// site wide globals.
const (
	AdminRoute = "dash"
	ApiRoute   = "api"
)

type UseCases struct {
}

// Repositories represents the specific repostiories that this instance of envoker needs.
// Add any custom repositories here should you need to expand on the defaults.
type Repositories struct {
	// User *userrepo.Repository
}

// Envoker is the overall configuration for the envoker application.
// modify as needed for your use case.
type Envoker struct {
	Build  string
	Logger *logger.Logger

	// Repositories & Cases
	Repositories Repositories
	Telemetry    telemetry.Telemetry
	// Datastores
	// MYSQLDatastore *mysqldb.Datastore
	// SQLiteDatastore
}
