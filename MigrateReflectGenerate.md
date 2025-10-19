## Overview

The workflow has three stages:

1. **Migrate**: Apply database migrations to PostgreSQL
2. **Reflect**: Introspect the database schema and output JSON
3. **Generate**: Create Go code from the reflected schema JSON

```bash
# 🚀 Quick Commands (most commonly used)

# After creating a migration, regenerate a single table
make regen TABLE=api_keys

# After creating migrations, regenerate all tables
make regen-all

# 🔧 Individual Steps (for manual control)

make migrate              # Apply migrations only
make db-reflect           # Reflect schema to JSON only
make generate-all         # Generate all tables only
make generate-table TABLE=api_keys  # Generate single table only
```

## The Three Layers

For each database table, the generator creates three layers using an **embedding pattern** that cleanly separates generated code from customizations.

### Key Concept: Embedding Pattern

All layers follow the same pattern:

- **`generated.go`** - Contains ALL generated code (models, methods, handlers)
- **Custom files** - Use type aliases and struct embedding to extend generated code
- **Override methods** - Simply define a method on your custom struct to override the generated version

### 1. Repository Layer (`core/repositories/{table}repo/`)

**Business logic layer** - Contains domain models and repository interface

**Files:**

- 🔄 `generated.go` - **ALWAYS REGENERATED** - ALL generated code (models, CRUD methods, FOP)
- ✅ `model.go` - **NEVER OVERWRITTEN** - Type aliases for models (generated once)
- ✅ `fop.go` - **NEVER OVERWRITTEN** - Type alias for filter (generated once)
- ✅ `repository.go` - **NEVER OVERWRITTEN** - Repository struct with embedding (generated once)

**How it works:**

```go
// generated.go (always regenerated)
type GeneratedApiKey struct {
    ApiKeyId string
    Name     string
    // ... fields
}

type GeneratedRepository struct {
    log    *logger.Logger
    storer Storer
}

func (r *GeneratedRepository) Create(ctx context.Context, input GeneratedCreateApiKey) (GeneratedApiKey, error) {
    // Default implementation
}

// model.go (never overwritten - customize here)
type ApiKey = GeneratedApiKey           // Type alias (zero cost)
type CreateApiKey = GeneratedCreateApiKey

// repository.go (never overwritten - customize here)
type Repository struct {
    GeneratedRepository  // Embedding - inherits all methods
}

// Override the Create method to add custom logic
func (r *Repository) Create(ctx context.Context, input CreateApiKey) (ApiKey, error) {
    // Custom business logic here
    // For example, generate an API key value
    input.KeyValue = generateSecureKey()

    // Call the generated method if needed
    return r.GeneratedRepository.Create(ctx, input)
}
```

### 2. Store Layer (`core/repositories/{table}repo/stores/{table}pgxstore/`)

**Data access layer** - PostgreSQL implementation using PGX

**Files:**

- 🔄 `generated.go` - **ALWAYS REGENERATED** - ALL generated code (SQL queries, FOP)
- ✅ `store.go` - **NEVER OVERWRITTEN** - Store struct with embedding (generated once)

**How it works:**

```go
// generated.go (always regenerated)
type GeneratedStore struct {
    log  *logger.Logger
    pool *postgresdb.Pool
}

func (s *GeneratedStore) Create(ctx context.Context, input apikeysrepo.CreateApiKey) (apikeysrepo.ApiKey, error) {
    // Default SQL implementation
}

// store.go (never overwritten - customize here)
type Store struct {
    GeneratedStore  // Embedding - inherits all SQL methods
}

// Override the Create method for custom SQL
func (s *Store) Create(ctx context.Context, input apikeysrepo.CreateApiKey) (apikeysrepo.ApiKey, error) {
    // Custom SQL query here
    // Or call the generated method with extra logic
    return s.GeneratedStore.Create(ctx, input)
}
```

### 3. Bridge Layer (`bridge/repositories/{table}repobridge/`)

**HTTP/API layer** - REST endpoints and request/response handling

**Files:**

- 🔄 `generated.go` - **ALWAYS REGENERATED** - ALL generated code (models, handlers, marshaling, FOP)
- ✅ `model.go` - **NEVER OVERWRITTEN** - Type aliases for bridge models (generated once)
- ✅ `bridge.go` - **NEVER OVERWRITTEN** - Bridge struct with embedding (generated once)
- ✅ `http.go` - **NEVER OVERWRITTEN** - Route registration with auth/middleware (generated once)

**How it works:**

```go
// generated.go (always regenerated)
type GeneratedApiKey struct {
    ApiKeyId string `json:"api_key_id"`
    Name     string `json:"name"`
}

type GeneratedBridge struct {
    apiKeyRepository ApiKeyRepository
}

func (b *GeneratedBridge) httpCreate(ctx context.Context, r *http.Request) web.Encoder {
    // Default HTTP handler implementation
}

// SUGGESTED ROUTES FOR http.go
// ============================================================================
// Copy these routes to http.go's AddHttpRoutes function.
// If new foreign keys are added by migrations, new routes will appear here.
//
//	// Standard CRUD routes
//	group.GET("/api-keys", b.httpList)
//	group.GET("/api-keys/{api_key_id}", b.httpGetByID)
//	group.POST("/api-keys", b.httpCreate)
// ============================================================================

// model.go (never overwritten - customize here)
type ApiKey = GeneratedApiKey  // Type alias

// bridge.go (never overwritten - customize here)
type bridge struct {
    GeneratedBridge  // Embedding - inherits all HTTP handlers
}

// Override the httpCreate handler for custom logic
func (b *bridge) httpCreate(ctx context.Context, r *http.Request) web.Encoder {
    // Custom HTTP handling here
    // Add custom validation, logging, etc.

    // Call generated handler if needed
    return b.GeneratedBridge.httpCreate(ctx, r)
}

// http.go (never overwritten - route registration)
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
    b := newBridge(cfg.Repository)

    // Standard CRUD routes
    group.GET("/api-keys", b.httpList)
    group.POST("/api-keys", b.httpCreate, cfg.Middleware.RequireAuth()...)
}
```

## Type Aliases vs Struct Embedding for Models

By default, custom files use **type aliases** for models:

```go
// Type alias (default) - zero cost, no overhead
type ApiKey = GeneratedApiKey
```

**When to switch to struct embedding:**

If you need to add custom fields or methods to a model, change from type alias to struct embedding:

```go
// Change from:
type ApiKey = GeneratedApiKey

// To:
type ApiKey struct {
    GeneratedApiKey
    CustomField string `json:"custom_field"`
}

// Now you can add custom methods
func (a ApiKey) IsExpired() bool {
    return time.Now().After(a.ExpiresAt)
}
```

**Important:** Once you switch to struct embedding, JSON marshaling will work correctly because the embedded struct's fields are promoted.

## 🚨 CRITICAL: Route Updates and Security

### First Generation

When you generate a table for the **first time**, `http.go` is created with **ALL routes active**:

```go
// http.go (generated once on first run)
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
    b := newBridge(cfg.Repository)

    // Standard CRUD routes - ALL ACTIVE
    group.GET("/api-keys", b.httpList)
    group.GET("/api-keys/{api_key_id}", b.httpGetByID)
    group.POST("/api-keys", b.httpCreate)
    group.PUT("/api-keys/{api_key_id}", b.httpUpdate)
    group.DELETE("/api-keys/{api_key_id}", b.httpDelete)

    // Foreign key routes - ALL ACTIVE
    group.GET("/applications/{application_id}/api-keys", b.httpListByApplicationId)
}
```

### ⚠️ SECURITY WARNING ⚠️

**YOU MUST REVIEW AND SECURE THESE ROUTES BEFORE DEPLOYING!**

1. **Comment out routes you don't want public**
2. **Add authentication middleware to protected routes**
3. **Add authorization checks for sensitive operations**

Example of properly secured routes:

```go
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
    b := newBridge(cfg.Repository)

    // Public route - no auth needed
    group.GET("/api-keys", b.httpList)

    // Protected routes - require authentication
    authenticated := []web.Middleware{cfg.Middleware.RequireAuth()}

    group.GET("/api-keys/{api_key_id}", b.httpGetByID, authenticated...)
    group.POST("/api-keys", b.httpCreate, authenticated...)
    group.PUT("/api-keys/{api_key_id}", b.httpUpdate, authenticated...)
    group.DELETE("/api-keys/{api_key_id}", b.httpDelete, authenticated...)

    // Admin-only route
    adminOnly := []web.Middleware{cfg.Middleware.RequireAdmin()}
    group.DELETE("/api-keys/{api_key_id}", b.httpDelete, adminOnly...)
}
```

### Subsequent Generations (Re-runs)

When you re-run the generator (e.g., after adding a column or foreign key):

- ✅ `http.go` is **NEVER OVERWRITTEN** - Your auth/middleware is safe
- 🔄 `generated.go` is **REGENERATED** with updated handler methods
- 📝 New routes appear as **SUGGESTIONS** in comments at the top of `generated.go`

```go
// generated.go (regenerated on every run)
// ============================================================================
// SUGGESTED ROUTES FOR http.go
// ============================================================================
// Copy these routes to http.go's AddHttpRoutes function.
// If new foreign keys are added by migrations, new routes will appear here.
//
//	// Standard CRUD routes
//	group.GET("/api-keys", b.httpList)
//	group.GET("/api-keys/{api_key_id}", b.httpGetByID)
//	group.POST("/api-keys", b.httpCreate)
//	group.PUT("/api-keys/{api_key_id}", b.httpUpdate)
//	group.DELETE("/api-keys/{api_key_id}", b.httpDelete)
//
//	// Foreign key routes
//	group.GET("/users/{created_by}/api-keys", b.httpListByCreatedBy)
// ============================================================================
```

**Action Required:**

1. Check `generated.go` for new suggested routes
2. Copy routes you want to expose to `http.go`
3. Add appropriate authentication/authorization middleware
4. Test security before deploying

## Overriding Generated Methods

### Repository Layer Example

```go
// repository.go
package apikeysrepo

type Repository struct {
    GeneratedRepository
}

// Override Create to add API key generation logic
func (r *Repository) Create(ctx context.Context, input CreateApiKey) (ApiKey, error) {
    // Generate a secure API key
    input.KeyValue = generateSecureAPIKey()
    input.KeyHash = hashAPIKey(input.KeyValue)

    // Call the generated Create method
    return r.GeneratedRepository.Create(ctx, input)
}

// Add a completely new method
func (r *Repository) RegenerateKey(ctx context.Context, apiKeyId string) (ApiKey, error) {
    // Custom business logic
    existing, err := r.Get(ctx, apiKeyId)
    if err != nil {
        return ApiKey{}, err
    }

    newKey := generateSecureAPIKey()
    return r.Update(ctx, apiKeyId, UpdateApiKey{
        KeyValue: &newKey,
    })
}
```

### Bridge Layer Example

```go
// bridge.go
package apikeysrepobridge

type bridge struct {
    GeneratedBridge
}

// Override httpCreate to add custom validation
func (b *bridge) httpCreate(ctx context.Context, r *http.Request) web.Encoder {
    // Decode input
    var input CreateApiKeyInput
    if err := input.Decode(/* read request body */); err != nil {
        return errs.Newf(errs.InvalidArgument, "invalid input: %v", err)
    }

    // Custom validation
    if len(input.Name) < 3 {
        return errs.Newf(errs.InvalidArgument, "name must be at least 3 characters")
    }

    // Call the generated handler (which will call repository)
    return b.GeneratedBridge.httpCreate(ctx, r)
}

// Add a completely new handler
func (b *bridge) httpRegenerateKey(ctx context.Context, r *http.Request) web.Encoder {
    apiKeyId := web.Param(r, "api_key_id")

    result, err := b.apiKeyRepository.RegenerateKey(ctx, apiKeyId)
    if err != nil {
        return errs.Newf(errs.Internal, "regenerate key: %v", err)
    }

    return MarshalToBridge(result)
}
```

Then register the custom route in `http.go`:

```go
// http.go
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
    b := newBridge(cfg.Repository)

    // Standard routes...

    // Custom route for key regeneration
    authenticated := []web.Middleware{cfg.Middleware.RequireAuth()}
    group.POST("/api-keys/{api_key_id}/regenerate", b.httpRegenerateKey, authenticated...)
}
```

## File Protection Summary

### ✅ Files That Are NEVER Overwritten

These files are safe to customize and **generated only once**:

- `model.go` - Type aliases for models
- `fop.go` - Type alias for filter
- `repository.go` - Repository struct with embedding
- `store.go` - Store struct with embedding
- `bridge.go` - Bridge struct with embedding
- `http.go` - Route registration with auth/middleware

### 🔄 Files That Are ALWAYS Regenerated

These files should **NOT** be edited manually (changes will be lost):

- `generated.go` - ALL generated code in each layer

### 🛡️ Force Flag Behavior

```bash
# Without -force: Fails if generated.go exists
make generate-table TABLE=users

# With -force: Overwrites generated.go
make generate-table TABLE=users FORCE=-force
```

**Note:** Custom files (`model.go`, `repository.go`, `bridge.go`, `http.go`) are **NEVER** overwritten, even with `-force`.

## Adding a New Table

### Step 1: Create Migration

```bash
# Create migration file
touch schema/pgmigrations/004_add_my_table.sql
```

```sql
-- schema/pgmigrations/004_add_my_table.sql
CREATE TABLE my_table (
    my_table_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Step 2: Apply Migration

```bash
make migrate
```

### Step 3: Reflect and Generate

```bash
# Quick way - does migrate + reflect + generate in one command
make regen TABLE=my_table

# Or do it manually step by step
make migrate
make db-reflect  # Updates schema/reflector/output/public.json
make generate-table TABLE=my_table
```

This creates:

```
core/repositories/mytablerepo/
├── generated.go      # Generated code
├── model.go          # Type aliases
├── fop.go            # Filter alias
├── repository.go     # Custom repo
└── stores/mytablepgxstore/
    ├── generated.go  # Generated SQL
    └── store.go      # Custom store

bridge/repositories/mytablerepobridge/
├── generated.go      # Generated code
├── model.go          # Type aliases
├── bridge.go         # Custom bridge
└── http.go           # Route registration
```

### Step 5: Wire Up the API

Add the new repository to `app/api/main.go`:

```go
import (
    "github.com/gpsimpact/taskmaster/bridge/repositories/mytablerepobridge"
    "github.com/gpsimpact/taskmaster/core/repositories/mytablerepo"
    "github.com/gpsimpact/taskmaster/core/repositories/mytablerepo/stores/mytablepgxstore"
)

type Repositories struct {
    // ... existing repos
    MyTableRepository *mytablerepo.Repository
}

repositories := Repositories{
    // ... existing repos
    MyTableRepository: mytablerepo.NewRepository(log, mytablepgxstore.NewStore(log, pg)),
}

func setupAPIv1Routes(app *web.WebHandler, cfg APIConfig) *web.RouteGroup {
    api := app.Group("/api/v1")

    // ... existing routes

    mytablerepobridge.AddHttpRoutes(api, mytablerepobridge.Config{
        Log:        cfg.Logger,
        Repository: cfg.Repositories.MyTableRepository,
    })

    return api
}
```

### Step 6: 🚨 SECURE THE ROUTES 🚨

Open `bridge/repositories/mytablerepobridge/http.go` and:

1. Comment out routes you don't want to expose
2. Add authentication middleware to protected routes
3. Add authorization checks for sensitive operations
4. Test thoroughly

## Modifying an Existing Table

### Step 1: Create Migration

```bash
# Example: Add a new column
touch schema/pgmigrations/005_add_column_to_users.sql
```

```sql
-- schema/pgmigrations/005_add_column_to_users.sql
ALTER TABLE users
ADD COLUMN phone_number TEXT;
```

### Step 2: Apply Migration

```bash
make migrate
```

### Step 3: Reflect and Regenerate

```bash
# Quick way
make regen TABLE=users

# Or manually
make migrate
make db-reflect
make generate-table TABLE=users
```

**What happens:**

- `generated.go` is regenerated with the new `PhoneNumber` field
- `model.go`, `repository.go`, `bridge.go`, `http.go` are **NOT** touched
- Your custom methods and route configurations remain intact

### Step 4: Check for New Routes

If you added a foreign key column:

1. Open `bridge/repositories/usersrepobridge/generated.go`
2. Look at the **SUGGESTED ROUTES** section near the top
3. Copy any new routes you want to `http.go`
4. Add appropriate authentication/authorization
5. Test security

## Example Workflow: Adding a Foreign Key

Let's say we add a `created_by` column to `api_keys` that references `users`:

### Step 1: Migration

```sql
-- schema/pgmigrations/003_add_created_by_to_api_keys.sql
ALTER TABLE api_keys
ADD COLUMN created_by UUID;

ALTER TABLE api_keys
ADD CONSTRAINT api_keys_created_by_fkey
FOREIGN KEY (created_by)
REFERENCES users(user_id)
ON DELETE SET NULL;
```

### Step 2: Migrate → Reflect → Generate

```bash
# Quick way
make regen TABLE=api_keys

# Or manually
make migrate
make db-reflect
make generate-table TABLE=api_keys
```

### Step 3: New Handler Method Created

The generator detects the foreign key and creates a new method in `generated.go`:

```go
// generated.go (regenerated)
func (b *GeneratedBridge) httpListByCreatedBy(ctx context.Context, r *http.Request) web.Encoder {
    // Handler implementation for listing API keys by creator
}
```

### Step 4: New Route Suggested

Check the top of `generated.go`:

```go
// ============================================================================
// SUGGESTED ROUTES FOR http.go
// ============================================================================
// Copy these routes to http.go's AddHttpRoutes function.
// If new foreign keys are added by migrations, new routes will appear here.
//
//	// Foreign key routes
//	group.GET("/users/{created_by}/api-keys", b.httpListByCreatedBy)
// ============================================================================
```

### Step 5: Add Route to http.go (WITH AUTH!)

```go
// http.go (you edit this manually)
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
    b := newBridge(cfg.Repository)

    // Existing routes...

    // New route - require authentication to see who created which API keys
    authenticated := []web.Middleware{cfg.Middleware.RequireAuth()}
    group.GET("/users/{created_by}/api-keys", b.httpListByCreatedBy, authenticated...)
}
```

## Common Commands

```bash
# 🚀 Most Common Workflows

# After creating a migration for a single table
make regen TABLE=api_keys

# After creating migrations for multiple tables
make regen-all

# Complete database reset (drops schema, runs migrations, regenerates all)
make db-code-full-reset

# 🔧 Individual Steps (for manual control)

# Just run migrations
make migrate

# Just reflect schema to JSON
make db-reflect

# Just generate all tables from existing JSON
make generate-all

# Just generate one table from existing JSON
make generate-table TABLE=users

# 🗄️ Database Management

# Reset database schema (keeps Docker running)
make db-reset-local

# Connect to database
make dev-psql

# Start database Docker container
make dev-data-up

# Stop database Docker container
make dev-data-down

# 📚 Get Help

# Show all generator commands
make generate-help
```

## Directory Structure

```
taskmaster/
├── schema/
│   ├── pgmigrations/           # SQL migration files
│   │   ├── 001_initial_schema.sql
│   │   ├── 002_add_users_table.sql
│   │   └── 003_add_created_by_to_api_keys.sql
│   └── reflector/output/       # Reflected schema (generated)
│       └── public.json
│
├── core/repositories/          # Repository layer (business logic)
│   ├── usersrepo/
│   │   ├── generated.go       # 🔄 ALWAYS REGENERATED
│   │   ├── model.go           # ✅ NEVER OVERWRITTEN
│   │   ├── fop.go             # ✅ NEVER OVERWRITTEN
│   │   ├── repository.go      # ✅ NEVER OVERWRITTEN
│   │   └── stores/
│   │       └── userspgxstore/
│   │           ├── generated.go  # 🔄 ALWAYS REGENERATED
│   │           └── store.go      # ✅ NEVER OVERWRITTEN
│   └── apikeysrepo/
│       └── ...
│
└── bridge/repositories/        # Bridge layer (HTTP/API)
    ├── usersrepobridge/
    │   ├── generated.go       # 🔄 ALWAYS REGENERATED - Models, handlers, marshaling
    │   ├── model.go           # ✅ NEVER OVERWRITTEN - Type aliases
    │   ├── bridge.go          # ✅ NEVER OVERWRITTEN - Custom handlers
    │   └── http.go            # ✅ NEVER OVERWRITTEN - Routes with auth
    └── apikeysrepobridge/
        └── ...
```

## Best Practices

### ✅ DO:

- Review `generated.go` for suggested routes after regenerating
- Add authentication/authorization middleware to all routes in `http.go`
- Use the `-force` flag when you want to regenerate after schema changes
- Keep custom business logic in `repository.go` and `bridge.go`
- Override methods by defining them on your custom struct
- Use type aliases for models by default (zero cost)
- Switch to struct embedding when you need to add custom fields
- Test route security before deploying
- Comment out routes you don't want to expose

### ❌ DON'T:

- Edit `generated.go` files (changes will be overwritten)
- Copy suggested routes to `http.go` without adding auth middleware
- Expose all CRUD operations publicly
- Skip security review of generated routes
- Assume generated routes are production-ready
- Use struct embedding for models unless you need custom fields

## Advanced: Adding Custom Fields to Models

If you need to add custom fields to a generated model:

```go
// model.go - Change from type alias to struct embedding
package apikeysrepo

// Change from:
// type ApiKey = GeneratedApiKey

// To:
type ApiKey struct {
    GeneratedApiKey

    // Custom fields
    IsExpired bool `json:"is_expired"`
}

// Add custom methods
func (a ApiKey) CalculateExpired() bool {
    return time.Now().After(a.ExpiresAt)
}

// You may need custom marshaling if you compute fields
func (a ApiKey) MarshalJSON() ([]byte, error) {
    type Alias ApiKey
    return json.Marshal(&struct {
        *Alias
        IsExpired bool `json:"is_expired"`
    }{
        Alias:     (*Alias)(&a),
        IsExpired: a.CalculateExpired(),
    })
}
```

## Security Checklist

Before deploying, verify:

- [ ] All routes in `http.go` have appropriate authentication
- [ ] Sensitive operations (create/update/delete) are protected
- [ ] Public routes are intentionally public
- [ ] Authorization checks verify users can only access their own data
- [ ] Foreign key routes don't leak sensitive information
- [ ] Rate limiting is configured for public endpoints
- [ ] Input validation is in place
- [ ] Audit logging is enabled for sensitive operations
- [ ] Custom method overrides maintain security constraints
- [ ] New foreign key routes are reviewed and secured

## Troubleshooting

### "File already exists" error

The `-force` flag is automatically applied by the Makefile commands. If you see this error, just run:

```bash
make regen TABLE=users
```

This overwrites `generated.go` only - your custom files are safe.

### New routes not appearing

Use the `regen` command which does all steps:

```bash
make regen TABLE=users
```

Or do it manually:

1. Check that migration was applied: `make migrate`
2. Reflect the schema: `make db-reflect`
3. Regenerate: `make generate-table TABLE=users`
4. Look in `generated.go` for suggested routes

### Method override not working

Make sure you're using the exact same signature:

```go
// ✅ Correct - exact signature match
func (r *Repository) Create(ctx context.Context, input CreateApiKey) (ApiKey, error) {
    // Your implementation
}

// ❌ Wrong - different types
func (r *Repository) Create(ctx context.Context, input GeneratedCreateApiKey) (GeneratedApiKey, error) {
    // Won't override because CreateApiKey != GeneratedCreateApiKey (even though they're aliases)
}
```

### Routes lost after regeneration

You edited `generated.go` instead of `http.go`. Routes belong in `http.go` (never overwritten).
Check git history to recover your routes.

### Custom fields not appearing in JSON

If you changed from type alias to struct embedding, make sure the embedded struct is not a pointer and fields are exported:

```go
// ✅ Correct
type ApiKey struct {
    GeneratedApiKey      // No pointer - fields are promoted
    CustomField string   // Exported field
}

// ❌ Wrong
type ApiKey struct {
    *GeneratedApiKey     // Pointer - fields won't promote correctly
    customField string   // Unexported - won't appear in JSON
}
```

---

## Quick Reference Card

### After Creating a Migration

```bash
# Single table
make regen TABLE=api_keys

# All tables
make regen-all
```

### File Structure per Table

```
Repository Layer (core/repositories/apikeysrepo/):
  generated.go      🔄 ALWAYS regenerated
  model.go          ✅ NEVER overwritten (type aliases)
  fop.go            ✅ NEVER overwritten (filter alias)
  repository.go     ✅ NEVER overwritten (your methods here)
  stores/apikeyspgxstore/:
    generated.go    🔄 ALWAYS regenerated
    store.go        ✅ NEVER overwritten (custom SQL here)

Bridge Layer (bridge/repositories/apikeysrepobridge/):
  generated.go      🔄 ALWAYS regenerated (check for suggested routes!)
  model.go          ✅ NEVER overwritten (type aliases)
  bridge.go         ✅ NEVER overwritten (override handlers here)
  http.go           ✅ NEVER overwritten (register routes with auth)
```

### Override a Method

```go
// repository.go
func (r *Repository) Create(ctx context.Context, input CreateApiKey) (ApiKey, error) {
    // Your custom logic
    return r.GeneratedRepository.Create(ctx, input)
}

// bridge.go
func (b *bridge) httpCreate(ctx context.Context, r *http.Request) web.Encoder {
    // Your custom validation
    return b.GeneratedBridge.httpCreate(ctx, r)
}
```

### Add Custom Fields to Model

```go
// model.go - change from:
type ApiKey = GeneratedApiKey

// to:
type ApiKey struct {
    GeneratedApiKey
    CustomField string `json:"custom_field"`
}
```

---

## 🎯 Remember: Security First!

The generator creates **permissive routes by default** to get you started quickly. It's **YOUR RESPONSIBILITY** to:

1. Review all generated routes
2. Add authentication where needed
3. Add authorization where needed
4. Comment out routes you don't want
5. Test security thoroughly
6. Review custom method overrides for security implications

**Generated code is a starting point, not production-ready code.**
