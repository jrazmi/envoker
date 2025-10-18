package reflector

import "time"

// ReflectedSchema represents the complete schema reflection for a single database schema
type ReflectedSchema struct {
	Version      string                  `json:"version"`       // Schema format version (e.g., "1.0")
	Source       string                  `json:"source"`        // Database type (e.g., "postgres")
	Database     string                  `json:"database"`      // Database name
	SchemaName   string                  `json:"schema_name"`   // Schema name (e.g., "public")
	ReflectedAt  time.Time               `json:"reflected_at"`  // Timestamp of reflection
	Tables       map[string]*TableInfo   `json:"tables"`        // Map of table_name -> TableInfo
}

// TableInfo represents a single table's metadata
type TableInfo struct {
	TableName   string          `json:"table_name"`
	Schema      string          `json:"schema"`
	PrimaryKey  *PrimaryKeyInfo `json:"primary_key"`
	Columns     []ColumnInfo    `json:"columns"`
	ForeignKeys []ForeignKeyInfo `json:"foreign_keys"`
	Indexes     []IndexInfo     `json:"indexes"`
	Constraints []ConstraintInfo `json:"constraints"`
	Comment     string          `json:"comment,omitempty"`
}

// ColumnInfo represents a single column's metadata
type ColumnInfo struct {
	Name           string `json:"name"`
	DBType         string `json:"db_type"`          // PostgreSQL type (e.g., "uuid", "varchar(255)")
	GoType         string `json:"go_type"`          // Go type (e.g., "string", "*time.Time")
	GoImport       string `json:"go_import"`        // Import path if needed (e.g., "time")
	IsNullable     bool   `json:"is_nullable"`
	IsPrimaryKey   bool   `json:"is_primary_key"`
	IsForeignKey   bool   `json:"is_foreign_key"`
	DefaultValue   string `json:"default_value,omitempty"`
	HasDefault     bool   `json:"has_default"`
	MaxLength      int    `json:"max_length,omitempty"`      // For varchar(n)
	Precision      int    `json:"precision,omitempty"`       // For numeric(p,s)
	Scale          int    `json:"scale,omitempty"`           // For numeric(p,s)
	ValidationTags string `json:"validation_tags,omitempty"` // Validation tags (e.g., "required,uuid")
	Comment        string `json:"comment,omitempty"`
}

// PrimaryKeyInfo represents primary key metadata
type PrimaryKeyInfo struct {
	Column      string `json:"column"`
	DBType      string `json:"db_type"`
	GoType      string `json:"go_type"`
	HasDefault  bool   `json:"has_default"`
	DefaultExpr string `json:"default_expr,omitempty"`
}

// ForeignKeyInfo represents a foreign key relationship
type ForeignKeyInfo struct {
	ColumnName string `json:"column_name"`
	RefTable   string `json:"ref_table"`
	RefSchema  string `json:"ref_schema"`
	RefColumn  string `json:"ref_column"`
	OnDelete   string `json:"on_delete"` // CASCADE, SET NULL, RESTRICT, NO ACTION
	OnUpdate   string `json:"on_update"` // CASCADE, SET NULL, RESTRICT, NO ACTION
}

// IndexInfo represents an index
type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Method  string   `json:"method"` // btree, hash, gin, gist, etc.
}

// ConstraintInfo represents a table constraint (CHECK, UNIQUE, EXCLUDE)
type ConstraintInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`       // CHECK, UNIQUE, EXCLUDE
	Definition string `json:"definition"` // The constraint expression
}

// Store is the interface that database stores must implement for reflection
// This is the "store" layer - it knows how to query the database
type Store interface {
	// GetTables returns all table names in the schema
	GetTables(schemaName string) ([]string, error)

	// GetColumns returns column metadata for a table
	GetColumns(schemaName, tableName string) ([]ColumnInfo, error)

	// GetPrimaryKey returns primary key information
	GetPrimaryKey(schemaName, tableName string, columns []ColumnInfo) (*PrimaryKeyInfo, error)

	// GetForeignKeys returns foreign key relationships
	GetForeignKeys(schemaName, tableName string) ([]ForeignKeyInfo, error)

	// GetIndexes returns index information
	GetIndexes(schemaName, tableName string) ([]IndexInfo, error)

	// GetConstraints returns constraint information
	GetConstraints(schemaName, tableName string) ([]ConstraintInfo, error)

	// GetTableComment returns table comment
	GetTableComment(schemaName, tableName string) (string, error)

	// GetDatabaseName returns the database name
	GetDatabaseName() string

	// GetSourceType returns the database type (e.g., "postgres", "firestore")
	GetSourceType() string
}

// Config holds configuration for schema reflection output
type Config struct {
	SchemaName string // Schema to reflect (default: "public")
	OutputDir  string // Where to write output files
}
