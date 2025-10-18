# Schema Reflector

The schema reflector queries a live PostgreSQL database and generates database-agnostic schema metadata files. This allows code generators to work from the **actual database schema** rather than parsing SQL migration files.

## Directory Structure

```
schema/reflector/
├── README.md           # This file
├── types.go            # JSON schema format definitions
├── postgres.go         # PostgreSQL reflection queries
├── reflector.go        # Core reflection orchestrator
└── output/             # Generated artifacts (gitignored)
    ├── public.json     # JSON schema for 'public' schema
    ├── public.sql      # SQL dump for 'public' schema
    └── ...             # Additional schemas as needed
```

## Workflow

### 1. Run Migrations

First, apply your migrations to update the database schema:

```bash
go run app/tooling/main.go migrate
```

### 2. Reflect Schema

Query the live database and generate JSON/SQL artifacts:

```bash
go run app/tooling/main.go reflect-schema \
  --host=localhost \
  --port=5432 \
  --dbname=taskmaster \
  --user=postgres \
  --password=postgres \
  --schema=public \
  --output=schema/reflector/output
```

**Default values:**

- Most flags use environment variables (`DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`)
- `--schema` defaults to `public`
- `--output` defaults to `schema/reflector/output`

**Output files:**

- `{schema_name}.json` - Database-agnostic metadata (for code generation)
- `{schema_name}.sql` - Human-readable SQL dump (for documentation)

### 3. Generate Code

Use the reflected JSON schema to generate code:

```bash
# Generate all layers for a specific table
go run app/generators/main.go generate \
  -json=schema/reflector/output/public.json \
  -table=users \
  -force

# Generate only specific layers
go run app/generators/main.go generate \
  -json=schema/reflector/output/public.json \
  -table=tasks \
  -layers=repository,store
```

## JSON Schema Format

The reflected JSON schema is database-agnostic and contains:

```json
{
  "version": "1.0",
  "source": "postgres",
  "database": "taskmaster",
  "schema_name": "public",
  "reflected_at": "2025-10-17T...",
  "tables": {
    "users": {
      "table_name": "users",
      "primary_key": {
        "column": "user_id",
        "db_type": "uuid",
        "go_type": "string"
      },
      "columns": [...],
      "foreign_keys": [...],
      "indexes": [...],
      "constraints": [...]
    }
  }
}
```

## Benefits

1. **Truth Source**: Generators work from actual database schema, not migration files
2. **Drift Resilience**: If migrations drift from reality, code stays in sync with database
3. **Database Agnostic**: JSON format can represent any database (Postgres, Firestore, MongoDB)
4. **Multi-Schema**: Easily reflect multiple schemas (public, auth, analytics, etc.)
5. **Version Control**: JSON diffs show schema evolution over time

## Advanced Usage

### Reflect Multiple Schemas

```bash
# Reflect public schema
go run app/tooling/main.go reflect-schema --schema=public

# Reflect auth schema
go run app/tooling/main.go reflect-schema --schema=auth

# Generate code from different schemas
go run app/generators/main.go generate -json=schema/reflector/output/auth.json -table=sessions
```

### List Available Tables

To see what tables are available in a reflected schema:

```bash
cat schema/reflector/output/public.json | jq '.tables | keys'
```

### Compare Schemas

Track schema changes over time:

```bash
# Commit current schema
git add schema/reflector/output/public.json
git commit -m "Schema snapshot: 2025-10-17"

# After migrations, reflect again and diff
go run app/tooling/main.go reflect-schema
git diff schema/reflector/output/public.json
```

## Integration with CI/CD

You can automate schema reflection in your CI/CD pipeline:

```yaml
# Example GitHub Actions workflow
- name: Run Migrations
  run: go run app/tooling/main.go migrate

- name: Reflect Schema
  run: go run app/tooling/main.go reflect-schema

- name: Generate Code
  run: |
    for table in users tasks applications; do
      go run app/generators/main.go generate \
        -json=schema/reflector/output/public.json \
        -table=$table \
        -force
    done

- name: Commit Generated Code
  run: |
    git config user.name "CI Bot"
    git add .
    git commit -m "Regenerate code from schema"
```

## Troubleshooting

### Connection Issues

If you can't connect to the database, check:

1. Database is running: `docker compose ps`
2. Environment variables are set correctly
3. Network connectivity: `psql -h localhost -U postgres -d taskmaster`

### Missing Tables

If tables are missing from the JSON:

1. Verify schema name is correct (default: `public`)
2. Check table actually exists: `\dt` in psql
3. Ensure user has permissions: `GRANT SELECT ON ALL TABLES IN SCHEMA public TO postgres`

### Type Mapping Issues

If Go types are incorrect:

1. Check [postgres.go](postgres.go) `mapPostgreSQLTypeToGo()` function
2. Add custom mappings for your specific types
3. Update validation tag logic in `deriveValidationTags()`
