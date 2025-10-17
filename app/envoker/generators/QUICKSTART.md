# Quick Start Guide

Get from zero to a working API in 60 seconds! ⚡

## The 60-Second Tutorial

### Step 1: Create Your SQL Schema (10 seconds)

```bash
cat > schema/posts.sql << 'EOF'
CREATE TABLE public.posts (
    post_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    title varchar(200) NOT NULL,
    content text,
    published boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    FOREIGN KEY (user_id) REFERENCES public.users (user_id) ON DELETE CASCADE
);
EOF
```

### Step 2: Generate Everything (20 seconds)

```bash
# Option A: Using the CLI directly
go run app/generators/main.go generate -sql=schema/posts.sql -force

# Option B: Using the script
./scripts/generate.sh schema/posts.sql -force

# Option C: Using Make
make generate-sql SQL=schema/posts.sql
```

### Step 3: Check What Was Generated (30 seconds)

```bash
# Repository Layer (Domain Models)
ls -la core/repositories/postsrepo/
#   model_gen.go        - Post, CreatePost, UpdatePost, FilterPost
#   repository_gen.go   - Repository interface & implementation

# Store Layer (Database)
ls -la core/repositories/postsrepo/stores/postspgxstore/
#   store_gen.go        - pgx v5 CRUD implementation

# Bridge Layer (REST API)
ls -la bridge/repositories/postsrepobridge/
#   bridge_gen.go       - Bridge struct
#   http_gen.go         - REST handlers (GET, POST, PUT, DELETE)
#   model_gen.go        - API models (camelCase JSON)
#   marshal_gen.go      - Conversion functions
#   fop_gen.go          - Query parsers
```

## What You Just Got

### 🎯 REST Endpoints (Auto-Generated)

```http
GET    /posts               # List with pagination
GET    /posts/{post_id}     # Get by ID
POST   /posts               # Create
PUT    /posts/{post_id}     # Update
DELETE /posts/{post_id}     # Delete

# Foreign key relationship
GET    /users/{user_id}/posts  # List posts by user
```

### 📦 Repository Layer

```go
// Domain models (core/repositories/postsrepo/model_gen.go)
type Post struct {
    PostId    string          `json:"post_id" db:"post_id"`
    UserId    string          `json:"user_id" db:"user_id"`
    Title     string          `json:"title" db:"title"`
    Content   *string         `json:"content" db:"content"`
    Published bool            `json:"published" db:"published"`
    CreatedAt *time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt *time.Time      `json:"updated_at" db:"updated_at"`
}

// Repository methods (repository_gen.go)
repo.Create(ctx, CreatePost)
repo.Get(ctx, postID)
repo.Update(ctx, postID, UpdatePost)
repo.Delete(ctx, postID)
repo.List(ctx, filter, fop)
repo.ListByUserId(ctx, userID, fop)  // Auto-generated from FK!
```

### 🗄️ Store Layer

```go
// Full pgx v5 implementation (store_gen.go)
- CREATE with RETURNING (handles auto-generated PKs)
- GET with proper error handling
- UPDATE with dynamic SET clause
- DELETE with row count validation
- LIST with cursor pagination
- ListByUserId with pagination (from FK!)
```

### 🌉 Bridge Layer

```go
// REST API with camelCase JSON (http_gen.go)
type Post struct {
    PostId    string     `json:"postId"`      // camelCase!
    UserId    string     `json:"userId"`
    Title     string     `json:"title"`
    Content   *string    `json:"content,omitempty"`
    Published bool       `json:"published"`
    CreatedAt *time.Time `json:"createdAt,omitempty"`
    UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// All handlers auto-generated
httpList()           // GET /posts
httpGetByID()        // GET /posts/{post_id}
httpCreate()         // POST /posts
httpUpdate()         // PUT /posts/{post_id}
httpDelete()         // DELETE /posts/{post_id}
httpListByUserId()   // GET /users/{user_id}/posts
```

## Next Steps

### 1. Wire Up Your Routes (2 minutes)

```go
// In your main.go or routes setup
import (
    "github.com/jrazmi/envoker/bridge/repositories/postsrepobridge"
    "github.com/jrazmi/envoker/core/repositories/postsrepo"
    "github.com/jrazmi/envoker/core/repositories/postsrepo/stores/postspgxstore"
)

// Create the stack
store := postspgxstore.NewStore(log, pool)
repo := postsrepo.NewRepository(log, store)

// Wire up routes
apiGroup := router.Group("/api/v1")
postsrepobridge.AddHttpRoutes(apiGroup, postsrepobridge.Config{
    Log:        log,
    Repository: repo,
})
```

### 2. Add Custom Business Logic (optional)

```go
// core/repositories/postsrepo/posts.go (your custom file - won't be overwritten!)
package postsrepo

// Custom method
func (r *Repository) PublishPost(ctx context.Context, postID string) error {
    // Your custom logic here
    return r.Update(ctx, postID, UpdatePost{
        Published: &[]bool{true}[0],  // Helper to get *bool
    })
}
```

### 3. Test It!

```bash
# Start your API
make watch

# Create a post
curl -X POST http://localhost:3000/api/v1/posts \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user-123",
    "title": "My First Post",
    "content": "Hello World!",
    "published": true
  }'

# List posts
curl http://localhost:3000/api/v1/posts?limit=10

# List posts by user
curl http://localhost:3000/api/v1/users/user-123/posts
```

## Common Workflows

### Generate Multiple Tables

```bash
# Create multiple schemas
cat > schema/comments.sql << 'EOF'
CREATE TABLE public.comments (
    comment_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id uuid NOT NULL,
    user_id uuid NOT NULL,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    FOREIGN KEY (post_id) REFERENCES public.posts (post_id),
    FOREIGN KEY (user_id) REFERENCES public.users (user_id)
);
EOF

# Generate all at once
./scripts/generate.sh all -force

# Or one at a time
make generate-sql SQL=schema/posts.sql
make generate-sql SQL=schema/comments.sql
```

### Update Existing Schema

```bash
# 1. Edit your SQL file
vim schema/posts.sql

# 2. Regenerate (with -force to overwrite)
make generate-sql SQL=schema/posts.sql

# 3. Your custom files (posts.go, custom_methods.go) are safe!
#    Only *_gen.go files are overwritten
```

### Generate Only Specific Layers

```bash
# Only repository
go run app/generators/main.go repositorygen -sql=schema/posts.sql -force

# Only store
go run app/generators/main.go storegen -sql=schema/posts.sql -force

# Only bridge
go run app/generators/main.go bridgegen -sql=schema/posts.sql -force

# Repository + Store (no bridge)
go run app/generators/main.go generate -sql=schema/posts.sql -layers=repository,store -force
```

## Troubleshooting

### "File already exists"

**Solution:** Use the `-force` flag

### "Unknown PostgreSQL type"

**Solution:** Check `app/generators/sqlparser/mapper.go` for supported types

### Foreign key routes not showing up

**Solution:** Make sure your CREATE TABLE includes the FK constraint:

```sql
FOREIGN KEY (user_id) REFERENCES public.users (user_id)
```

### Need help?

```bash
# Show all generator commands
go run app/generators/main.go help

# Show makefile targets
make generate-help
```

## Pro Tips 💡

1. **Always use `-force` in development** - saves time clicking through prompts
2. **Name your tables meaningfully** - table names drive all generated code
3. **Use foreign keys** - they auto-generate relationship endpoints
4. **Review the generated code** - it's readable and follows your patterns
5. **Never edit `*_gen.go` files** - they'll be overwritten on next generation
6. **Keep SQL simple** - one table per file works best

## What's Next?

- Read the full [README](README.md) for advanced features
- Check out the [Generation Plan](generationPlan.md) for architecture details
- Explore the generated code to understand the patterns
- Start building your API! 🚀

---

**You just went from SQL to REST API in 60 seconds!** ⚡

Happy generating! 🎉
