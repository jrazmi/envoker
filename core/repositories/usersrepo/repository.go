// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// This file uses:
// - Type aliases (type Foo = GeneratedFoo) for using generated types as-is
// - Struct embedding (type Foo struct { GeneratedFoo }) for extending generated types
// - Interface embedding (type Storer interface { GeneratedStorer }) for adding custom methods
// - Method overriding via embedding for custom business logic
//
// Examples:
//
// Use generated type as-is:
//   type User = GeneratedUser
//
// Extend a generated struct:
//   type UpdateUser struct {
//       GeneratedUpdateUser
//       CustomField string `json:"custom_field"`
//   }
//
// Override a repository method:
//   func (r *Repository) Create(ctx context.Context, input CreateUser) (User, error) {
//       // Your custom logic here
//       r.log.Info("custom create logic")
//       return r.GeneratedRepository.Create(ctx, input)
//   }

package usersrepo

import (
	"github.com/jrazmi/envoker/sdk/logger"
)

// ========================================
// STORER INTERFACE
// ========================================

// Storer defines the complete data storage interface for User.
// It embeds GeneratedStorer (from generated.go) which contains all auto-generated methods.
// Add your custom storage methods below the embedded interface.
type Storer interface {
	GeneratedStorer

	// Add custom store methods below this line.
	// These methods should be implemented in the store layer (e.g., stores/usersrepopgxstore/store.go)
	//
	// Example:
	// GetActiveUsers(ctx context.Context) ([]User, error)
	// FindByUserPrefix(ctx context.Context, prefix string) ([]User, error)
}

// ========================================
// REPOSITORY
// ========================================

// Repository provides access to user storage.
// It embeds GeneratedRepository to inherit all default CRUD operations.
// You can override any method by defining it in this file with the same signature.
type Repository struct {
	GeneratedRepository
}

// NewRepository creates a new User repository
func NewRepository(log *logger.Logger, storer Storer) *Repository {
	return &Repository{
		GeneratedRepository: GeneratedRepository{
			log:    log,
			storer: storer,
		},
	}
}

// ========================================
// CUSTOM METHODS & OVERRIDES
// ========================================

// Add custom repository methods or override generated methods below.
//
// To override a generated method (e.g., Create), define it with the same signature:
//
// func (r *Repository) Create(ctx context.Context, input CreateUser) (User, error) {
//     r.log.Info("creating user", "input", input)
//
//     // Add custom business logic here (validation, transformation, etc.)
//     // ...
//
//     // Option 1: Call the store layer directly
//     entity, err := r.storer.Create(ctx, input)
//     if err != nil {
//         r.log.Error("failed to create user", "error", err)
//         return User{}, fmt.Errorf("create user: %w", err)
//     }
//
//     // Option 2: Call the generated implementation
//     // entity, err := r.GeneratedRepository.Create(ctx, input)
//     // if err != nil {
//     //     return User{}, err
//     // }
//
//     r.log.Info("created user", "id", entity.UserId)
//     return entity, nil
// }
//
// To add a completely new method:
//
// func (r *Repository) ArchiveUser(ctx context.Context, userId string) error {
//     r.log.Info("archiving user", "userId", userId)
//     // Custom logic here
//     return nil
// }
