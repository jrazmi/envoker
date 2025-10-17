# Bridge Layer Refactor Plan

## Overview
The bridge layer generation has several issues that need to be fixed to match the working hand-written examples and ensure proper compilation.

## Current Issues

### 1. **fop_gen.go Issues**

#### a. Query Parameters Structure
**Problem**: Using mixed pointer/value semantics inconsistently
```go
// Current (WRONG)
type queryParams struct {
    Limit  int              // Should be string
    Cursor string
    Order  string
    TaskId *string          // Pointer - unnecessary
    Status *string          // Pointer - unnecessary
}
```

**Solution**: Use strings for everything that comes from query params
```go
// Correct
type queryParams struct {
    Limit  string           // Parse to int when needed
    Cursor string
    Order  string
    // Filter fields - all strings from query
    TaskId string
    Status string
    Priority string          // Parse to int when converting to filter
    CreatedAtBefore string   // Parse to time.Time when converting
}
```

**Reasoning**:
- HTTP query params are always strings
- Parsing happens in `parseFilter()`, not `parseQueryParams()`
- Simpler to work with - no pointer dereferencing needed
- Matches the working example

---

#### b. Filter Naming
**Problem**: Filter struct is named `Filter{Entity}`
```go
func parseFilter(qp queryParams) (FilterTaskExecution, error)
```

**Solution**: Should be `{Entity}Filter` to match repository layer
```go
func parseFilter(qp queryParams) (TaskExecutionFilter, error)
```

**Reasoning**: Consistency with repository filter naming

---

#### c. parseFilter Implementation
**Problem**: Doesn't parse strings to proper types or handle errors
```go
// Current (WRONG)
func parseFilter(qp queryParams) (FilterTask, error) {
    filter := FilterTask{
        TaskId: qp.TaskId,  // Just assigns pointer directly
    }
    return filter, nil
}
```

**Solution**: Parse and validate each field type
```go
// Correct
func parseFilter(qp queryParams) (tasksrepo.TaskFilter, error) {
    filter := tasksrepo.TaskFilter{}

    // String filters - simple assignment if not empty
    if qp.SearchTerm != "" {
        filter.SearchTerm = &qp.SearchTerm
    }
    if qp.TaskId != "" {
        filter.TaskId = &qp.TaskId
    }

    // Time filters - parse RFC3339
    if qp.CreatedAtBefore != "" {
        if t, err := time.Parse(time.RFC3339, qp.CreatedAtBefore); err == nil {
            filter.CreatedAtBefore = &t
        } else {
            return filter, fmt.Errorf("invalid createdAtBefore format: %s", qp.CreatedAtBefore)
        }
    }

    // Integer filters - parse to int
    if qp.Priority != "" {
        if val, err := strconv.Atoi(qp.Priority); err == nil {
            filter.Priority = &val
        } else {
            return filter, fmt.Errorf("invalid priority: %s", qp.Priority)
        }
    }

    return filter, nil
}
```

**Reasoning**:
- Need proper error handling for invalid formats
- Repository filter expects pointers to actual types, not pointers to strings
- Must parse strings to time.Time, int, etc.

---

#### d. parseOrderBy Implementation
**Problem**: Returns wrong type and doesn't validate fields
```go
// Current (WRONG)
func parseOrderBy(order string) fop.OrderBy {
    return fop.OrderBy{
        Field: order,  // No validation
        Desc: false,   // Doesn't parse direction
    }
}
```

**Solution**: Return `fop.By` and use orderByFields map
```go
// Correct
var orderByFields = map[string]string{
    "task_id":     tasksrepo.OrderByPK,
    "created_at":  tasksrepo.OrderByCreatedAt,
    "updated_at":  tasksrepo.OrderByUpdatedAt,
    "status":      tasksrepo.OrderByStatus,
}

func parseOrderBy(order string) fop.By {
    if order == "" {
        return tasksrepo.DefaultOrderBy
    }

    orderBy, err := fop.ParseOrder(orderByFields, order, tasksrepo.DefaultOrderBy)
    if err != nil {
        return tasksrepo.DefaultOrderBy
    }

    return orderBy
}
```

**Reasoning**:
- Need to validate order fields against allowed fields
- Must return `fop.By` to match repository.List() signature
- Should use repository's OrderBy constants for type safety

---

### 2. **http_gen.go Issues**

#### a. Route Path Templates
**Problem**: Using Go template syntax in route paths
```go
// Current (WRONG)
group.GET("/task-executions/{{.execution_id}}", b.httpGetByID)
```

**Solution**: Use standard Go route parameter syntax
```go
// Correct
group.GET("/task-executions/{execution_id}", b.httpGetByID)
```

**Reasoning**: `{{.}}` is Go template syntax, not HTTP route syntax

---

#### b. Limit Parsing
**Problem**: Passing int to function expecting string
```go
// Current (WRONG)
page, err := fop.ParsePageStringCursor(qp.Limit, qp.Cursor)
// qp.Limit is int, function wants string
```

**Solution**: Keep Limit as string in queryParams
```go
// Correct
page, err := fop.ParsePageStringCursor(qp.Limit, qp.Cursor)
// qp.Limit is already string
```

---

#### c. parseFilter Error Handling
**Problem**: Not checking parseFilter error
```go
// Current (WRONG)
filter := MarshalFilterToRepository(parseFilter(qp))
// parseFilter returns (filter, error) but error is ignored
```

**Solution**: Handle the error
```go
// Correct
filter, err := parseFilter(qp)
if err != nil {
    return errs.NewFieldErrors("filter", err)
}
```

**Reasoning**: parseFilter can return validation errors that must be handled

---

#### d. Update Return Value
**Problem**: Expecting Update to return a record
```go
// Current (WRONG)
record, err := b.taskExecutionRepository.Update(ctx, qpath.ExecutionId, updateInput)
```

**Solution**: Update returns only error
```go
// Correct
err = b.taskExecutionRepository.Update(ctx, qpath.ExecutionId, updateInput)
if err != nil {
    return errs.Newf(errs.Internal, "update taskExecution: %s", err)
}

return fopbridge.CodeResponse{
    Code:    errs.OK.String(),
    Message: "TaskExecution updated successfully",
}
```

---

#### e. Foreign Key Method Naming
**Problem**: Wrong method name for FK lookups
```go
// Current (WRONG)
b.taskExecutionRepository.ListTaskId(...)  // Method doesn't exist
```

**Solution**: Use correct method name
```go
// Correct
b.taskExecutionRepository.ListByTaskId(...)  // Matches generated FK method
```

---

### 3. **marshal_gen.go Issues**

#### Filter Type Name
**Problem**: Using `Filter{Entity}` instead of `{Entity}Filter`
```go
// Current (WRONG)
func MarshalFilterToRepository(filter FilterTaskExecution) taskexecutionsrepo.FilterTaskExecution
```

**Solution**: Match repository naming
```go
// Correct
func MarshalFilterToRepository(filter TaskExecutionFilter) taskexecutionsrepo.TaskExecutionFilter
```

---

### 4. **model_gen.go Issues**

#### Filter Struct Naming
**Problem**: Filter struct named `Filter{Entity}`
```go
// Current (WRONG)
type FilterTaskExecution struct {
    ...
}
```

**Solution**: Name it `{Entity}Filter`
```go
// Correct
type TaskExecutionFilter struct {
    ...
}
```

---

## Implementation Plan

### Phase 1: Update template_fop.go ✅ COMPLETE
1. ✅ Change queryParams to use all strings
2. ✅ Fix parseFilter to:
   - ✅ Return repository filter type (not bridge filter)
   - ✅ Parse strings to proper types (time.Time, int, etc.)
   - ✅ Handle validation errors
3. ✅ Fix parseOrderBy to:
   - ✅ Return `fop.By`
   - ✅ Use orderByFields map
   - ✅ Call fop.ParseOrder()
4. ✅ Add orderByFields map generation

### Phase 2: Update template_http.go ✅ COMPLETE
1. ✅ Fix route paths (remove {{.}} syntax)
2. ✅ Fix Limit usage (string not int - no changes needed, queryParams.Limit already string)
3. ✅ Add parseFilter error handling
4. ✅ Fix Update to not expect return value
5. ✅ Fix FK method names (ListBy{FK} not List{FK})

### Phase 3: Update template_model.go ✅ COMPLETE
1. ✅ Remove bridge filter struct entirely (not needed - parseFilter returns repo filter directly)

### Phase 4: Update template_marshal.go ✅ COMPLETE
1. ✅ Remove MarshalFilterToRepository function (not needed)

### Phase 5: Test and Verify ✅ COMPLETE
1. ✅ Regenerate all bridge code
2. ✅ Compile all repositories - ALL PASS
3. ✅ Verify against working examples - Routes, filters, and methods all correct

---

## REFACTOR COMPLETE ✅

All phases completed successfully:
- Query params now use strings throughout
- **Query param keys use snake_case** (e.g., `rate_limit_requests_per_minute` not `rateLimitRequestsPerMinute`)
- parseFilter returns repository filter directly with proper validation
- parseOrderBy uses fop.By with orderByFields map
- Route paths use correct `{param}` syntax
- Update method returns CodeResponse (no record)
- FK methods use correct `ListBy{FK}` naming
- Bridge filter struct removed (not needed)
- MarshalFilterToRepository removed (not needed)
- PKs and JSONB fields excluded from filters
- Error messages use snake_case field names
- All code compiles successfully

---

## Key Decisions

### Query Parameter Philosophy
**Decision**: Query params stay as strings until converted to repository filter
**Reasoning**:
- HTTP query params are strings by nature
- Parsing/validation happens in one place (parseFilter)
- Error handling is centralized
- Matches working hand-written code

### Filter Location
**Decision**: parseFilter returns repository filter directly (not bridge filter)
**Reasoning**:
- Simpler - no intermediate bridge filter needed
- Less marshaling code
- Matches working examples

### Type Naming Convention
**Decision**: Use `{Entity}Filter` not `Filter{Entity}`
**Reasoning**:
- Consistency with repository layer
- Common Go naming pattern (TaskFilter, UserFilter, etc.)
- Easier to read and autocomplete

---

## Files to Modify

1. `/app/generators/bridgegen/template_fop.go` - Major rewrite
2. `/app/generators/bridgegen/template_http.go` - Several fixes
3. `/app/generators/bridgegen/template_model.go` - Filter naming
4. `/app/generators/bridgegen/template_marshal.go` - Filter naming
5. `/app/generators/bridgegen/generator.go` - May need data structure updates

---

## Testing Strategy

After each phase:
1. Run `make generate`
2. Check compilation errors
3. Compare generated code to working examples
4. Verify key differences are intentional

Final test:
1. Generate all three entities (applications, tasks, task_executions)
2. Ensure all compile
3. Verify HTTP handlers match working patterns
4. Check filter parsing logic is complete
