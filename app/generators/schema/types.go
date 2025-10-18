package schema

import "time"

// TableSchema represents a database table schema definition
type TableSchema struct {
	Name        string // "task_executions"
	Schema      string // "public" (default)
	Columns     []Column
	PrimaryKey  PrimaryKeyInfo
	ForeignKeys []ForeignKey
	Indexes     []Index
	Constraints []Constraint
	Comments    map[string]string // column name -> comment
}

// Column represents a table column definition
type Column struct {
	Name           string // "execution_id"
	DBType         string // "uuid", "varchar(100)", "timestamp"
	GoType         string // "string", "*time.Time"
	GoImportPath   string // "time", "encoding/json"
	IsNullable     bool
	IsPrimaryKey   bool
	IsForeignKey   bool
	DefaultValue   string // "gen_random_uuid()", "now()"
	HasDefault     bool
	MaxLength      int // for varchar(n)
	Precision      int // for numeric(p,s)
	Scale          int
	References     *ForeignKey
	ValidationTags string // "required,uuid,min=1"
	Comment        string
}

// ForeignKey represents a foreign key relationship
type ForeignKey struct {
	ColumnName string // "task_id"
	RefTable   string // "tasks"
	RefSchema  string // "public"
	RefColumn  string // "task_id"
	OnDelete   string // "CASCADE", "SET NULL", "RESTRICT", "NO ACTION"
	OnUpdate   string // "CASCADE", "SET NULL", "RESTRICT", "NO ACTION"

	// Derived names for code generation
	EntityName       string // "Task"
	RepoPackageName  string // "tasksrepo"
	MethodSuffix     string // "ByTaskID"
	HTTPPathSegment  string // "tasks/{task_id}/executions"
	GoParamName      string // "taskID"
	GoParamNameLower string // "taskId"
}

// PrimaryKeyInfo represents primary key information
type PrimaryKeyInfo struct {
	ColumnName  string
	GoType      string
	HasDefault  bool   // true if DEFAULT clause exists
	DefaultExpr string // "gen_random_uuid()", "nextval(...)"
}

// Constraint represents a table constraint
type Constraint struct {
	Name       string
	Type       string // "CHECK", "UNIQUE", "EXCLUDE"
	Definition string
	Columns    []string
}

// Index represents a table index
type Index struct {
	Name    string
	Columns []string
	Unique  bool
	Method  string // "btree", "hash", "gin", "gist"
}

// TypeMapping represents Go type information for a PostgreSQL type
type TypeMapping struct {
	GoType       string
	Import       string // import path if needed
	Validation   string // validation tag pattern
	JSONType     string // JSON schema type
	IsPointer    bool   // whether nullable version should be pointer
	IsTime       bool   // special handling for time.Time
	IsJSONB      bool   // special handling for jsonb
	IsNumeric    bool   // for range filters
	BaseType     string // type without pointer (for conversions)
	DefaultValue string // default Go zero value
}

// NamingContext holds all derived names for code generation
type NamingContext struct {
	// From table name: "task_executions"
	TableName         string // "task_executions"
	TableNameSingular string // "task_execution"

	// Entity names
	EntityName       string // "TaskExecution"
	EntityNameLower  string // "taskExecution"
	EntityNameSnake  string // "task_execution"
	EntityNamePlural string // "TaskExecutions"

	// Package names
	PackageName   string // "taskexecutionsrepo"
	StorePackage  string // "taskexecutionspgxstore"
	BridgePackage string // "taskexecutionsrepobridge"

	// Paths
	RepoPath   string // "core/repositories/taskexecutionsrepo"
	StorePath  string // "core/repositories/taskexecutionsrepo/stores/taskexecutionspgxstore"
	BridgePath string // "bridge/repositories/taskexecutionsrepobridge"

	// HTTP
	HTTPBasePath string // "/task-executions"
	HTTPSingular string // "/task-execution"

	// Primary Key
	PKColumn    string // "execution_id"
	PKGoName    string // "ExecutionID"
	PKParamName string // "executionID"
	PKURLParam  string // "execution_id"
}

// FilterSpec represents a filterable field specification
type FilterSpec struct {
	Name          string
	GoType        string
	DBColumn      string
	IsRange       bool // supports Before/After or Min/Max
	IsExactMatch  bool
	IsSearch      bool
	ValidationTag string
}

// TableDefinition contains the complete table definition with naming context
type TableDefinition struct {
	Schema    *TableSchema
	Naming    *NamingContext
	Timestamp time.Time
	Source    string // source description (e.g., "JSON: public.tasks")
}

// ParseResult contains the complete parse result
type ParseResult struct {
	Schema    *TableSchema
	Naming    *NamingContext
	Timestamp time.Time
	SQLSource string // original SQL
}
