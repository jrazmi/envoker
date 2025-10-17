# Code Generation Plan: SQL-to-API Scaffolding System

## Vision

Automatically generate complete REST API endpoints from SQL CREATE TABLE statements, creating:

- Repository layer (data access)
- Store layer (pgx database implementation)
- Bridge layer (HTTP/REST API)

**From this:**

```sql
CREATE TABLE task_executions (
    execution_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id uuid NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    ...
);
```

**To this:**

```
GET    /api/v1/task-executions
POST   /api/v1/task-executions
GET    /api/v1/task-executions/{id}
PUT    /api/v1/task-executions/{id}
DELETE /api/v1/task-executions/{id}

# Foreign key relationships
GET    /api/v1/tasks/{task_id}/executions
GET    /api/v1/applications/{app_id}/executions
```

---

## Core Principles

### 1. Generated Files Use `_gen` Suffix

**All** generated files must have `_gen` suffix to protect custom code from being overwritten.

```
✅ model_gen.go        - Safe to regenerate
✅ model.go            - Custom extensions (manual)
✅ store_gen.go        - Safe to regenerate
✅ store.go            - Custom methods (manual)
```

### 2. Overwrite Protection

- Prompt user before overwriting existing `_gen` files
- Never touch files without `_gen` suffix
- Provide `--force` flag to skip prompts
- Provide `--dry-run` flag to preview changes

### 3. Foreign Key Relationships

Automatically generate `ListByForeignKey` methods for all foreign keys:

```go
// From: task_id REFERENCES tasks(task_id)
// Generate:
func (r *Repository) ListByTaskID(ctx, taskID, orderBy, page) ([]TaskExecution, PageInfo, error)

// From: application_id REFERENCES applications(application_id)
// Generate:
func (r *Repository) ListByApplicationID(ctx, appID, orderBy, page) ([]TaskExecution, PageInfo, error)
```

### 4. REST-First Architecture

Follow existing `bridge/repositories/tasksrepobridge` pattern:

- `http_gen.go` - Generated HTTP handlers
- `model_gen.go` - Generated DTOs
- `marshal_gen.go` - Generated conversions

### 5. Composable Generators

Each generator can run independently:

```bash
generator repo-from-sql --sql=schema.sql --table=task_executions
generator pgxstore --entity=TaskExecution --table=task_executions
generator bridge --entity=TaskExecution --repo=taskexecutionsrepo
```

Or chained together:

```bash
generator scaffold-from-sql --sql=schema.sql --table=task_executions
```

---

## File Structure

### Repository Layer

```
core/repositories/taskexecutionsrepo/
├── model_gen.go              # Generated: Entity, CreateEntity, UpdateEntity
├── model.go                  # Manual: Custom model extensions
├── fop_gen.go                # Generated: QueryFilter, OrderBy constants
├── fop.go                    # Manual: Custom filter logic
├── taskexecutionsrepo_gen.go # Generated: Repository interface + implementation
├── taskexecutionsrepo.go     # Manual: Custom repository methods
└── stores/
    └── taskexecutionspgxstore/
        ├── store.go          # Manual: Store struct + custom methods
        ├── store_gen.go      # Generated: CRUD operations
        └── fop_gen.go        # Generated: Filter application logic
```

### Bridge Layer

```
bridge/repositories/taskexecutionsrepobridge/
├── model_gen.go                       # Generated: Bridge DTOs
├── model.go                           # Manual: Custom DTOs
├── marshal_gen.go                     # Generated: Entity ↔ DTO conversions
├── marshal.go                         # Manual: Custom conversions
├── fop_gen.go                         # Generated: Query param parsing
├── fop.go                             # Manual: Custom filter parsing
├── http_gen.go                        # Generated: HTTP handlers
├── http.go                            # Manual: Custom HTTP handlers
├── taskexecutionsrepobridge_gen.go    # Generated: Bridge struct
└── taskexecutionsrepobridge.go        # Manual: Custom bridge methods
```

---

## Phase 0: SQL Parser & Schema Analysis

### Purpose

Parse CREATE TABLE statements into structured schema representation.

### Package Structure

```
app/generators/sqlparser/
├── parser.go          # Main SQL parser
├── types.go           # Schema data structures
├── mapper.go          # PostgreSQL → Go type mapping
├── analyzer.go        # FK detection, constraint analysis
└── validator.go       # Validation rule generation
```

### Core Data Structures

```go
type TableSchema struct {
    Name        string           // "task_executions"
    Schema      string           // "public" (default)
    Columns     []Column
    PrimaryKey  PrimaryKeyInfo
    ForeignKeys []ForeignKey
    Indexes     []Index
    Constraints []Constraint
    Comments    map[string]string // column comments
}

type Column struct {
    Name           string        // "execution_id"
    DBType         string        // "uuid", "varchar(100)", "timestamp"
    GoType         string        // "string", "*time.Time"
    GoImportPath   string        // "time", "encoding/json"
    IsNullable     bool
    IsPrimaryKey   bool
    IsForeignKey   bool
    DefaultValue   string        // "gen_random_uuid()", "now()"
    HasDefault     bool
    MaxLength      int           // for varchar(n)
    Precision      int           // for numeric(p,s)
    Scale          int
    References     *ForeignKey
    ValidationTags string        // "required,uuid,min=1"
    Comment        string
}

type ForeignKey struct {
    ColumnName      string        // "task_id"
    RefTable        string        // "tasks"
    RefSchema       string        // "public"
    RefColumn       string        // "task_id"
    OnDelete        string        // "CASCADE", "SET NULL", "RESTRICT"
    OnUpdate        string        // "CASCADE", "SET NULL", "RESTRICT"

    // Derived names for code generation
    EntityName      string        // "Task"
    RepoPackageName string        // "tasksrepo"
    MethodSuffix    string        // "ByTaskID"
    HTTPPathSegment string        // "tasks/{task_id}/executions"
}

type PrimaryKeyInfo struct {
    ColumnName  string
    GoType      string
    HasDefault  bool          // true if DEFAULT clause exists
    DefaultExpr string        // "gen_random_uuid()", "nextval(...)"
}

type Constraint struct {
    Name       string
    Type       string         // "CHECK", "UNIQUE", etc.
    Definition string
}
```

### PostgreSQL → Go Type Mapping

```go
var typeMap = map[string]TypeMapping{
    "uuid": {
        GoType:       "string",
        Validation:   "uuid",
        JSONType:     "string",
    },
    "varchar": {
        GoType:       "string",
        Validation:   "max=%d",  // %d replaced with length
        JSONType:     "string",
    },
    "text": {
        GoType:       "string",
        JSONType:     "string",
    },
    "int2": {
        GoType:       "int16",
        Validation:   "number",
        JSONType:     "number",
    },
    "int4": {
        GoType:       "int",
        Validation:   "number",
        JSONType:     "number",
    },
    "int8": {
        GoType:       "int64",
        Validation:   "number",
        JSONType:     "number",
    },
    "float4": {
        GoType:       "float32",
        Validation:   "number",
        JSONType:     "number",
    },
    "float8": {
        GoType:       "float64",
        Validation:   "number",
        JSONType:     "number",
    },
    "numeric": {
        GoType:       "float64",
        Validation:   "number",
        JSONType:     "number",
    },
    "bool": {
        GoType:       "bool",
        JSONType:     "boolean",
    },
    "timestamp": {
        GoType:       "*time.Time",
        Import:       "time",
        JSONType:     "string (ISO 8601)",
    },
    "timestamptz": {
        GoType:       "*time.Time",
        Import:       "time",
        JSONType:     "string (ISO 8601)",
    },
    "date": {
        GoType:       "*time.Time",
        Import:       "time",
        JSONType:     "string (ISO 8601)",
    },
    "jsonb": {
        GoType:       "*json.RawMessage",
        Import:       "encoding/json",
        JSONType:     "object",
    },
    "json": {
        GoType:       "*json.RawMessage",
        Import:       "encoding/json",
        JSONType:     "object",
    },
    "bytea": {
        GoType:       "[]byte",
        JSONType:     "string (base64)",
    },
}
```

### Validation Rule Generation

From SQL constraints to Go struct tags:

```sql
-- SQL
task_id uuid NOT NULL
status varchar(50) NOT NULL
priority int4 DEFAULT 0
attempt_number int4 NOT NULL CHECK (attempt_number >= 1)
```

```go
// Generated validation tags
TaskID         string `validate:"required,uuid"`
Status         string `validate:"required,max=50"`
Priority       *int   `validate:"omitempty,min=0"`
AttemptNumber  int    `validate:"required,min=1"`
```

### Naming Conventions

```go
type NamingContext struct {
    // From table name: "task_executions"
    TableName       string   // "task_executions"
    TableNameSingular string // "task_execution"

    // Entity names
    EntityName      string   // "TaskExecution"
    EntityNameLower string   // "taskExecution"
    EntityNameSnake string   // "task_execution"
    EntityNamePlural string  // "TaskExecutions"

    // Package names
    PackageName     string   // "taskexecutionsrepo"
    StorePackage    string   // "taskexecutionspgxstore"
    BridgePackage   string   // "taskexecutionsrepobridge"

    // Paths
    RepoPath        string   // "core/repositories/taskexecutionsrepo"
    StorePath       string   // "core/repositories/taskexecutionsrepo/stores/taskexecutionspgxstore"
    BridgePath      string   // "bridge/repositories/taskexecutionsrepobridge"

    // HTTP
    HTTPBasePath    string   // "/task-executions"
    HTTPSingular    string   // "/task-execution"
}

// Special handling for acronyms
type AcronymRule struct {
    Pattern     string
    Replacement string
}

var acronyms = []AcronymRule{
    {"api_key", "APIKey"},
    {"api_keys", "APIKeys"},
    {"id", "ID"},
    {"url", "URL"},
    {"uri", "URI"},
    {"uuid", "UUID"},
    {"http", "HTTP"},
    {"https", "HTTPS"},
    {"sql", "SQL"},
}
```

### Foreign Key Analysis

**Input:**

```sql
task_id uuid NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE
```

**Analysis Output:**

```go
ForeignKey{
    ColumnName:      "task_id",
    RefTable:        "tasks",
    RefColumn:       "task_id",
    OnDelete:        "CASCADE",

    // Derived from ref table name
    EntityName:      "Task",
    RepoPackageName: "tasksrepo",
    MethodSuffix:    "ByTaskID",
    HTTPPathSegment: "tasks/{task_id}/executions",
}
```

**Generated Methods:**

- Repository: `ListByTaskID(ctx, taskID, orderBy, page)`
- Store: `ListByTaskID(ctx, taskID, orderBy, page, forPrevious)`
- Bridge: `httpListByTaskID(ctx, r)`
- HTTP Route: `GET /tasks/{task_id}/executions`

---

## Phase 1: Repository Generator

### Purpose

Generate repository layer from parsed schema.

### Package Structure

```
app/generators/repogen/
├── generator.go       # Main repository generator
├── model.go           # model_gen.go template
├── fop.go             # fop_gen.go template
├── repository.go      # {entity}repo_gen.go template
└── naming.go          # Naming convention utilities
```

### 1.1 Generate `model_gen.go`

**Template:**

```go
// Code generated by repogen. DO NOT EDIT.

package {{.PackageName}}

import (
{{- range .Imports}}
    "{{.}}"
{{- end}}
)

// {{.EntityName}} represents the {{.TableName}} table structure
type {{.EntityName}} struct {
{{- range .Columns}}
    {{.Name}} {{.GoType}} `db:"{{.DBColumn}}" json:"{{.JSONName}}"{{if .OmitEmpty}},omitempty{{end}}`
{{- end}}
}

// Create{{.EntityName}} represents the input for creating a {{.EntityNameLower}}
type Create{{.EntityName}} struct {
{{- range .CreateColumns}}
    {{.Name}} {{.GoType}} `db:"{{.DBColumn}}" json:"{{.JSONName}}" validate:"{{.ValidationTags}}"{{if .OmitEmpty}},omitempty{{end}}`
{{- end}}
}

// Update{{.EntityName}} represents the input for updating a {{.EntityNameLower}}
type Update{{.EntityName}} struct {
{{- range .UpdateColumns}}
    {{.Name}} {{.PointerType}} `db:"{{.DBColumn}}" json:"{{.JSONName}},omitempty" validate:"{{.ValidationTags}}"`
{{- end}}
}
```

**Field Inclusion Rules:**

**Entity struct:**

- Include ALL columns
- Nullable columns = pointer types
- Add `omitempty` for nullable fields

**CreateEntity struct:**

- EXCLUDE primary key IF it has DEFAULT
- EXCLUDE columns with DEFAULT (created_at, updated_at, started_at)
- INCLUDE all foreign key columns
- Nullable columns = pointer types
- Add validation tags from constraints

**UpdateEntity struct:**

- EXCLUDE primary key
- EXCLUDE immutable fields (created_at, typically)
- ALL fields are pointers (for optional updates)
- Add validation tags with `omitempty`

### 1.2 Generate `fop_gen.go`

**Template:**

```go
// Code generated by repogen. DO NOT EDIT.

package {{.PackageName}}

import "time"

// Filter field constants
const (
{{- range .FilterableColumns}}
    Filter{{.ConstName}} = "{{.DBColumn}}"
{{- end}}
)

// Order field constants
const (
{{- range .OrderableColumns}}
    OrderBy{{.ConstName}} = "{{.DBColumn}}"
{{- end}}
)

// QueryFilter defines available filters for {{.TableName}}
type QueryFilter struct {
    // Exact match filters
{{- range .ExactMatchFilters}}
    {{.Name}} *{{.GoType}}
{{- end}}

    // Range filters
{{- range .RangeFilters}}
    {{.Name}}After  *{{.GoType}}
    {{.Name}}Before *{{.GoType}}
{{- end}}

{{- range .NumericRangeFilters}}
    {{.Name}}Min *{{.GoType}}
    {{.Name}}Max *{{.GoType}}
{{- end}}

    // Boolean filters
{{- range .BooleanFilters}}
    {{.Name}} *bool  // {{.Description}}
{{- end}}

    // Search
    SearchTerm *string
}
```

**Filter Generation Rules:**

**Exact match:** All non-text columns under 100 chars

```go
ExecutionID    *string
TaskID         *string
Status         *string
```

**Range filters:** All timestamp/date columns

```go
StartedAtAfter    *time.Time
StartedAtBefore   *time.Time
CompletedAtAfter  *time.Time
CompletedAtBefore *time.Time
```

**Numeric range:** All numeric columns

```go
DurationMsMin *int
DurationMsMax *int
PriorityMin   *int
PriorityMax   *int
```

**Boolean helpers:** Generated from nullable columns

```go
HasError    *bool  // error_message IS NOT NULL
IsCompleted *bool  // completed_at IS NOT NULL
```

**SearchTerm:** Searches across all text/varchar columns with ILIKE

### 1.3 Generate `{entity}repo_gen.go`

**Template:**

```go
// Code generated by repogen. DO NOT EDIT.

package {{.PackageName}}

import (
    "context"
    "errors"
    "fmt"

    "github.com/jrazmi/envoker/core/scaffolding/fop"
    "github.com/jrazmi/envoker/sdk/logger"
)

var (
    ErrNotFound = errors.New("{{.EntityNameLower}} not found")
)

// Storer interface defines storage operations
type Storer interface {
    Create(ctx context.Context, payload Create{{.EntityName}}) ({{.EntityName}}, error)
    Update(ctx context.Context, ID string, payload Update{{.EntityName}}) error
    Get(ctx context.Context, ID string, filter QueryFilter) ({{.EntityName}}, error)
    List(ctx context.Context, filter QueryFilter, orderBy fop.By, page fop.PageStringCursor, forPrevious bool) ([]{{.EntityName}}, error)
    Delete(ctx context.Context, ID string) error

    // FK relationship methods
{{- range .ForeignKeys}}
    ListBy{{.MethodSuffix}}(ctx context.Context, {{.ParamName}} string, orderBy fop.By, page fop.PageStringCursor, forPrevious bool) ([]{{$.EntityName}}, error)
{{- end}}
}

type Repository struct {
    log    *logger.Logger
    storer Storer
}

func NewRepository(log *logger.Logger, storer Storer) *Repository {
    return &Repository{
        log:    log,
        storer: storer,
    }
}

// Get retrieves a single {{.EntityNameLower}} by ID
func (r *Repository) Get(ctx context.Context, ID string, filter QueryFilter) ({{.EntityName}}, error) {
    record, err := r.storer.Get(ctx, ID, filter)
    if err != nil {
        return {{.EntityName}}{}, fmt.Errorf("get {{.EntityNameLower}}[%s]: %w", ID, err)
    }
    return record, nil
}

// List retrieves {{.EntityNamePlural}} with pagination
func (r *Repository) List(ctx context.Context, filter QueryFilter, order fop.By, page fop.PageStringCursor) ([]{{.EntityName}}, fop.PageInfoStringCursor, error) {
    // Standard pagination logic (fetch n+1, build pageInfo)
    listPage := fop.PageStringCursor{
        Limit:  page.Limit + 1,
        Cursor: page.Cursor,
    }

    records, err := r.storer.List(ctx, filter, order, listPage, false)
    if err != nil {
        return nil, fop.PageInfoStringCursor{}, fmt.Errorf("list: %w", err)
    }

    returnableRecords := records
    nextCursor := ""

    if len(records) > page.Limit {
        returnableRecords = records[:page.Limit]
        lastRecord := returnableRecords[len(returnableRecords)-1]
        nextCursor, err = Encode{{.EntityName}}Cursor(*lastRecord.{{.PrimaryKey.Name}}, lastRecord.{{.PrimaryKey.DBColumn}})
        if err != nil {
            return nil, fop.PageInfoStringCursor{}, fmt.Errorf("encode next cursor: %w", err)
        }
    }

    pageInfo := fop.PageInfoStringCursor{
        HasPrev:        false,
        Limit:          page.Limit,
        PreviousCursor: "",
        NextCursor:     nextCursor,
        PageTotal:      len(returnableRecords),
    }

    // Check for previous page
    if page.Cursor != "" {
        prevRecords, err := r.storer.List(ctx, filter, order, page, true)
        if err == nil && len(prevRecords) > 0 {
            pageInfo.HasPrev = true
            if len(prevRecords) == page.Limit {
                firstRecord := prevRecords[0]
                pageInfo.PreviousCursor, err = Encode{{.EntityName}}Cursor(*firstRecord.{{.PrimaryKey.Name}}, firstRecord.{{.PrimaryKey.DBColumn}})
                if err != nil {
                    return nil, fop.PageInfoStringCursor{}, fmt.Errorf("encode prev cursor: %w", err)
                }
            }
        }
    }

    return returnableRecords, pageInfo, nil
}

// Create adds a new {{.EntityNameLower}}
func (r *Repository) Create(ctx context.Context, payload Create{{.EntityName}}) ({{.EntityName}}, error) {
    // ID generated by database{{if not .PrimaryKey.HasDefault}} or provided{{end}}
    record, err := r.storer.Create(ctx, payload)
    if err != nil {
        return {{.EntityName}}{}, fmt.Errorf("create {{.EntityNameLower}}: %w", err)
    }
    return record, nil
}

// Update modifies an existing {{.EntityNameLower}}
func (r *Repository) Update(ctx context.Context, ID string, payload Update{{.EntityName}}) error {
    if err := r.storer.Update(ctx, ID, payload); err != nil {
        return fmt.Errorf("update {{.EntityNameLower}}[%s]: %w", ID, err)
    }
    return nil
}

// Delete removes a {{.EntityNameLower}}
func (r *Repository) Delete(ctx context.Context, ID string) error {
    if err := r.storer.Delete(ctx, ID); err != nil {
        return fmt.Errorf("delete {{.EntityNameLower}}[%s]: %w", ID, err)
    }
    return nil
}

{{- range .ForeignKeys}}

// ListBy{{.MethodSuffix}} retrieves all {{$.EntityNamePlural}} for a specific {{.EntityName}}
func (r *Repository) ListBy{{.MethodSuffix}}(ctx context.Context, {{.ParamName}} string, order fop.By, page fop.PageStringCursor) ([]{{$.EntityName}}, fop.PageInfoStringCursor, error) {
    listPage := fop.PageStringCursor{
        Limit:  page.Limit + 1,
        Cursor: page.Cursor,
    }

    records, err := r.storer.ListBy{{.MethodSuffix}}(ctx, {{.ParamName}}, order, listPage, false)
    if err != nil {
        return nil, fop.PageInfoStringCursor{}, fmt.Errorf("list by {{.ParamName}}: %w", err)
    }

    // Same pagination logic as List
    // ... (build pageInfo)

    return returnableRecords, pageInfo, nil
}
{{- end}}
```

### 1.4 Generate Store Scaffold

Create `stores/{entity}pgxstore/store.go` (manual file):

```go
package {{.StorePackage}}

//go:generate go run ../../../../../app/generators/main.go pgxstore -entity={{.EntityName}} -table={{.TableName}} -pk={{.PKColumn}}

import (
    "github.com/jrazmi/envoker/infrastructure/postgresdb"
    "github.com/jrazmi/envoker/sdk/logger"
)

type Store struct {
    log  *logger.Logger
    pool *postgresdb.Pool
}

func NewStore(log *logger.Logger, pool *postgresdb.Pool) *Store {
    return &Store{
        log:  log,
        pool: pool,
    }
}

// Add custom methods here
```

---

## Phase 2: Enhanced PGX Store Generator

### Purpose

Generate pgx store implementation with CRUD and FK methods.

### Updates to Existing Generator

```
app/generators/pgxstores/
├── generator.go       # Updated: accept schema, generate FK methods
├── template.go        # Updated: store_gen.go template
├── fop_template.go    # NEW: fop_gen.go template
└── fk_template.go     # NEW: FK method templates
```

### 2.1 Generate `store_gen.go`

**Enhanced with FK methods:**

```go
// Code generated by storegen. DO NOT EDIT.

package {{.StorePackage}}

import (
    "bytes"
    "context"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/jackc/pgx/v5"
    "{{.ModulePath}}/core/repositories/{{.RepoPackage}}"
    "{{.ModulePath}}/core/scaffolding/fop"
    "{{.ModulePath}}/infrastructure/postgresdb"
)

// Get, Create, Update, Delete, List methods (already implemented)
// ...

{{- range .ForeignKeys}}

// ListBy{{.MethodSuffix}} retrieves {{$.EntityNamePlural}} for a specific {{.EntityName}}
func (s *Store) ListBy{{.MethodSuffix}}(ctx context.Context, {{.ParamName}} string, orderBy fop.By, page fop.PageStringCursor, forPrevious bool) ([]{{$.RepoPackage}}.{{$.EntityName}}, error) {
    data := pgx.NamedArgs{
        "{{.DBColumn}}": {{.ParamName}},
    }
    aliases := map[string]string{}

    // Start building the query with FK filter
    buf := bytes.NewBufferString(`
        SELECT
            {{range $i, $f := $.EntityFields}}{{if $i}}, {{end}}{{$f.DBColumn}}{{end}}
        FROM
            {{$.Schema}}.{{$.TableName}}
        WHERE
            {{.DBColumn}} = @{{.DBColumn}}`)

    // Setup cursor pagination
    cursorConfig := postgresdb.StringCursorConfig{
        Cursor:     page.Cursor,
        OrderField: orderByFields[orderBy.Field],
        PKField:    "{{$.PK}}",
        TableName:  "{{$.TableName}}",
        Direction:  orderBy.Direction,
        Limit:      page.Limit,
    }

    // Apply cursor pagination
    if page.Cursor != "" {
        err := postgresdb.ApplyStringCursorPagination(buf, data, cursorConfig, forPrevious)
        if err != nil {
            return nil, fmt.Errorf("cursor pagination: %w", err)
        }
    }

    // Add ordering
    err := postgresdb.AddOrderByClause(buf, cursorConfig.OrderField, cursorConfig.PKField, cursorConfig.Direction, forPrevious)
    if err != nil {
        return nil, fmt.Errorf("order: %w", err)
    }

    // Add limit
    postgresdb.AddLimitClause(cursorConfig.Limit, data, buf)

    // Execute
    query := buf.String()
    rows, err := s.pool.Query(ctx, query, data)
    if err != nil {
        return nil, postgresdb.HandlePgError(err)
    }
    defer rows.Close()

    records, err := pgx.CollectRows(rows, pgx.RowToStructByName[{{$.RepoPackage}}.{{$.EntityName}}])
    if err != nil {
        return nil, postgresdb.HandlePgError(err)
    }

    // Reverse if fetching previous page
    if forPrevious && len(records) > 0 {
        for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
            records[i], records[j] = records[j], records[i]
        }
    }

    return records, nil
}
{{- end}}
```

### 2.2 Generate `fop_gen.go`

**NEW: Filter application logic:**

```go
// Code generated by storegen. DO NOT EDIT.

package {{.StorePackage}}

import (
    "bytes"
    "strings"

    "github.com/jackc/pgx/v5"
    "{{.ModulePath}}/core/repositories/{{.RepoPackage}}"
)

// orderByFields maps repository field names to database column names
var orderByFields = map[string]string{
{{- range .OrderableColumns}}
    {{$.RepoPackage}}.OrderBy{{.ConstName}}: "{{.DBColumn}}",
{{- end}}
}

// applyFilter applies query filters to the SQL query
func (s *Store) applyFilter(filter {{.RepoPackage}}.QueryFilter, data pgx.NamedArgs, buf *bytes.Buffer, aliases map[string]string) {
    var conditions []string

    // Exact match filters
{{- range .ExactMatchFilters}}
    if filter.{{.Name}} != nil {
        conditions = append(conditions, "{{.DBColumn}} = @{{.ParamName}}")
        data["{{.ParamName}}"] = *filter.{{.Name}}
    }
{{- end}}

    // Range filters - timestamps
{{- range .TimestampRangeFilters}}
    if filter.{{.Name}}After != nil {
        conditions = append(conditions, "{{.DBColumn}} > @{{.ParamName}}_after")
        data["{{.ParamName}}_after"] = *filter.{{.Name}}After
    }
    if filter.{{.Name}}Before != nil {
        conditions = append(conditions, "{{.DBColumn}} < @{{.ParamName}}_before")
        data["{{.ParamName}}_before"] = *filter.{{.Name}}Before
    }
{{- end}}

    // Range filters - numeric
{{- range .NumericRangeFilters}}
    if filter.{{.Name}}Min != nil {
        conditions = append(conditions, "{{.DBColumn}} >= @{{.ParamName}}_min")
        data["{{.ParamName}}_min"] = *filter.{{.Name}}Min
    }
    if filter.{{.Name}}Max != nil {
        conditions = append(conditions, "{{.DBColumn}} <= @{{.ParamName}}_max")
        data["{{.ParamName}}_max"] = *filter.{{.Name}}Max
    }
{{- end}}

    // Boolean helper filters
{{- range .BooleanFilters}}
    if filter.{{.Name}} != nil {
        if *filter.{{.Name}} {
            conditions = append(conditions, "{{.SQLCondition}}")
        } else {
            conditions = append(conditions, "{{.SQLNegatedCondition}}")
        }
    }
{{- end}}

    // Search term (ILIKE across text columns)
    if filter.SearchTerm != nil && *filter.SearchTerm != "" {
        searchPattern := "%" + *filter.SearchTerm + "%"
        searchConditions := []string{
{{- range .SearchableColumns}}
            "{{.DBColumn}} ILIKE @search_term",
{{- end}}
        }
        conditions = append(conditions, "("+strings.Join(searchConditions, " OR ")+")")
        data["search_term"] = searchPattern
    }

    // Apply all conditions
    if len(conditions) > 0 {
        buf.WriteString(" WHERE ")
        buf.WriteString(strings.Join(conditions, " AND "))
    }
}
```

---

## Phase 3: Bridge Generator

### Purpose

Generate REST API bridge layer with HTTP handlers.

### Package Structure

```
app/generators/bridgegen/
├── generator.go       # Main bridge generator
├── model.go           # model_gen.go template (DTOs)
├── marshal.go         # marshal_gen.go template
├── fop.go             # fop_gen.go template (query param parsing)
├── http.go            # http_gen.go template
└── bridge.go          # {entity}repobridge_gen.go template
```

### 3.1 Generate `model_gen.go`

**DTOs for API:**

```go
// Code generated by bridgegen. DO NOT EDIT.

package {{.BridgePackage}}

import "encoding/json"

// {{.EntityName}} represents the API response model
type {{.EntityName}} struct {
{{- range .Columns}}
    {{.Name}} {{.JSONGoType}} `json:"{{.JSONName}}"{{if .OmitEmpty}},omitempty{{end}}`
{{- end}}
}

// Encode implements web.Encoder
func (e {{.EntityName}}) Encode() ([]byte, string, error) {
    data, err := json.Marshal(e)
    return data, "application/json", err
}

// Create{{.EntityName}}Input represents the input for creating a {{.EntityNameLower}}
type Create{{.EntityName}}Input struct {
{{- range .CreateColumns}}
    {{.Name}} {{.JSONGoType}} `json:"{{.JSONName}}"{{if not .Required}},omitempty{{end}} validate:"{{.ValidationTags}}"`
{{- end}}
}

// Decode implements web.Decoder
func (c *Create{{.EntityName}}Input) Decode(data []byte) error {
    return json.Unmarshal(data, c)
}

// Update{{.EntityName}}Input represents the input for updating a {{.EntityNameLower}}
type Update{{.EntityName}}Input struct {
{{- range .UpdateColumns}}
    {{.Name}} {{.PointerJSONType}} `json:"{{.JSONName}},omitempty" validate:"{{.ValidationTags}}"`
{{- end}}
}

// Decode implements web.Decoder
func (u *Update{{.EntityName}}Input) Decode(data []byte) error {
    return json.Unmarshal(data, u)
}

// {{.EntityName}}List represents a list response
type {{.EntityName}}List struct {
    {{.EntityNamePlural}} []{{.EntityName}} `json:"{{.JSONPluralName}}"`
}

// Encode implements web.Encoder
func (l {{.EntityName}}List) Encode() ([]byte, string, error) {
    data, err := json.Marshal(l)
    return data, "application/json", err
}
```

**Type transformations:**

- `*time.Time` → `string` (ISO 8601)
- `*json.RawMessage` → `map[string]interface{}`
- Pointers for nullable fields

### 3.2 Generate `marshal_gen.go`

**Conversion functions:**

```go
// Code generated by bridgegen. DO NOT EDIT.

package {{.BridgePackage}}

import (
    "encoding/json"

    "{{.ModulePath}}/core/repositories/{{.RepoPackage}}"
    "{{.ModulePath}}/sdk/validation"
)

// MarshalToBridge converts repository entity to bridge DTO
func MarshalToBridge(entity {{.RepoPackage}}.{{.EntityName}}) {{.EntityName}} {
{{- range .Columns}}
    {{if .IsJSONB}}
    var {{.VarName}} map[string]interface{}
    if entity.{{.Name}} != nil {
        json.Unmarshal(*entity.{{.Name}}, &{{.VarName}})
    }
    {{else if .IsTime}}
    {{.VarName}} := validation.FormatTimePtrToString(entity.{{.Name}})
    {{else if .IsPointer}}
    var {{.VarName}} {{.BaseType}}
    if entity.{{.Name}} != nil {
        {{.VarName}} = *entity.{{.Name}}
    }
    {{end}}
{{- end}}

    return {{.EntityName}}{
{{- range .Columns}}
        {{.Name}}: {{if .NeedsConversion}}{{.VarName}}{{else}}entity.{{.Name}}{{end}},
{{- end}}
    }
}

// MarshalListToBridge converts a list of entities to DTOs
func MarshalListToBridge(entities []{{.RepoPackage}}.{{.EntityName}}) []{{.EntityName}} {
    dtos := make([]{{.EntityName}}, len(entities))
    for i, entity := range entities {
        dtos[i] = MarshalToBridge(entity)
    }
    return dtos
}

// MarshalCreateToRepository converts create input to repository type
func MarshalCreateToRepository(input Create{{.EntityName}}Input) {{.RepoPackage}}.Create{{.EntityName}} {
{{- range .CreateColumns}}
    {{if .IsJSONB}}
    var {{.VarName}}Raw *json.RawMessage
    if len(input.{{.Name}}) > 0 {
        bytes, _ := json.Marshal(input.{{.Name}})
        raw := json.RawMessage(bytes)
        {{.VarName}}Raw = &raw
    }
    {{else if .IsTime}}
    {{.VarName}}, _ := validation.ParseTimeString(input.{{.Name}})
    {{end}}
{{- end}}

    return {{.RepoPackage}}.Create{{.EntityName}}{
{{- range .CreateColumns}}
        {{.Name}}: {{if .NeedsConversion}}{{.VarName}}{{else}}input.{{.Name}}{{end}},
{{- end}}
    }
}

// MarshalUpdateToRepository converts update input to repository type
func MarshalUpdateToRepository(input Update{{.EntityName}}Input) {{.RepoPackage}}.Update{{.EntityName}} {
{{- range .UpdateColumns}}
    {{if .IsJSONB}}
    var {{.VarName}}Raw *json.RawMessage
    if input.{{.Name}} != nil && len(*input.{{.Name}}) > 0 {
        bytes, _ := json.Marshal(*input.{{.Name}})
        raw := json.RawMessage(bytes)
        {{.VarName}}Raw = &raw
    }
    {{else if .IsTime}}
    var {{.VarName}} *time.Time
    if input.{{.Name}} != nil {
        parsed, _ := validation.ParseTimeString(*input.{{.Name}})
        {{.VarName}} = parsed
    }
    {{end}}
{{- end}}

    return {{.RepoPackage}}.Update{{.EntityName}}{
{{- range .UpdateColumns}}
        {{.Name}}: {{if .NeedsConversion}}{{.VarName}}{{else}}input.{{.Name}}{{end}},
{{- end}}
    }
}
```

### 3.3 Generate `fop_gen.go`

**Query parameter parsing:**

```go
// Code generated by bridgegen. DO NOT EDIT.

package {{.BridgePackage}}

import (
    "net/http"
    "time"

    "{{.ModulePath}}/core/repositories/{{.RepoPackage}}"
    "{{.ModulePath}}/sdk/validation"
)

// queryParams represents parsed query parameters
type queryParams struct {
    Limit  int
    Cursor string
    Order  string

    // Filters
{{- range .FilterableColumns}}
    {{.Name}} {{if .IsPointer}}*{{end}}{{.GoType}}
{{- end}}
}

// parseQueryParams extracts and validates query parameters
func parseQueryParams(r *http.Request) queryParams {
    qp := queryParams{}

    // Pagination
    qp.Limit = validation.QueryInt(r, "limit", 20)
    qp.Cursor = validation.QueryString(r, "cursor", "")
    qp.Order = validation.QueryString(r, "order", "{{.DefaultOrder}}")

    // Filters
{{- range .FilterableColumns}}
    if value := validation.QueryString(r, "{{.QueryParam}}", ""); value != "" {
        {{if .IsTime}}
        parsed, _ := validation.ParseTimeString(value)
        qp.{{.Name}} = parsed
        {{else if .IsInt}}
        intVal := validation.QueryInt(r, "{{.QueryParam}}", 0)
        qp.{{.Name}} = {{if .IsPointer}}&{{end}}intVal
        {{else if .IsBool}}
        boolVal := validation.QueryBool(r, "{{.QueryParam}}", false)
        qp.{{.Name}} = {{if .IsPointer}}&{{end}}boolVal
        {{else}}
        qp.{{.Name}} = {{if .IsPointer}}&{{end}}value
        {{end}}
    }
{{- end}}

    return qp
}

// parseFilter converts query params to repository filter
func parseFilter(qp queryParams) ({{.RepoPackage}}.QueryFilter, error) {
    return {{.RepoPackage}}.QueryFilter{
{{- range .FilterableColumns}}
        {{.Name}}: qp.{{.Name}},
{{- end}}
    }, nil
}

// parseOrderBy converts order string to repository orderBy
func parseOrderBy(orderStr string) fop.By {
    // Parse "field:direction" format
    // Default to "{{.DefaultOrder}}"
    parts := strings.Split(orderStr, ":")

    field := parts[0]
    direction := fop.ASC
    if len(parts) > 1 && parts[1] == "desc" {
        direction = fop.DESC
    }

    return fop.By{
        Field:     field,
        Direction: direction,
    }
}

// pathParams represents parsed path parameters
type pathParams struct {
{{- range .PathParams}}
    {{.Name}} string
{{- end}}
}

// parsePath extracts path parameters
func parsePath(r *http.Request) (pathParams, error) {
    return pathParams{
{{- range .PathParams}}
        {{.Name}}: web.Param(r, "{{.URLParam}}"),
{{- end}}
    }, nil
}
```

### 3.4 Generate `http_gen.go`

**HTTP handlers:**

```go
// Code generated by bridgegen. DO NOT EDIT.

package {{.BridgePackage}}

import (
    "context"
    "net/http"

    "{{.ModulePath}}/bridge/scaffolding/errs"
    "{{.ModulePath}}/bridge/scaffolding/fopbridge"
    "{{.ModulePath}}/core/repositories/{{.RepoPackage}}"
    "{{.ModulePath}}/core/scaffolding/fop"
    "{{.ModulePath}}/infrastructure/web"
    "{{.ModulePath}}/sdk/logger"
)

type Config struct {
    Log        *logger.Logger
    Repository *{{.RepoPackage}}.Repository
    Middleware []web.Middleware
}

// AddHttpRoutes registers all HTTP routes for {{.EntityNamePlural}}
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
    bridge := newBridge(cfg.Repository)

    // Standard CRUD routes
    group.GET("/{{.HTTPBasePath}}", bridge.httpList)
    group.GET("/{{.HTTPBasePath}}/{{"{"}}{{.PKParam}}{{"}}}", bridge.httpGetByID)
    group.POST("/{{.HTTPBasePath}}", bridge.httpCreate)
    group.PUT("/{{.HTTPBasePath}}/{{"{"}}{{.PKParam}}{{"}}}", bridge.httpUpdate)
    group.DELETE("/{{.HTTPBasePath}}/{{"{"}}{{.PKParam}}{{"}}}", bridge.httpDelete)

{{- range .ForeignKeys}}
    // Foreign key relationship route
    group.GET("/{{.HTTPPathSegment}}", bridge.httpListBy{{.MethodSuffix}})
{{- end}}
}

// httpList handles GET requests for listing {{.EntityNamePlural}}
func (b *bridge) httpList(ctx context.Context, r *http.Request) web.Encoder {
    qp := parseQueryParams(r)

    page, err := fop.ParsePageStringCursor(qp.Limit, qp.Cursor)
    if err != nil {
        return errs.NewFieldErrors("page", err)
    }

    filter, err := parseFilter(qp)
    if err != nil {
        return errs.NewFieldErrors("filter", err)
    }

    orderBy := parseOrderBy(qp.Order)

    records, pageInfo, err := b.repository.List(ctx, filter, orderBy, page)
    if err != nil {
        return errs.Newf(errs.Internal, "list {{.EntityNamePlural}}: %s", err)
    }

    return fopbridge.NewPaginatedResultStringCursor(MarshalListToBridge(records), pageInfo)
}

// httpGetByID handles GET requests for a specific {{.EntityNameLower}}
func (b *bridge) httpGetByID(ctx context.Context, r *http.Request) web.Encoder {
    qpath, err := parsePath(r)
    if err != nil {
        return errs.Newf(errs.InvalidArgument, "invalid path: %s", err)
    }

    if qpath.{{.PKName}} == "" {
        return errs.Newf(errs.InvalidArgument, "{{.PKParam}} is required")
    }

    qp := parseQueryParams(r)
    filter, err := parseFilter(qp)
    if err != nil {
        return errs.NewFieldErrors("filter", err)
    }

    record, err := b.repository.Get(ctx, qpath.{{.PKName}}, filter)
    if err != nil {
        if err == {{.RepoPackage}}.ErrNotFound {
            return errs.Newf(errs.NotFound, "{{.EntityNameLower}} not found: %s", qpath.{{.PKName}})
        }
        return errs.Newf(errs.Internal, "get {{.EntityNameLower}}: %s", err)
    }

    return MarshalToBridge(record)
}

// httpCreate handles POST requests for creating a {{.EntityNameLower}}
func (b *bridge) httpCreate(ctx context.Context, r *http.Request) web.Encoder {
    var input Create{{.EntityName}}Input
    if err := web.Decode(r, &input); err != nil {
        return errs.Newf(errs.InvalidArgument, "decode input: %s", err)
    }

    payload := MarshalCreateToRepository(input)

    record, err := b.repository.Create(ctx, payload)
    if err != nil {
        return errs.Newf(errs.Internal, "create {{.EntityNameLower}}: %s", err)
    }

    return MarshalToBridge(record)
}

// httpUpdate handles PUT requests for updating a {{.EntityNameLower}}
func (b *bridge) httpUpdate(ctx context.Context, r *http.Request) web.Encoder {
    qpath, err := parsePath(r)
    if err != nil {
        return errs.Newf(errs.InvalidArgument, "invalid path: %s", err)
    }

    var input Update{{.EntityName}}Input
    if err := web.Decode(r, &input); err != nil {
        return errs.Newf(errs.InvalidArgument, "decode input: %s", err)
    }

    payload := MarshalUpdateToRepository(input)

    if err := b.repository.Update(ctx, qpath.{{.PKName}}, payload); err != nil {
        if err == {{.RepoPackage}}.ErrNotFound {
            return errs.Newf(errs.NotFound, "{{.EntityNameLower}} not found: %s", qpath.{{.PKName}})
        }
        return errs.Newf(errs.Internal, "update {{.EntityNameLower}}: %s", err)
    }

    // Return updated record
    record, _ := b.repository.Get(ctx, qpath.{{.PKName}}, {{.RepoPackage}}.QueryFilter{})
    return MarshalToBridge(record)
}

// httpDelete handles DELETE requests for removing a {{.EntityNameLower}}
func (b *bridge) httpDelete(ctx context.Context, r *http.Request) web.Encoder {
    qpath, err := parsePath(r)
    if err != nil {
        return errs.Newf(errs.InvalidArgument, "invalid path: %s", err)
    }

    if err := b.repository.Delete(ctx, qpath.{{.PKName}}); err != nil {
        if err == {{.RepoPackage}}.ErrNotFound {
            return errs.Newf(errs.NotFound, "{{.EntityNameLower}} not found: %s", qpath.{{.PKName}})
        }
        return errs.Newf(errs.Internal, "delete {{.EntityNameLower}}: %s", err)
    }

    return web.StatusNoContent
}

{{- range .ForeignKeys}}

// httpListBy{{.MethodSuffix}} handles GET requests for {{$.EntityNamePlural}} by {{.EntityName}}
func (b *bridge) httpListBy{{.MethodSuffix}}(ctx context.Context, r *http.Request) web.Encoder {
    qpath, err := parsePath(r)
    if err != nil {
        return errs.Newf(errs.InvalidArgument, "invalid path: %s", err)
    }

    if qpath.{{.PathParamName}} == "" {
        return errs.Newf(errs.InvalidArgument, "{{.PathParamName}} is required")
    }

    qp := parseQueryParams(r)
    page, err := fop.ParsePageStringCursor(qp.Limit, qp.Cursor)
    if err != nil {
        return errs.NewFieldErrors("page", err)
    }

    orderBy := parseOrderBy(qp.Order)

    records, pageInfo, err := b.repository.ListBy{{.MethodSuffix}}(ctx, qpath.{{.PathParamName}}, orderBy, page)
    if err != nil {
        return errs.Newf(errs.Internal, "list by {{.PathParamName}}: %s", err)
    }

    return fopbridge.NewPaginatedResultStringCursor(MarshalListToBridge(records), pageInfo)
}
{{- end}}
```

### 3.5 Generate `{entity}repobridge_gen.go`

**Bridge struct:**

```go
// Code generated by bridgegen. DO NOT EDIT.

package {{.BridgePackage}}

import "{{.ModulePath}}/core/repositories/{{.RepoPackage}}"

type bridge struct {
    repository *{{.RepoPackage}}.Repository
}

func newBridge(repository *{{.RepoPackage}}.Repository) *bridge {
    return &bridge{
        repository: repository,
    }
}
```

---

## Phase 4: Command Structure & Orchestration

### 4.1 Command Hierarchy

```
app/generators/main.go commands:

generator scaffold-from-sql   # Full end-to-end scaffold
generator repo-from-sql        # Repository layer only
generator pgxstore             # Store layer (enhanced with FK)
generator bridge               # Bridge layer
generator help                 # Show help
```

### 4.2 Scaffold Command Flow

```bash
generator scaffold-from-sql --sql=schema.sql --table=task_executions [--force] [--dry-run]
```

**Execution steps:**

1. Parse SQL table definition
2. Analyze schema (columns, PK, FKs, constraints)
3. Generate naming context
4. Check if repository exists
5. **If repository doesn't exist:**
   - Generate `model_gen.go`
   - Generate `fop_gen.go`
   - Generate `{entity}repo_gen.go`
   - Generate store scaffold `store.go`
6. Generate/update `store_gen.go`
7. Generate/update `fop_gen.go` (in store)
8. **If bridge doesn't exist:**
   - Generate `model_gen.go`
   - Generate `marshal_gen.go`
   - Generate `fop_gen.go`
   - Generate `http_gen.go`
   - Generate `{entity}repobridge_gen.go`
9. Print summary of generated files

### 4.3 Overwrite Protection

**Implementation:**

```go
type FileOperation struct {
    Path      string
    Content   string
    Exists    bool
    IsGenFile bool
}

func (g *Generator) WriteFiles(ops []FileOperation, force bool, dryRun bool) error {
    if dryRun {
        // Print what would be done
        for _, op := range ops {
            if op.Exists {
                fmt.Printf("Would overwrite: %s\n", op.Path)
            } else {
                fmt.Printf("Would create: %s\n", op.Path)
            }
        }
        return nil
    }

    // Check for overwrites
    var overwrites []string
    for _, op := range ops {
        if op.Exists && op.IsGenFile {
            overwrites = append(overwrites, op.Path)
        }
    }

    if len(overwrites) > 0 && !force {
        fmt.Println("The following generated files would be overwritten:")
        for _, path := range overwrites {
            fmt.Printf("  - %s\n", path)
        }
        fmt.Print("\nContinue? [y/N]: ")

        var response string
        fmt.Scanln(&response)

        if response != "y" && response != "Y" {
            return fmt.Errorf("aborted by user")
        }
    }

    // Write files
    for _, op := range ops {
        if err := os.MkdirAll(filepath.Dir(op.Path), 0755); err != nil {
            return err
        }
        if err := os.WriteFile(op.Path, []byte(op.Content), 0644); err != nil {
            return err
        }
    }

    return nil
}
```

### 4.4 Command Examples

```bash
# Full scaffold (creates everything)
generator scaffold-from-sql \
  --sql=infrastructure/postgresdb/migrate/sql/schema.sql \
  --table=task_executions

# Dry run (preview changes)
generator scaffold-from-sql \
  --sql=schema.sql \
  --table=task_executions \
  --dry-run

# Force overwrite without prompting
generator scaffold-from-sql \
  --sql=schema.sql \
  --table=task_executions \
  --force

# Repository only
generator repo-from-sql \
  --sql=schema.sql \
  --table=task_executions

# Store only (requires existing repo)
generator pgxstore \
  --entity=TaskExecution \
  --table=task_executions \
  --pk=execution_id

# Bridge only (requires existing repo)
generator bridge \
  --entity=TaskExecution \
  --repo=taskexecutionsrepo
```

---

## Phase 5: Implementation Roadmap

### Sprint 1: SQL Parser (Week 1)

- [ ] Create `sqlparser` package
- [ ] Implement basic CREATE TABLE parsing
- [ ] PostgreSQL type → Go type mapping
- [ ] FK detection and analysis
- [ ] Constraint parsing
- [ ] Unit tests for parser

**Deliverable:** Parse SQL → `TableSchema` struct

### Sprint 2: Repository Generator (Week 2)

- [ ] Create `repogen` package
- [ ] Naming convention utilities
- [ ] `model_gen.go` template + generator
- [ ] `fop_gen.go` template + generator
- [ ] `{entity}repo_gen.go` template + generator
- [ ] Store scaffold generator
- [ ] Unit tests

**Deliverable:** Generate complete repository layer from schema

### Sprint 3: Enhanced PGX Store (Week 3)

- [ ] Update `pgxstores` package
- [ ] FK method generation in `store_gen.go`
- [ ] `fop_gen.go` template for filters
- [ ] Integration with schema parser
- [ ] Unit tests

**Deliverable:** Generate store with FK methods and filters

### Sprint 4: Bridge Generator (Week 4)

- [ ] Create `bridgegen` package
- [ ] `model_gen.go` template (DTOs)
- [ ] `marshal_gen.go` template
- [ ] `fop_gen.go` template (query parsing)
- [ ] `http_gen.go` template
- [ ] Unit tests

**Deliverable:** Generate complete bridge layer

### Sprint 5: Command Integration (Week 5)

- [ ] Create `scaffold-from-sql` command
- [ ] Overwrite protection implementation
- [ ] Dry-run mode
- [ ] File operation utilities
- [ ] Integration tests
- [ ] Documentation

**Deliverable:** End-to-end SQL → API generation

### Sprint 6: Polish & Testing (Week 6)

- [ ] Error handling improvements
- [ ] Progress indicators
- [ ] Logging
- [ ] Edge case testing
- [ ] Documentation with examples
- [ ] Video walkthrough

**Deliverable:** Production-ready generator

---

## Success Criteria

### Functional Requirements

- ✅ Parse CREATE TABLE statements correctly
- ✅ Generate all `_gen` files without errors
- ✅ Generated code compiles without modification
- ✅ FK relationships create working endpoints
- ✅ Pagination works correctly
- ✅ Filters apply as expected
- ✅ Overwrite protection prevents data loss

### Quality Requirements

- ✅ Generated code follows project conventions
- ✅ No manual edits needed for basic CRUD
- ✅ Clear error messages for invalid SQL
- ✅ Dry-run shows accurate preview
- ✅ Documentation is comprehensive

### Performance Requirements

- ✅ Generation completes in < 5 seconds
- ✅ Parser handles complex schemas
- ✅ Memory efficient for large tables

---

## Future Enhancements

### Phase 6+ (Future)

- [ ] GraphQL schema generation
- [ ] gRPC service generation
- [ ] OpenAPI/Swagger spec generation
- [ ] Test generation
- [ ] Mock generation
- [ ] Migration file generation from schema diff
- [ ] Batch processing multiple tables
- [ ] Web UI for schema design
- [ ] Database introspection (reverse engineer from DB)

---

## Notes & Considerations

### Special Cases to Handle

**Composite Primary Keys:**
Currently assuming single-column PK. Need to handle:

```sql
PRIMARY KEY (tenant_id, resource_id)
```

**Self-Referential FKs:**

```sql
parent_id uuid REFERENCES categories(category_id)
```

Should generate `ListByParentID` but be careful with naming.

**Many-to-Many Join Tables:**

```sql
CREATE TABLE user_roles (
    user_id uuid REFERENCES users(user_id),
    role_id uuid REFERENCES roles(role_id),
    PRIMARY KEY (user_id, role_id)
);
```

May need special handling or skip generation.

**Enum Types:**

```sql
status task_status NOT NULL
```

Need to map to Go string with validation tags.

**Array Types:**

```sql
tags text[]
```

Map to `[]string` or `pq.StringArray`.

### Testing Strategy

**Unit Tests:**

- SQL parser with various CREATE TABLE formats
- Type mapping edge cases
- Naming convention edge cases
- Template rendering

**Integration Tests:**

- Full end-to-end generation
- Compile generated code
- Run generated tests
- HTTP endpoint smoke tests

**Manual Testing:**

- Complex real-world schemas
- Edge cases from production
- Performance with large schemas

---

## Getting Started

To begin implementation:

```bash
# 1. Create branch
git checkout -b feature/sql-scaffolding

# 2. Start with Sprint 1
cd app/generators
mkdir sqlparser
cd sqlparser

# 3. Create initial files
touch parser.go types.go mapper.go analyzer.go

# 4. Run tests as you go
go test ./...
```

---

**Last Updated:** {{ .Date }}
**Status:** Planning Phase
**Next Step:** Sprint 1 - SQL Parser
