// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// You can override any SQL operation by defining it with the same signature.
// For example, to add custom logic to Create:
//
//   func (s *Store) Create(ctx context.Context, input schemamigrationsrepo.CreateSchemaMigration) (schemamigrationsrepo.SchemaMigration, error) {
//       // Your custom SQL or pre/post-processing
//       return s.GeneratedStore.Create(ctx, input)
//   }

package schemamigrationspgxstore

import (
	"github.com/jrazmi/envoker/infrastructure/postgresdb"
	"github.com/jrazmi/envoker/sdk/logger"
)

// ========================================
// STORE
// ========================================

// Store provides database access for SchemaMigration.
// It embeds GeneratedStore to inherit all default SQL operations.
// You can override any method by defining it in this file with the same signature.
type Store struct {
	GeneratedStore
}

// NewStore creates a new SchemaMigration store
func NewStore(log *logger.Logger, pool *postgresdb.Pool) *Store {
	return &Store{
		GeneratedStore: GeneratedStore{
			log:  log,
			pool: pool,
		},
	}
}

// ========================================
// CUSTOM QUERIES
// ========================================

// Add custom SQL queries below.
//
// To override a generated method (e.g., Create), define it with the same signature:
//
// func (s *Store) Create(ctx context.Context, input schemamigrationsrepo.CreateSchemaMigration) (schemamigrationsrepo.SchemaMigration, error) {
//     s.log.Info("custom create logic")
//
//     // Option 1: Call the generated implementation
//     return s.GeneratedStore.Create(ctx, input)
//
//     // Option 2: Write completely custom SQL
//     // query := "INSERT INTO ... custom logic ..."
//     // ...
// }
//
// To add a completely new query:
//
// func (s *Store) GetActiveSchemaMigrationRecords(ctx context.Context) ([]schemamigrationsrepo.SchemaMigration, error) {
//     query := `SELECT * FROM public.schema_migrations WHERE status = 'active' ORDER BY created_at DESC`
//
//     rows, err := s.pool.Query(ctx, query)
//     if err != nil {
//         return nil, postgresdb.HandlePgError(err)
//     }
//     defer rows.Close()
//
//     entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[schemamigrationsrepo.SchemaMigration])
//     if err != nil {
//         return nil, postgresdb.HandlePgError(err)
//     }
//
//     return entities, nil
// }
