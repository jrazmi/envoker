package repositorygen

import (
	"github.com/jrazmi/envoker/app/generators/sqlparser"
)

// Config holds configuration for repository generation
type Config struct {
	ModulePath     string // e.g., "github.com/jrazmi/envoker"
	OutputDir      string // Base output directory
	ForceOverwrite bool   // If true, overwrite without prompting
}

// TemplateData holds all data needed for repository template rendering
type TemplateData struct {
	// Package and imports
	PackageName string   // e.g., "tasksrepo"
	Imports     []string // Required imports

	// Entity naming
	EntityName       string // e.g., "Task"
	EntityNamePlural string // e.g., "Tasks"
	EntityNameLower  string // e.g., "task"

	// Structs to generate
	CreateStructName string // e.g., "CreateTask"
	UpdateStructName string // e.g., "UpdateTask"
	FilterStructName string // e.g., "FilterTask"

	// Primary key
	PKColumn    string // e.g., "task_id"
	PKGoName    string // e.g., "TaskID"
	PKGoType    string // e.g., "string"
	PKParamName string // e.g., "taskID"
	PKInCreate  bool   // True if PK is in CreateStruct

	// Timestamp fields metadata
	HasCreatedAt       bool // True if table has created_at column
	CreatedAtIsPointer bool // True if CreatedAt is *time.Time

	// Columns and fields
	Columns      []sqlparser.Column
	EntityFields []FieldInfo // Fields for Entity struct
	CreateFields []FieldInfo // Fields for Create struct
	UpdateFields []FieldInfo // Fields for Update struct
	FilterFields []FieldInfo // Fields for Filter struct

	// Foreign keys
	ForeignKeys []FKMethodInfo

	// Storer interface info
	StorerInterfaceName string // e.g., "Storer"

	// Schema features
	HasStatusColumn bool // True if table has a status column for archive support
	HasDeletedAt    bool // True if table has a deleted_at column

	// File paths
	ModelFilePath      string // Where to write model_gen.go
	RepositoryFilePath string // Where to write repository_gen.go
}

// FieldInfo represents a struct field for code generation
type FieldInfo struct {
	Name         string // Go field name (PascalCase)
	GoType       string // Go type with pointer if nullable
	DBColumn     string // Database column name (snake_case)
	JSONTag      string // JSON tag value
	DBTag        string // DB tag value
	ValidateTag  string // Validation tag value
	Comment      string // Field comment
	IsPointer    bool   // Whether the field uses a pointer
	IsTime       bool   // Whether the field is time.Time
	IsJSON       bool   // Whether the field is json.RawMessage
	IsPrimaryKey bool   // Whether this is the primary key
	IsForeignKey bool   // Whether this is a foreign key
	HasDefault   bool   // Whether DB has a default value
}

// FKMethodInfo represents information for generating a ListByFK method
type FKMethodInfo struct {
	MethodName     string // e.g., "ListByApplicationID"
	FKColumn       string // e.g., "application_id"
	FKGoType       string // e.g., "string"
	FKParamName    string // e.g., "applicationID"
	FKGoName       string // e.g., "ApplicationID"
	RefEntityName  string // e.g., "Application"
	RefRepoPackage string // e.g., "applicationsrepo"
	Comment        string // Method documentation
}

// GenerateResult holds the results of a generation operation
type GenerateResult struct {
	ModelFile      string   // Path to generated model_gen.go
	RepositoryFile string   // Path to generated repository_gen.go
	Errors         []error  // Any errors encountered
	Warnings       []string // Any warnings
}
