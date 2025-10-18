## Overview

The workflow has three stages:

1. **Migrate**: Apply database migrations to PostgreSQL
2. **Reflect**: Introspect the database schema and output JSON
3. **Generate**: Create Go code from the reflected schema JSON

```bash
# Full workflow
make migrate          # Apply migrations
make reflect          # Reflect schema to JSON
make generate         # Generate all code layers

# Or run individually per table
make migrate
make reflect-table TABLE=users
make generate-table TABLE=users
```

## The Three Layers

For each database table, the generator creates three layers:

### 1. Repository Layer (`core/repositories/{table}repo/`)

**Business logic layer** - Contains domain models and repository interface

**Files:**

- ✅ `repo.go` - **NEVER OVERWRITTEN** - Add custom business logic here
- 🔄 `repo_gen.go` - Regenerated - CRUD methods
- 🔄 `model_gen.go` - Regenerated - Domain models (structs)
- 🔄 `fop_gen.go` - Regenerated - Filter/Order/Page helpers

### 2. Store Layer (`core/repositories/{table}repo/stores/{table}pgxstore/`)

**Data access layer** - PostgreSQL implementation using PGX

**Files:**

- 🔄 `store_gen.go` - Regenerated - SQL queries and PGX implementation

### 3. Bridge Layer (`bridge/repositories/{table}repobridge/`)

**HTTP/API layer** - REST endpoints and request/response handling

**Files:**

- ✅ `bridge.go` - **NEVER OVERWRITTEN** - Add custom bridge logic here
- ✅ `http.go` - **NEVER OVERWRITTEN** - Route registration with auth/middleware
- 🔄 `http_gen.go` - Regenerated - HTTP handler methods
- 🔄 `model_gen.go` - Regenerated - API request/response models
- 🔄 `marshal_gen.go` - Regenerated - Conversion between layers
- 🔄 `fop_gen.go` - Regenerated - Query parameter parsing

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
- 🔄 `http_gen.go` is **REGENERATED** with updated handler methods
- 📝 New routes appear as **SUGGESTIONS** in comments at the top of `http_gen.go`

```go
// http_gen.go (regenerated on every run)
// ============================================================================
// SUGGESTED ROUTES FOR http.go
// ============================================================================
// Copy the routes you need to http.go's AddHttpRoutes function:
//
//  //// Standard CRUD routes
//  // group.GET("/api-keys", b.httpList)
//  // group.GET("/api-keys/{api_key_id}", b.httpGetByID)
//  // group.POST("/api-keys", b.httpCreate)
//  // group.PUT("/api-keys/{api_key_id}", b.httpUpdate)
//  // group.DELETE("/api-keys/{api_key_id}", b.httpDelete)
//
//  //// Foreign key route: httpListByCreatedBy
//  // group.GET("/users/{created_by}/api-keys", b.httpListByCreatedBy)
// ============================================================================
```

**Action Required:**

1. Check `http_gen.go` for new suggested routes
2. Copy routes you want to expose to `http.go`
3. Add appropriate authentication/authorization middleware
4. Test security before deploying

## File Protection Summary

### ✅ Files That Are NEVER Overwritten

These files are safe to customize:

- `repo.go` - Add custom repository methods
- `bridge.go` - Add custom bridge logic
- `http.go` - Add routes with auth/middleware

### 🔄 Files That Are ALWAYS Regenerated

These files should NOT be edited manually:

- `*_gen.go` - All generated files
- `model_gen.go` - Domain and API models
- `store_gen.go` - SQL queries

### 🛡️ Force Flag Behavior

```bash
# Without -force: Fails if *_gen.go files exist
make generate-table TABLE=users

# With -force: Overwrites all *_gen.go files
make generate-table TABLE=users FORCE=-force
```

**Note:** User-editable files (`repo.go`, `bridge.go`, `http.go`) are **NEVER** overwritten, even with `-force`.

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

### Step 3: Reflect Schema

```bash
# Reflect just the new table
make reflect-table TABLE=my_table

# Or reflect all tables
make reflect
```

This creates: `schema/json/my_table.json`

### Step 4: Generate Code

```bash
# Generate all layers for the new table
make generate-table TABLE=my_table

# Or generate for all tables
make generate
```

### Step 5: Wire Up the API

Add the new repository to `app/api/main.go`:

```go
import (
    "github.com/jrazmi/envoker/bridge/repositories/mytablerepobridge"
    "github.com/jrazmi/envoker/core/repositories/mytablerepo"
    "github.com/jrazmi/envoker/core/repositories/mytablerepo/stores/mytablepgxstore"
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
make reflect-table TABLE=users
make generate-table TABLE=users FORCE=-force
```

### Step 4: Check for New Routes

1. Open `bridge/repositories/usersrepobridge/http_gen.go`
2. Look at the **SUGGESTED ROUTES** section at the top
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
make migrate
make reflect-table TABLE=api_keys
make generate-table TABLE=api_keys FORCE=-force
```

### Step 3: New Handler Method Created

The generator detects the foreign key and creates a new method in `http_gen.go`:

```go
// http_gen.go (regenerated)
func (b *bridge) httpListByCreatedBy(ctx context.Context, r *http.Request) web.Encoder {
    // Handler implementation for listing API keys by creator
}
```

### Step 4: New Route Suggested

Check the top of `http_gen.go`:

```go
// ============================================================================
// SUGGESTED ROUTES FOR http.go
// ============================================================================
//  //// Foreign key route: httpListByCreatedBy
//  // group.GET("/users/{created_by}/api-keys", b.httpListByCreatedBy)
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
# Full workflow for all tables
make migrate && make reflect && make generate

# Work with a specific table
make reflect-table TABLE=users
make generate-table TABLE=users

# Force regenerate (overwrites *_gen.go files)
make generate-table TABLE=users FORCE=-force

# Reset and reseed database
make db-reset
psql $DATABASE_URL < seed.sql
```

## Directory Structure

```
taskmaster/
├── schema/
│   ├── pgmigrations/           # SQL migration files
│   │   ├── 001_initial_schema.sql
│   │   ├── 002_add_users_table.sql
│   │   └── 003_add_created_by_to_api_keys.sql
│   └── json/                   # Reflected schema (generated)
│       ├── users.json
│       └── api_keys.json
│
├── core/repositories/          # Repository layer (business logic)
│   ├── usersrepo/
│   │   ├── repo.go            # ✅ NEVER OVERWRITTEN
│   │   ├── repo_gen.go        # 🔄 Regenerated
│   │   ├── model_gen.go       # 🔄 Regenerated
│   │   ├── fop_gen.go         # 🔄 Regenerated
│   │   └── stores/
│   │       └── userspgxstore/
│   │           └── store_gen.go  # 🔄 Regenerated
│   └── apikeysrepo/
│       └── ...
│
└── bridge/repositories/        # Bridge layer (HTTP/API)
    ├── usersrepobridge/
    │   ├── bridge.go          # ✅ NEVER OVERWRITTEN
    │   ├── http.go            # ✅ NEVER OVERWRITTEN - Routes with auth
    │   ├── http_gen.go        # 🔄 Regenerated - Handler methods
    │   ├── model_gen.go       # 🔄 Regenerated
    │   ├── marshal_gen.go     # 🔄 Regenerated
    │   └── fop_gen.go         # 🔄 Regenerated
    └── apikeysrepobridge/
        └── ...
```

## Best Practices

### ✅ DO:

- Review `http_gen.go` for suggested routes after regenerating
- Add authentication/authorization middleware to all routes in `http.go`
- Use the `-force` flag when you want to regenerate after schema changes
- Keep custom business logic in `repo.go` and `bridge.go`
- Test route security before deploying
- Comment out routes you don't want to expose

### ❌ DON'T:

- Edit any `*_gen.go` files (changes will be overwritten)
- Copy suggested routes to `http.go` without adding auth middleware
- Expose all CRUD operations publicly
- Skip security review of generated routes
- Assume generated routes are production-ready

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

## Troubleshooting

### "File already exists" error

Use the `-force` flag to regenerate:

```bash
make generate-table TABLE=users FORCE=-force
```

### New routes not appearing

1. Check that migration was applied: `make migrate`
2. Reflect the schema: `make reflect-table TABLE=users`
3. Regenerate with force: `make generate-table TABLE=users FORCE=-force`
4. Look in `http_gen.go` for suggested routes

### Routes lost after regeneration

You edited `http_gen.go` instead of `http.go`. Routes belong in `http.go` (never overwritten).
Check git history to recover your routes.

---

## 🎯 Remember: Security First!

The generator creates **permissive routes by default** to get you started quickly. It's **YOUR RESPONSIBILITY** to:

1. Review all generated routes
2. Add authentication where needed
3. Add authorization where needed
4. Comment out routes you don't want
5. Test security thoroughly

**Generated code is a starting point, not production-ready code.**
