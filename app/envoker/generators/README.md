# Taskmaster Code Generators

> **From SQL to Production API in Milliseconds** 🚀

A powerful, composable code generation system that transforms SQL `CREATE TABLE` statements into complete, production-ready Go applications with full-stack type safety.

## 🎯 What It Does

Write a SQL schema, run one command, and get:

- ✅ **Repository Layer**: Domain models with validation
- ✅ **Store Layer**: PostgreSQL implementation with pgx v5
- ✅ **Bridge Layer**: REST API handlers with JSON serialization
- ✅ **Foreign Key Methods**: Automatic relationship endpoints
- ✅ **Pagination**: Cursor-based pagination out of the box
- ✅ **Type Safety**: End-to-end type checking

All generated code follows the `_gen` suffix pattern to protect your custom logic.

## 🚀 Quick Start

### 1. Create a SQL Schema

```sql
-- schema/users.sql
CREATE TABLE public.users (
    user_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email varchar(255) NOT NULL,
    name varchar(100) NOT NULL,
    avatar_url varchar(500),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);
```

### 2. Generate Everything

```bash
# Generate all layers (Repository + Store + Bridge)
go run app/generators/main.go generate -sql=schema/users.sql -force

# Or use the convenience script
./scripts/generate.sh schema/users.sql -force

# Generate all schemas at once
./scripts/generate.sh all -force
```

### 3. You're Done! 🎉

10 files generated in ~5ms!

## 📚 Commands

### `generate` (Recommended)

Generate all layers from SQL:

```bash
go run app/generators/main.go generate -sql=schema/users.sql [-force] [-layers=all]
```

**Flags:**

- `-sql`: Path to SQL file (required)
- `-force`: Overwrite existing files
- `-layers`: Comma-separated layers (default: `all`)
  - Options: `repository`, `store`, `bridge`, `all`
- `-output`: Base output directory (default: `.`)
- `-module`: Go module path (default: `github.com/jrazmi/envoker`)

### Individual Layer Commands

```bash
# Repository only
go run app/generators/main.go repositorygen -sql=schema/users.sql

# Store only
go run app/generators/main.go storegen -sql=schema/users.sql

# Bridge only
go run app/generators/main.go bridgegen -sql=schema/users.sql
```

## 🎨 What Gets Generated

From one SQL file, you get 10 production-ready files:

```
core/repositories/usersrepo/
├── model_gen.go           # User, CreateUser, UpdateUser, FilterUser
├── repository_gen.go      # Repository interface & implementation
└── stores/
    └── userspgxstore/
        └── store_gen.go   # pgx database implementation

bridge/repositories/usersrepobridge/
├── bridge_gen.go          # Bridge struct
├── http_gen.go            # REST API handlers
├── model_gen.go           # API models (camelCase JSON)
├── marshal_gen.go         # Conversion functions
└── fop_gen.go             # Query/path parsers
```

## 🔧 Advanced Features

### Foreign Key Relationships

```sql
CREATE TABLE public.posts (
    post_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    title varchar(200) NOT NULL,
    content text,
    FOREIGN KEY (user_id) REFERENCES public.users (user_id) ON DELETE CASCADE
);
```

**Auto-generates:**

- Repository method: `ListByUserId(ctx, userID, fop)`
- Store method: `ListByUserId(ctx, userID, fop)`
- HTTP route: `GET /users/{user_id}/posts`

### Type Mappings

| PostgreSQL Type            | Go Type           | JSON Type |
| -------------------------- | ----------------- | --------- |
| `uuid`                     | `string`          | `string`  |
| `varchar(n)`               | `string`          | `string`  |
| `text`                     | `string`          | `string`  |
| `integer`, `int`           | `int`             | `integer` |
| `bigint`, `int8`           | `int64`           | `integer` |
| `boolean`                  | `bool`            | `boolean` |
| `timestamp with time zone` | `time.Time`       | `string`  |
| `jsonb`, `json`            | `json.RawMessage` | `object`  |

**Nullable columns** become pointers: `*string`, `*int`, `*time.Time`

## 🛡️ Overwrite Protection

All generated files use `_gen` suffix:

- `model_gen.go` ← **generated, will overwrite**
- `model.go` ← **your custom code, safe**

## 📊 Performance

| Operation      | Time     | Files Generated |
| -------------- | -------- | --------------- |
| **Full Stack** | **~5ms** | **10**          |

## 💡 Best Practices

1. **Keep SQL Simple**: One table per file
2. **Use Meaningful Names**: Table and column names drive everything
3. **Review Generated Code**: Check validation tags and types
4. **Custom Logic Separately**: Never edit `_gen` files
5. **Regenerate Often**: SQL changes → regenerate immediately

## 📁 Project Structure

```
app/generators/
├── main.go                    # CLI entry point
├── sqlparser/                 # SQL parsing & analysis
├── repositorygen/             # Repository layer generation
├── pgxstores/                 # Store layer generation
├── bridgegen/                 # Bridge layer generation
└── orchestrator/              # Full-stack orchestration
```

## 🚀 Integration Example

```go
// Wire up in your main.go
import (
    "github.com/jrazmi/envoker/bridge/repositories/usersrepobridge"
    "github.com/jrazmi/envoker/core/repositories/usersrepo"
    "github.com/jrazmi/envoker/core/repositories/usersrepo/stores/userspgxstore"
)

// Create repository
store := userspgxstore.NewStore(log, pool)
repo := usersrepo.NewRepository(log, store)

// Wire up routes
apiGroup := router.Group("/api/v1")
usersrepobridge.AddHttpRoutes(apiGroup, usersrepobridge.Config{
    Log:        log,
    Repository: repo,
})
```

## 🐛 Troubleshooting

### "File already exists"

Use `-force` flag to overwrite

### Foreign Key Routes Not Working

Ensure FK constraint is in CREATE TABLE:

```sql
FOREIGN KEY (user_id) REFERENCES public.users (user_id)
```

## 📖 Learn More

See `app/generators/generationPlan.md` for the complete vision and architecture.

---

**Built with ❤️ by the Taskmaster team**

_From SQL to API in milliseconds_ 🚀
