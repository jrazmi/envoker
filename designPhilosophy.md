# Go Project Organization Rules

## 1. Clean/Hexagonal Architecture Implementation

### App Layer (/app)

- Contains application entry points and executable applications
- Configures and wires together all dependencies through dependency injection
- Sets up HTTP servers, middleware, and telemetry
- Loads environment configurations
- Examples: `/app/api/`, `/app/worker-*/`

### Bridge Layer (/bridge)

- Adapts core layer to external concerns (HTTP, gRPC, etc.)
- Contains HTTP routes and handlers
- Marshals between domain models and external protocol models (JSON, protobuf)
- Organized by domain component, mirroring the core structure
- Bridge implementations append "bridge" to core component names (e.g., `marketsrepobridge`)

### Core Layer (/core)

- Contains domain models, business logic, and interfaces
- Defines use cases in `/core/cases/` for complex business logic
- Defines repositories in `/core/repositories/` for data access
- Defines shared utilities in `/core/scaffolding/`
- No dependencies on external frameworks or infrastructure
- All interfaces follow the `-er` pattern (e.g., `Storer`, `Searcher`)

### Infrastructure Layer (/infrastructure)

- Contains external system integrations (databases, web framework, etc.)
- Database connection management and migrations
- Web framework implementation
- External service clients
- Infrastructure logging should take in a default logger.

### SDK Layer (/sdk)

- Pure utilities independent of business logic
- Framework-like components (`logger`, `cryptids`, `environment`, etc.)
- Reusable across multiple applications or domains
- No domain-specific knowledge

### Vendor (/vendor)

- External Go modules managed by `go mod vendor`
- Populated via `make tidy`

### Wraps (/wraps)

- Deployment and infrastructure configurations
- Docker files, Kubernetes manifests, deployment scripts
- Language and deployment environment agnostic

## 2. Directory Specifications

### SDK Layer (/sdk)

#### Logger (/sdk/logger)

- Structured logging with context support
- Configurable output formats and levels
- Context extractors for automatic field inclusion (user_id, request_id)
- Infrastructure logging should take in a slog logger if logging there is ABSOLUTELY necessary, generally logging should be handled in above. e.g. an http.Server has a spot for an Error.log on the language level.
- Core and above should take in the sdk logger and use the helper functions.

#### Cryptids (/sdk/cryptids)

- Secure ID generation utilities
- Cryptographically secure random string generation
- URL-safe alphabet support

#### Environment (/sdk/environment)

- Environment variable loading and parsing
- Struct tag-based configuration binding
- Support for prefixed environment variables

### Core Domain Organization (/core)

#### Scaffolding (/core/scaffolding)

- **FOP (Filter, Order, Pagination) (/core/scaffolding/fop)**
  - String cursor-based pagination system
  - Standardized filtering and ordering interfaces
  - `PageStringCursor` and `PageInfoStringCursor` types
  - Order parsing with field mappings

#### Repository Design (/core/repositories)

Each repository follows a consistent pattern:

- **Package Structure**:

  - `model.go`: Domain models and CRUD input/output types
  - `fop.go`: Query filters, order constants, and pagination defaults
  - `{domain}repo.go`: Repository implementation with business logic
  - `stores/{domain}pgxstore/`: Database store implementation

- **Standard Storer Interface**:

  ```go
  type Storer interface {
      Create(ctx context.Context, payload CreateResource) (Resource, error)
      Get(ctx context.Context, ID string, filter QueryFilter) (Resource, error)
      List(ctx context.Context, filter QueryFilter, orderBy fop.By, page fop.PageStringCursor, forPrevious bool) ([]Resource, error)
      Update(ctx context.Context, ID string, payload UpdateResource) error
      Delete(ctx context.Context, ID string) error
  }
  ```

- **Standard Repository Methods**:

  - `GetByID(ctx, ID)`: Simple retrieval by ID
  - `Get(ctx, ID, filter)`: Retrieval with additional filtering
  - `List(ctx, filter, order, page)`: Paginated listing with filtering and sorting
  - `Create(ctx, payload)`: Creation with auto-generated ID if not provided
  - `Update(ctx, ID, payload)`: Partial updates using pointer fields
  - `Delete(ctx, ID)`: Hard deletion

- **ID Generation**:

  - Uses `cryptids.GenerateID()` for secure, URL-safe IDs
  - IDs generated if not provided in create payloads

- **Pagination**:

  - String cursor-based pagination using `fop.PageStringCursor`
  - Supports both forward and backward navigation
  - Uses primary key for cursor values

- **Filtering**:

  - Comprehensive `QueryFilter` structs with pointer fields
  - Time-based filtering with before/after patterns
  - Search term support where applicable

- **Timestamp Handling**:
  - Explicit timestamp parameters in repository methods
  - UTC standardization at repository boundaries
  - Automatic `updated_at` handling in store layer

### Bridge Pattern (Ports & Adapters)

#### Bridge Components (/bridge/repositories)

- **Naming Convention**: Append "bridge" to core domain names

  - Example: `marketsrepo` → `marketsrepobridge`

- **File Structure**:

  - `{domain}repobridge.go`: Main bridge implementation with handlers
  - `routes.go`: HTTP route configuration
  - `marshal.go`: Data transformation between domain and HTTP models
  - `parse.go`: Request parsing and validation

- **HTTP Handlers**:

  - `list`: GET with pagination, filtering, and sorting
  - `getByID`: GET single resource by ID
  - `create`: POST for resource creation
  - `update`: PUT for resource updates
  - `delete`: DELETE for resource removal

- **Request Parsing**:
  - Query parameter parsing for filters, pagination, and ordering
  - Path parameter extraction
  - Request body decoding with validation

#### Bridge Scaffolding (/bridge/scaffolding)

- **Error Handling (/bridge/scaffolding/errs)**: Standardized HTTP error responses
- **FOP Bridge (/bridge/scaffolding/fopbridge)**: Pagination response formatting
- **Middleware (/bridge/scaffolding/mid)**: HTTP middleware for logging, panics, etc.

### Applications (/app)

#### API Application (/app/api)

- HTTP REST API server
- Dependency injection and configuration
- Route registration and middleware setup
- Graceful shutdown handling

#### Worker Applications (/app/worker-\*)

- Single-purpose background workers
- Interval-based execution model
- Same dependency patterns as API applications
- Examples: document processors, notification senders

## 3. General Design Philosophies

### 3a. Dependency Injection

- Constructor functions accept all dependencies explicitly
- Configuration structs organize related dependencies
- Clear initialization order in main.go

### 3b. Error Handling

- Domain-specific error variables (e.g., `ErrNotFound`)
- Consistent error wrapping with context
- Centralized error handling in middleware

### 3c. Context Management

- Context-first design for all operations
- Request-scoped values carried through context
- Structured logging with automatic context extraction

### 3d. Package Naming

- Lowercase, no underscores (Go conventions)
- Domain-specific names (e.g., `marketsrepo`, `marketspgxstore`)
- Bridge packages append "bridge" suffix

### 3e. File Organization

- `model.go`: Data structures and types
- Domain-specific implementation files
- `routes.go`: HTTP routing configuration
- Consistent patterns across all domains

## 4. Data Access Patterns

### Store Implementation

- PostgreSQL stores using pgx driver
- Named parameters for SQL safety
- Comprehensive filtering and pagination support
- Connection pooling and transaction support

### Pagination Strategy

- String cursor-based pagination for performance
- Bidirectional navigation support
- Consistent ordering for reliable cursors
- Page size limits (default 20, max 100)

### Query Building

- Dynamic WHERE clause construction
- Parameterized queries for security
- Flexible filtering with optional fields
- ORDER BY with configurable directions

## 5. Testing Patterns

### Integration Testing

- Testcontainers for database testing
- Migration verification in tests
- Comprehensive CRUD operation testing
- Edge case and error condition testing

### Test Organization

- One test file per store implementation
- Helper functions for common test data
- Isolated test environments
- Cleanup and resource management

## 6. Configuration Management

### Environment Variables

- Struct tag-based configuration binding
- Prefix support for namespacing
- Default value fallbacks
- Type-safe parsing (string, int, bool, duration)

### Application Startup

- Environment loading from .env files
- Database connection establishment
- Telemetry and monitoring setup
- Graceful shutdown handling

# Worker Architecture

Workers are the execution units of our system. All workers implement the base `Worker` interface but differ in how they source and process work:

## Poller Workers

- **When**: Periodic, scheduled, or self-managed tasks
- **How**: Actively seeks work on intervals
- **Examples**: Data sync, report generation, maintenance tasks

## Event Workers

- **When**: Reactive, high-frequency, real-time processing
- **How**: Responds to external events/messages
- **Examples**: Order processing, webhook handling, notifications

## Orchestrator Workers

- **When**: Multi-step workflows with dependencies
- **How**: Manages state and coordinates multiple operations
- **Examples**: ETL pipelines, deployment workflows, complex business processes
