package sqlparser

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	sql := `
CREATE TABLE public.task_executions (
    execution_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id uuid NOT NULL,
    status varchar(50) NOT NULL DEFAULT 'pending',
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    error_message text,
    retry_count integer DEFAULT 0,
    metadata jsonb,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    FOREIGN KEY (task_id) REFERENCES public.tasks (task_id) ON DELETE CASCADE
);
`

	result, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify table name
	if result.Schema.Name != "task_executions" {
		t.Errorf("Expected table name 'task_executions', got '%s'", result.Schema.Name)
	}

	// Verify schema
	if result.Schema.Schema != "public" {
		t.Errorf("Expected schema 'public', got '%s'", result.Schema.Schema)
	}

	// Verify columns
	expectedColumns := []string{
		"execution_id", "task_id", "status", "started_at", "completed_at",
		"error_message", "retry_count", "metadata", "created_at", "updated_at",
	}

	if len(result.Schema.Columns) != len(expectedColumns) {
		t.Errorf("Expected %d columns, got %d", len(expectedColumns), len(result.Schema.Columns))
	}

	// Verify primary key
	if result.Schema.PrimaryKey.ColumnName != "execution_id" {
		t.Errorf("Expected primary key 'execution_id', got '%s'", result.Schema.PrimaryKey.ColumnName)
	}

	if !result.Schema.PrimaryKey.HasDefault {
		t.Error("Expected primary key to have default")
	}

	// Verify foreign key
	if len(result.Schema.ForeignKeys) != 1 {
		t.Fatalf("Expected 1 foreign key, got %d", len(result.Schema.ForeignKeys))
	}

	fk := result.Schema.ForeignKeys[0]
	if fk.ColumnName != "task_id" {
		t.Errorf("Expected FK column 'task_id', got '%s'", fk.ColumnName)
	}
	if fk.RefTable != "tasks" {
		t.Errorf("Expected FK ref table 'tasks', got '%s'", fk.RefTable)
	}
	if fk.RefColumn != "task_id" {
		t.Errorf("Expected FK ref column 'task_id', got '%s'", fk.RefColumn)
	}
	if fk.OnDelete != "CASCADE" {
		t.Errorf("Expected FK on delete 'CASCADE', got '%s'", fk.OnDelete)
	}

	// Verify specific column details
	for _, col := range result.Schema.Columns {
		switch col.Name {
		case "execution_id":
			if col.DBType != "uuid" {
				t.Errorf("execution_id: expected type 'uuid', got '%s'", col.DBType)
			}
			if !col.IsPrimaryKey {
				t.Error("execution_id: should be primary key")
			}
			if col.IsNullable {
				t.Error("execution_id: should not be nullable")
			}

		case "task_id":
			if col.DBType != "uuid" {
				t.Errorf("task_id: expected type 'uuid', got '%s'", col.DBType)
			}
			if !col.IsForeignKey {
				t.Error("task_id: should be foreign key")
			}
			if col.IsNullable {
				t.Error("task_id: should not be nullable")
			}

		case "status":
			if !strings.HasPrefix(col.DBType, "varchar") {
				t.Errorf("status: expected type 'varchar', got '%s'", col.DBType)
			}
			if col.IsNullable {
				t.Error("status: should not be nullable")
			}
			if !col.HasDefault {
				t.Error("status: should have default")
			}

		case "retry_count":
			if col.DBType != "integer" {
				t.Errorf("retry_count: expected type 'integer', got '%s'", col.DBType)
			}
			if !col.HasDefault {
				t.Error("retry_count: should have default")
			}

		case "metadata":
			if col.DBType != "jsonb" {
				t.Errorf("metadata: expected type 'jsonb', got '%s'", col.DBType)
			}
			if col.GoType != "*json.RawMessage" {
				t.Errorf("metadata: expected Go type '*json.RawMessage', got '%s'", col.GoType)
			}
			if col.GoImportPath != "encoding/json" {
				t.Errorf("metadata: expected import 'encoding/json', got '%s'", col.GoImportPath)
			}

		case "started_at", "completed_at":
			if !strings.Contains(col.DBType, "timestamp") {
				t.Errorf("%s: expected type 'timestamp', got '%s'", col.Name, col.DBType)
			}
			if !col.IsNullable {
				t.Errorf("%s: should be nullable", col.Name)
			}
			if col.GoType != "*time.Time" {
				t.Errorf("%s: expected Go type '*time.Time', got '%s'", col.Name, col.GoType)
			}
		}
	}
}

func TestExtractTableName(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		expectedTable string
		expectedSchema string
	}{
		{
			name:          "simple table",
			sql:           "CREATE TABLE users (id uuid)",
			expectedTable: "users",
			expectedSchema: "public",
		},
		{
			name:          "with schema",
			sql:           "CREATE TABLE public.tasks (id uuid)",
			expectedTable: "tasks",
			expectedSchema: "public",
		},
		{
			name:          "if not exists",
			sql:           "CREATE TABLE IF NOT EXISTS app_data (id uuid)",
			expectedTable: "app_data",
			expectedSchema: "public",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, schema, err := extractTableName(tt.sql)
			if err != nil {
				t.Fatalf("extractTableName failed: %v", err)
			}
			if table != tt.expectedTable {
				t.Errorf("Expected table '%s', got '%s'", tt.expectedTable, table)
			}
			if schema != tt.expectedSchema {
				t.Errorf("Expected schema '%s', got '%s'", tt.expectedSchema, schema)
			}
		})
	}
}

func TestMapPostgreSQLType(t *testing.T) {
	tests := []struct {
		dbType         string
		expectedGoType string
		expectedImport string
	}{
		{"uuid", "string", ""},
		{"varchar(100)", "string", ""},
		{"text", "string", ""},
		{"integer", "int", ""},
		{"bigint", "int64", ""},
		{"boolean", "bool", ""},
		{"timestamp", "time.Time", "time"},
		{"jsonb", "json.RawMessage", "encoding/json"},
		{"numeric(10,2)", "float64", ""},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			mapping, err := MapPostgreSQLType(tt.dbType)
			if err != nil {
				// Some types may return errors but still have fallback
			}

			if mapping.GoType != tt.expectedGoType {
				t.Errorf("Expected Go type '%s', got '%s'", tt.expectedGoType, mapping.GoType)
			}
			if mapping.Import != tt.expectedImport {
				t.Errorf("Expected import '%s', got '%s'", tt.expectedImport, mapping.Import)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"tasks", "task"},
		{"executions", "execution"},
		{"categories", "category"},
		{"boxes", "box"},
		{"users", "user"},
		{"data", "data"}, // already singular
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := singularize(tt.input)
			if result != tt.expected {
				t.Errorf("singularize(%s): expected '%s', got '%s'", tt.input, tt.expected, result)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"task_executions", "TaskExecutions"},
		{"user_id", "UserId"},
		{"created_at", "CreatedAt"},
		{"simple", "Simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToPascalCase(%s): expected '%s', got '%s'", tt.input, tt.expected, result)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"task_id", "taskId"},
		{"user_name", "userName"},
		{"created_at", "createdAt"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%s): expected '%s', got '%s'", tt.input, tt.expected, result)
			}
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"task_executions", "task-executions"},
		{"user_id", "user-id"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toKebabCase(tt.input)
			if result != tt.expected {
				t.Errorf("toKebabCase(%s): expected '%s', got '%s'", tt.input, tt.expected, result)
			}
		})
	}
}

func TestAnalyze(t *testing.T) {
	sql := `
CREATE TABLE public.task_executions (
    execution_id uuid PRIMARY KEY,
    task_id uuid NOT NULL,
    FOREIGN KEY (task_id) REFERENCES public.tasks (task_id)
);
`

	result, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	err = Analyze(result)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify naming context
	naming := result.Naming
	if naming == nil {
		t.Fatal("Naming context is nil")
	}

	if naming.EntityName != "TaskExecution" {
		t.Errorf("Expected EntityName 'TaskExecution', got '%s'", naming.EntityName)
	}

	if naming.EntityNamePlural != "TaskExecutions" {
		t.Errorf("Expected EntityNamePlural 'TaskExecutions', got '%s'", naming.EntityNamePlural)
	}

	if naming.PackageName != "taskexecutionsrepo" {
		t.Errorf("Expected PackageName 'taskexecutionsrepo', got '%s'", naming.PackageName)
	}

	if naming.HTTPBasePath != "/task-executions" {
		t.Errorf("Expected HTTPBasePath '/task-executions', got '%s'", naming.HTTPBasePath)
	}

	if naming.PKColumn != "execution_id" {
		t.Errorf("Expected PKColumn 'execution_id', got '%s'", naming.PKColumn)
	}

	if naming.PKGoName != "ExecutionId" {
		t.Errorf("Expected PKGoName 'ExecutionId', got '%s'", naming.PKGoName)
	}

	// Verify foreign key enrichment
	if len(result.Schema.ForeignKeys) != 1 {
		t.Fatalf("Expected 1 foreign key, got %d", len(result.Schema.ForeignKeys))
	}

	fk := result.Schema.ForeignKeys[0]
	if fk.EntityName != "Task" {
		t.Errorf("Expected FK EntityName 'Task', got '%s'", fk.EntityName)
	}

	if fk.RepoPackageName != "tasksrepo" {
		t.Errorf("Expected FK RepoPackageName 'tasksrepo', got '%s'", fk.RepoPackageName)
	}

	if fk.MethodSuffix != "ByTaskId" {
		t.Errorf("Expected FK MethodSuffix 'ByTaskId', got '%s'", fk.MethodSuffix)
	}
}

func TestValidateSchema(t *testing.T) {
	// Valid schema
	validSchema := &TableSchema{
		Name: "users",
		Columns: []Column{
			{Name: "id", DBType: "uuid"},
		},
		PrimaryKey: PrimaryKeyInfo{
			ColumnName: "id",
		},
	}

	errors := ValidateSchema(validSchema)
	if len(errors) > 0 {
		t.Errorf("Valid schema should have no errors, got: %v", errors)
	}

	// Invalid: no table name
	invalidSchema1 := &TableSchema{
		Columns: []Column{{Name: "id"}},
		PrimaryKey: PrimaryKeyInfo{ColumnName: "id"},
	}
	errors = ValidateSchema(invalidSchema1)
	if len(errors) == 0 {
		t.Error("Expected error for missing table name")
	}

	// Invalid: no columns
	invalidSchema2 := &TableSchema{
		Name: "users",
		PrimaryKey: PrimaryKeyInfo{ColumnName: "id"},
	}
	errors = ValidateSchema(invalidSchema2)
	if len(errors) == 0 {
		t.Error("Expected error for no columns")
	}

	// Invalid: no primary key
	invalidSchema3 := &TableSchema{
		Name: "users",
		Columns: []Column{{Name: "id"}},
	}
	errors = ValidateSchema(invalidSchema3)
	if len(errors) == 0 {
		t.Error("Expected error for no primary key")
	}
}
