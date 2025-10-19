package pgxstores

// StoreCustomTemplate is the template for store.go (generated only if doesn't exist)
// This file uses embedding to selectively override generated SQL operations
const StoreCustomTemplate = `// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// You can override any SQL operation by defining it with the same signature.
// For example, to add custom logic to Create:
//
//   func (s *Store) Create(ctx context.Context, input {{.RepoPackage}}.{{.Create}}) ({{.RepoPackage}}.{{.Entity}}, error) {
//       // Your custom SQL or pre/post-processing
//       return s.GeneratedStore.Create(ctx, input)
//   }

package {{.PackageName}}

import (
	"github.com/jrazmi/envoker/sdk/logger"
	"github.com/jrazmi/envoker/infrastructure/postgresdb"
)

// ========================================
// STORE
// ========================================

// Store provides database access for {{.Entity}}.
// It embeds GeneratedStore to inherit all default SQL operations.
// You can override any method by defining it in this file with the same signature.
type Store struct {
	GeneratedStore
}

// NewStore creates a new {{.Entity}} store
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
// func (s *Store) Create(ctx context.Context, input {{.RepoPackage}}.{{.Create}}) ({{.RepoPackage}}.{{.Entity}}, error) {
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
// func (s *Store) GetActive{{.Entity}}Records(ctx context.Context) ([]{{.RepoPackage}}.{{.Entity}}, error) {
//     query := ` + "`SELECT * FROM {{.Schema}}.{{.Table}} WHERE status = 'active' ORDER BY created_at DESC`" + `
//
//     rows, err := s.pool.Query(ctx, query)
//     if err != nil {
//         return nil, postgresdb.HandlePgError(err)
//     }
//     defer rows.Close()
//
//     entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[{{.RepoPackage}}.{{.Entity}}])
//     if err != nil {
//         return nil, postgresdb.HandlePgError(err)
//     }
//
//     return entities, nil
// }
`
