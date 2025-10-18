// Package schema contains embedded migration files.
package schema

import "embed"

// MigrationsFS contains all SQL migration files from pgmigrations directory.
//
//go:embed pgmigrations/*.sql
var MigrationsFS embed.FS
