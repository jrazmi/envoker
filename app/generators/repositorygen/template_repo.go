package repositorygen

// RepoTemplate is the template for repo.go (generated only if doesn't exist)
const RepoTemplate = `// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// You can override any generated method by defining it here with the same signature.
// For example, to add custom business logic to Create:
//
//   func (r *Repository) Create(ctx context.Context, input {{.CreateStructName}}) ({{.EntityName}}, error) {
//       // Your custom logic here
//       return r.GeneratedRepository.Create(ctx, input) // Call default implementation if needed
//   }

package {{.PackageName}}

import (
	"github.com/jrazmi/envoker/sdk/logger"
)

// Storer defines the complete data storage interface for {{.EntityName}}
// It embeds GeneratedStorer (from repo_gen.go) which contains all auto-generated methods.
// Add your custom storage methods here as needed.
type Storer interface {
	GeneratedStorer

	// Add custom store methods below this line:
	// Example:
	// GetPendingTasksForWorker(ctx context.Context, workerID string) ([]{{.EntityName}}, error)
}

// Repository provides access to {{.EntityNameLower}} storage.
// It embeds GeneratedRepository to inherit all default CRUD operations.
// You can override any method by defining it in this file with the same signature.
type Repository struct {
	GeneratedRepository
}

// NewRepository creates a new {{.EntityName}} repository
func NewRepository(log *logger.Logger, storer Storer) *Repository {
	return &Repository{
		GeneratedRepository: GeneratedRepository{
			log:    log,
			storer: storer,
		},
	}
}

// Add custom repository methods or overrides below this line.
// To override a generated method, simply define it with the same signature:
//
// func (r *Repository) Create(ctx context.Context, input {{.CreateStructName}}) ({{.EntityName}}, error) {
//     // Custom business logic (e.g., validation, key generation, etc.)
//     entity, err := r.storer.Create(ctx, input)
//     if err != nil {
//         return {{.EntityName}}{}, fmt.Errorf("create {{.EntityNameLower}}: %w", err)
//     }
//     return entity, nil
// }
`
