# Generator System Refactor Plan

**Date**: 2025-10-16
**Status**: Planning
**Scope**: Major refactor addressing parser bugs, FOP integration, and file generation strategy

---

## Executive Summary

This refactor addresses critical issues discovered during production use:

1. SQL parser treating comment lines as columns
2. Missing separation between generated and custom code
3. Incorrect FOP (Filter/Order/Pagination) implementation
4. Missing postgresdb helper usage in store layer
5. No support for Archive pattern with status columns

---

## 1. SQL Parser Bug Fix

### Problem

Parser treats SQL comment lines (starting with `--`) as column definitions, generating invalid struct fields:

```go
-- interface{} `json:"--" db:"--" validate:"required"`
```

### Root Cause

The `extractColumns()` function in [parser.go](app/generators/sqlparser/parser.go) doesn't filter out comment lines before processing.

### Solution

In `parseColumnDefinition()`, skip lines that start with `--` after trimming whitespace.

### Files Modified

- `app/generators/sqlparser/parser.go`

---

## 2. Repository Layer Changes

### 2.1 Separate Generated and Custom Code

**Problem**: Currently, `Storer` interface and `Repository` constructor are in `repository_gen.go`, which gets overwritten on regeneration. Users may want to add custom methods to the interface.

**Solution**: Move to `repo.go` and only generate if file doesn't exist.

#### File: `repo.go` (generate only if doesn't exist)

```go
package {entity}repo

import (
    "context"
    "github.com/jrazmi/envoker/core/scaffolding/logger"
    "github.com/jrazmi/envoker/core/scaffolding/fop"
)

// Storer defines the data storage interface for {Entity}
type Storer interface {
    Create(ctx context.Context, input Create{Entity}) ({Entity}, error)
    Get(ctx context.Context, {pkParam} {pkType}) ({Entity}, error)
    Update(ctx context.Context, {pkParam} {pkType}, input Update{Entity}) error
    Delete(ctx context.Context, {pkParam} {pkType}) error
    List(ctx context.Context, filter QueryFilter, orderBy fop.By, page fop.PageStringCursor, forPrevious bool) ([]Entity, error)
    // Archive sets status to 'archived' (only if status column exists)
    Archive(ctx context.Context, {pkParam} {pkType}) error
}

// Repository provides access to {entity} storage
type Repository struct {
    log    *logger.Logger
    storer Storer
}

// NewRepository creates a new {Entity} repository
func NewRepository(log *logger.Logger, storer Storer) *Repository {
    return &Repository{
        log:    log,
        storer: storer,
    }
}
```

### 2.2 Enhanced FOP Support in model_gen.go

**Problem**: Missing QueryFilter, OrderBy constants, and cursor helpers needed for proper pagination.

**Solution**: Add comprehensive FOP support to `model_gen.go`.

#### Additions to `model_gen.go`:

```go
// QueryFilter holds the available fields a query can be filtered on
type QueryFilter struct {
    SearchTerm      *string
    {EntityPK}      *{PKType}
    // For each filterable column:
    {Column}        *{GoType}
    // For timestamp columns:
    CreatedAtBefore *time.Time
    CreatedAtAfter  *time.Time
    UpdatedAtBefore *time.Time
    UpdatedAtAfter  *time.Time
    // For numeric columns (min/max pattern):
    Min{Column}     *{GoType}
    Max{Column}     *{GoType}
    // For nullable text columns:
    Has{Column}     *bool
}

// OrderBy constants for sorting
const (
    OrderByPK        = "{pk_column}"
    OrderByCreatedAt = "created_at"
    OrderByUpdatedAt = "updated_at"
    // For each sortable column:
    OrderBy{Column}  = "{db_column}"
)

// DefaultOrderBy specifies the default sort order
var DefaultOrderBy = fop.NewBy(OrderByCreatedAt, fop.DESC)

// {Entity}Cursor for cursor-based pagination
type {Entity}Cursor = fop.Cursor[{PKType}, time.Time]

func Decode{Entity}Cursor(token string) (*{Entity}Cursor, error) {
    return fop.DecodeCursor[{PKType}, time.Time](token)
}

func Encode{Entity}Cursor(createdAt time.Time, {pkParam} {PKType}) (string, error) {
    cursor := {Entity}Cursor{
        OrderValue: createdAt,
        PK:         {pkParam},
    }
    return cursor.Encode()
}
```

### Files Modified

- `app/generators/repositorygen/generator.go` - Generate `repo.go` conditionally
- `app/generators/repositorygen/template_repository.go` - Split into two templates
- `app/generators/repositorygen/template_model.go` - Add QueryFilter, OrderBy, cursor helpers

---

## 3. Store Layer Changes

### 3.1 Conditional store.go Generation

**Problem**: Store initialization code gets overwritten, preventing custom initialization logic.

**Solution**: Generate `store.go` only if it doesn't exist.

#### File: `store.go` (generate only if doesn't exist)

```go
package {entity}pgxstore

import (
    "github.com/jrazmi/envoker/core/scaffolding/logger"
    "github.com/jrazmi/envoker/infrastructure/postgresdb"
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
```

### 3.2 Create fop_gen.go for Filter/Order Logic

**Problem**: Filter and ordering logic is inlined in List functions, not reusable.

**Solution**: Create `fop_gen.go` with `orderByFields` map and `applyFilter()` function.

#### File: `fop_gen.go` (always regenerate)

```go
package {entity}pgxstore

import (
    "bytes"
    "strings"
    "github.com/jrazmi/envoker/core/repositories/{entity}repo"
    "github.com/jackc/pgx/v5"
)

// orderByFields maps repository field names to database column names
var orderByFields = map[string]string{
    {entity}repo.OrderByPK:        "{pk_column}",
    {entity}repo.OrderByCreatedAt: "created_at",
    {entity}repo.OrderByUpdatedAt: "updated_at",
    // For each sortable column:
    {entity}repo.OrderBy{Column}:  "{db_column}",
}

// applyFilter applies query filters to the SQL query
func (s *Store) applyFilter(filter {entity}repo.QueryFilter, data pgx.NamedArgs, buf *bytes.Buffer, aliases map[string]string) {
    var conditions []string

    if filter.{EntityPK} != nil {
        conditions = append(conditions, "{pk_column} = @{pk_param}")
        data["{pk_param}"] = *filter.{EntityPK}
    }

    // For each filterable column:
    if filter.{Column} != nil {
        conditions = append(conditions, "{db_column} = @{param}")
        data["{param}"] = *filter.{Column}
    }

    // For timestamp columns (before/after pattern):
    if filter.CreatedAtBefore != nil {
        conditions = append(conditions, "created_at < @created_at_before")
        data["created_at_before"] = *filter.CreatedAtBefore
    }

    if filter.CreatedAtAfter != nil {
        conditions = append(conditions, "created_at > @created_at_after")
        data["created_at_after"] = *filter.CreatedAtAfter
    }

    // For numeric columns (min/max pattern):
    if filter.Min{Column} != nil {
        conditions = append(conditions, "{db_column} >= @min_{param}")
        data["min_{param}"] = *filter.Min{Column}
    }

    if filter.Max{Column} != nil {
        conditions = append(conditions, "{db_column} <= @max_{param}")
        data["max_{param}"] = *filter.Max{Column}
    }

    // For nullable text columns (has pattern):
    if filter.Has{Column} != nil {
        if *filter.Has{Column} {
            conditions = append(conditions, "{db_column} IS NOT NULL AND {db_column} != ''")
        } else {
            conditions = append(conditions, "({db_column} IS NULL OR {db_column} = '')")
        }
    }

    // Search term (if applicable):
    if filter.SearchTerm != nil && *filter.SearchTerm != "" {
        searchPattern := "%" + *filter.SearchTerm + "%"
        // Build ILIKE conditions for text columns
        conditions = append(conditions, "({text_col1} ILIKE @search_term OR {text_col2} ILIKE @search_term)")
        data["search_term"] = searchPattern
    }

    // Apply conditions if any exist
    if len(conditions) > 0 {
        buf.WriteString(" WHERE ")
        buf.WriteString(strings.Join(conditions, " AND "))
    }
}
```

### 3.3 Refactor List Function

**Problem**: List functions manually implement cursor pagination instead of using postgresdb helpers.

**Solution**: Update signature and use helper functions.

#### Before (current):

```go
List(ctx context.Context, filter Filter{Entity}, fop fop.FOP) ([]Entity, *fop.Pagination, error)
```

#### After (new):

```go
List(ctx context.Context, filter {entity}repo.QueryFilter, orderBy fop.By, page fop.PageStringCursor, forPrevious bool) ([]{entity}repo.{Entity}, error)
```

#### Implementation Pattern:

```go
func (s *Store) List(ctx context.Context, filter {entity}repo.QueryFilter, orderBy fop.By, page fop.PageStringCursor, forPrevious bool) ([]{entity}repo.{Entity}, error) {
    data := pgx.NamedArgs{}
    aliases := map[string]string{}

    // Start building the query
    buf := bytes.NewBufferString(`
        SELECT
            {column_list}
        FROM
            {table_name}`)

    // Apply filters
    s.applyFilter(filter, data, buf, aliases)

    // Setup configuration for string cursor pagination
    cursorConfig := postgresdb.StringCursorConfig{
        Cursor:     page.Cursor,
        OrderField: orderByFields[orderBy.Field],
        PKField:    "{pk_column}",
        TableName:  "{table_name}",
        Direction:  orderBy.Direction,
        Limit:      page.Limit,
    }

    // Apply cursor pagination
    if page.Cursor != "" {
        err := postgresdb.ApplyStringCursorPagination(buf, data, cursorConfig, forPrevious)
        if err != nil {
            return nil, fmt.Errorf("cursorpagination: %s", err)
        }
    }

    // Add ordering
    err := postgresdb.AddOrderByClause(buf, cursorConfig.OrderField, cursorConfig.PKField, cursorConfig.Direction, forPrevious)
    if err != nil {
        return nil, fmt.Errorf("order: %w", err)
    }

    // Add limit
    postgresdb.AddLimitClause(cursorConfig.Limit, data, buf)

    // Execute the query
    query := buf.String()
    rows, err := s.pool.Query(ctx, query, data)
    if err != nil {
        return nil, postgresdb.HandlePgError(err)
    }
    defer rows.Close()

    entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[{entity}repo.{Entity}])
    if err != nil {
        return nil, postgresdb.HandlePgError(err)
    }

    // If we were getting previous page, reverse the results back to correct order
    if forPrevious && len(entities) > 0 {
        for i, j := 0, len(entities)-1; i < j; i, j = i+1, j-1 {
            entities[i], entities[j] = entities[j], entities[i]
        }
    }

    return entities, nil
}
```

### 3.4 Update Function Improvements

**Problem**: Update doesn't automatically set `updated_at`, and may generate SQL with trailing comma if no fields to update.

**Solution**: Always update `updated_at` with optional override, validate at least one field changes.

#### Implementation:

```go
func (s *Store) Update(ctx context.Context, {pkParam} {PKType}, input {entity}repo.Update{Entity}) error {
    data := pgx.NamedArgs{"{pk_param}": {pkParam}}
    var fields []string

    // ... existing field checks ...

    // Always update the updated_at field
    now := time.Now().UTC()
    if input.UpdatedAt != nil {
        data["updated_at"] = *input.UpdatedAt
    } else {
        data["updated_at"] = now
    }
    fields = append(fields, "updated_at = @updated_at")

    // If no fields to update besides updated_at, return early
    if len(fields) == 1 {
        return fmt.Errorf("no fields to update")
    }

    // Join fields and complete the query
    buf.WriteString(strings.Join(fields, ", "))
    buf.WriteString(` WHERE "{pk_column}" = @{pk_param}`)

    query := buf.String()
    s.log.DebugContext(ctx, "update {entity}", "query", query, "{pk_param}", {pkParam})

    result, err := s.pool.Exec(ctx, query, data)
    if err != nil {
        return postgresdb.HandlePgError(err)
    }

    if result.RowsAffected() == 0 {
        return fmt.Errorf("{entity} not found")
    }

    return nil
}
```

### 3.5 Archive Function

**Problem**: No support for soft-delete/archive pattern when tables have `status` column.

**Solution**: Generate `Archive()` method when status column is detected.

#### Implementation:

```go
// Archive sets the status to 'archived' and optionally sets deleted_at
func (s *Store) Archive(ctx context.Context, {pkParam} {PKType}) error {
    data := pgx.NamedArgs{
        "{pk_param}": {pkParam},
        "status": "archived",
    }

    query := `UPDATE {table_name} SET status = @status`

    // If deleted_at column exists, set it too
    {{- if .HasDeletedAt}}
    data["deleted_at"] = time.Now().UTC()
    query += `, deleted_at = @deleted_at`
    {{- end}}

    query += ` WHERE {pk_column} = @{pk_param}`

    result, err := s.pool.Exec(ctx, query, data)
    if err != nil {
        return postgresdb.HandlePgError(err)
    }

    if result.RowsAffected() == 0 {
        return fmt.Errorf("{entity} not found")
    }

    return nil
}
```

### Files Modified

- `app/generators/pgxstores/generator_sql.go` - Generate `store.go` and `fop_gen.go`
- `app/generators/pgxstores/template_sql.go` - Update List, Update, add Archive
- Create `app/generators/pgxstores/template_fop.go` - New file for fop_gen.go
- Create `app/generators/pgxstores/template_store.go` - New file for store.go

---

## 4. Bridge Layer Changes

### 4.1 Conditional bridge.go Generation

**Problem**: Bridge constructor gets overwritten, preventing custom initialization logic.

**Solution**: Move to `bridge.go` and only generate if file doesn't exist.

#### File: `bridge.go` (generate only if doesn't exist)

```go
package {entity}repobridge

import "github.com/jrazmi/envoker/core/repositories/{entity}repo"

type bridge struct {
    {entity}Repository *{entity}repo.Repository
}

func newBridge({entity}Repository *{entity}repo.Repository) *bridge {
    return &bridge{
        {entity}Repository: {entity}Repository,
    }
}
```

### Files Modified

- `app/generators/bridgegen/generator.go` - Generate `bridge.go` conditionally
- `app/generators/bridgegen/template_bridge.go` - Split into two templates

---

## 5. Validation Requirements

### 5.1 Timestamp Validation

**Problem**: No validation that tables have required `created_at` and `updated_at` columns.

**Solution**: Add validation in analyzer that warns if timestamps are missing.

#### Implementation:

```go
func ValidateTimestamps(schema *TableSchema) []string {
    var warnings []string

    hasCreatedAt := false
    hasUpdatedAt := false

    for _, col := range schema.Columns {
        if col.Name == "created_at" {
            hasCreatedAt = true
        }
        if col.Name == "updated_at" {
            hasUpdatedAt = true
        }
    }

    if !hasCreatedAt {
        warnings = append(warnings, "table missing 'created_at' timestamp column")
    }
    if !hasUpdatedAt {
        warnings = append(warnings, "table missing 'updated_at' timestamp column")
    }

    return warnings
}
```

### Files Modified

- `app/generators/sqlparser/analyzer.go` - Add ValidateTimestamps()

---

## 6. Implementation Order

### Phase 1: Parser Fix (CRITICAL - blocks everything)

1. Fix comment line handling in `parser.go`
2. Test with applications.sql
3. Verify no `-- interface{}` fields

### Phase 2: Repository Layer

1. Create `template_repo.go` for repo.go template
2. Update `generator.go` to generate repo.go conditionally
3. Add QueryFilter, OrderBy, cursors to `template_model.go`
4. Update repository method signatures
5. Test generation

### Phase 3: Store Layer

1. Create `template_store.go` for store.go template
2. Create `template_fop.go` for fop_gen.go template
3. Update `template_sql.go` for new List signature
4. Update Update function for updated_at handling
5. Add Archive function generation
6. Update generator to create all files
7. Test generation

### Phase 4: Bridge Layer

1. Split `template_bridge.go` into two templates
2. Update generator for conditional bridge.go
3. Test generation

### Phase 5: Validation

1. Add timestamp validation
2. Add status column detection
3. Test with various schemas

### Phase 6: Integration Testing

1. Test with applications.sql
2. Test with tasks.sql
3. Test with task_executions.sql
4. Verify all generated code compiles
5. Verify no overwrites of custom files

---

## 7. Breaking Changes

### For Existing Users

#### Repository Layer

- **Breaking**: `List()` signature changed from `(ctx, filter, fop.FOP)` to `(ctx, filter QueryFilter, orderBy fop.By, page fop.PageStringCursor)`
- **Breaking**: `Filter{Entity}` renamed to `QueryFilter` and moved to model_gen.go
- **Migration**: Update calls to List() to use new signature

#### Store Layer

- **Breaking**: `List()` signature changed to match repository
- **Breaking**: Store constructor moved to store.go (regenerate with `make generate-store`)
- **Migration**: Update store initialization code

#### Bridge Layer

- **Breaking**: Bridge constructor moved to bridge.go (regenerate with `make generate-bridge`)
- **Migration**: Update bridge initialization code

---

## 8. Testing Strategy

### Unit Tests

- [ ] Parser correctly skips comment lines
- [ ] Parser correctly extracts all columns from applications.sql
- [ ] Timestamp validation detects missing columns
- [ ] Status column detection works correctly

### Integration Tests

- [ ] Generate from applications.sql (18 columns)
- [ ] Generate from tasks.sql (existing test)
- [ ] Generate from task_executions.sql (existing test)
- [ ] Verify all generated Go code compiles
- [ ] Verify repo.go not overwritten on second run
- [ ] Verify store.go not overwritten on second run
- [ ] Verify bridge.go not overwritten on second run

### Performance Tests

- [ ] Generation time remains under 10ms per table
- [ ] No memory leaks in template execution

---

## 9. Documentation Updates

### Files to Update

- `README.md` - Update List signature examples
- `QUICKSTART.md` - Update tutorial with new signatures
- `SUMMARY.md` - Note breaking changes
- `schema/EXAMPLES.md` - Add timestamp requirements

### New Documentation

- `MIGRATION.md` - Guide for upgrading from old generator
- `FOP.md` - Deep dive on Filter/Order/Pagination patterns

---

## 10. Success Criteria

- [ ] Parser extracts all 18 columns from applications.sql correctly
- [ ] No `-- interface{}` fields in generated code
- [ ] repo.go, store.go, bridge.go not overwritten when they exist
- [ ] List functions use postgresdb helpers
- [ ] Update always sets updated_at
- [ ] Archive function generated when status column exists
- [ ] All generated code compiles without errors
- [ ] All existing tests pass
- [ ] Generation time remains fast (<10ms per table)
- [ ] Documentation reflects all changes

---

## Estimated Effort

- **Parser Fix**: 15 minutes
- **Repository Layer**: 45 minutes
- **Store Layer**: 60 minutes
- **Bridge Layer**: 20 minutes
- **Validation**: 15 minutes
- **Testing**: 30 minutes
- **Documentation**: 30 minutes

**Total**: ~3.5 hours

---

## Notes

- This is a **major refactor** touching all three layers
- Some breaking changes are unavoidable for correctness
- Priority is on correctness and maintainability over backward compatibility
- Generated code should follow established patterns from existing taskmaster codebase
- All custom code (repo.go, store.go, bridge.go) must be protected from overwrites
