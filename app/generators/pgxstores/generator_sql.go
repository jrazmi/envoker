package pgxstores

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jrazmi/envoker/app/generators/schema"
)

// GenerateConfig holds configuration for generation
type GenerateConfig struct {
	ModulePath     string
	OutputDir      string
	ForceOverwrite bool
}

// GenerateFromSchema creates a pgx store from a table definition
func GenerateFromSchema(tableDef *schema.TableDefinition, config GenerateConfig) (string, error) {
	tableSchema := tableDef.Schema
	naming := tableDef.Naming

	// Build template configuration
	cfg := Config{
		Entity:      naming.EntityName,
		Table:       tableSchema.Name,
		Schema:      tableSchema.Schema,
		PK:          naming.PKColumn,
		PackageName: naming.StorePackage,
		ModulePath:  config.ModulePath,
		Create:      "Create" + naming.EntityName,
		Update:      "Update" + naming.EntityName,
		Filter:      "Filter" + naming.EntityName,
		StoreType:   "Store",
	}

	// Convert parsed columns to Field format
	fields := make(map[string][]Field)
	fields[cfg.Entity] = convertColumnsToFields(tableSchema.Columns)
	fields[cfg.Create] = buildCreateFields(tableSchema.Columns, tableSchema.PrimaryKey)
	fields[cfg.Update] = buildUpdateFields(tableSchema.Columns, tableSchema.PrimaryKey)
	fields[cfg.Filter] = buildFilterFields(tableSchema.Columns)

	// Check if PK is in Create struct
	pkInCreate := false
	for _, f := range fields[cfg.Create] {
		if f.DBColumn == cfg.PK {
			pkInCreate = true
			break
		}
	}

	// Build template data
	templateData := map[string]interface{}{
		"ModulePath":       cfg.ModulePath,
		"PackageName":      cfg.PackageName,
		"RepoPackage":      naming.PackageName,
		"Entity":           cfg.Entity,
		"StoreType":        cfg.StoreType,
		"Schema":           cfg.Schema,
		"Table":            cfg.Table,
		"PK":               cfg.PK,
		"PKGoName":         naming.PKGoName,
		"PKGoType":         strings.TrimPrefix(tableSchema.PrimaryKey.GoType, "*"),
		"PKParamName":      naming.PKParamName,
		"Create":           cfg.Create,
		"Update":           cfg.Update,
		"Filter":           cfg.Filter,
		"EntityFields":     fields[cfg.Entity],
		"CreateFields":     fields[cfg.Create],
		"UpdateFields":     fields[cfg.Update],
		"FilterFields":     buildFilterFieldsForFop(tableSchema.Columns),
		"PKInCreate":       pkInCreate,
		"ForeignKeys":      buildFKMethodData(tableSchema.ForeignKeys, tableSchema.Columns, naming),
		"NeedsTime":        needsTimeImport(tableSchema.Columns),
		"NeedsJSON":        needsJSONImport(tableSchema.Columns),
		"HasStatusColumn":  schema.HasStatusColumn(tableSchema),
		"HasDeletedAt":     schema.HasDeletedAtColumn(tableSchema),
		"OrderByFields":    buildOrderByFields(tableSchema.Columns),
		"SearchableFields": buildSearchableFields(tableSchema.Columns),
	}

	// Determine output paths
	storeDir := filepath.Join(config.OutputDir, naming.StorePath)
	generatedFile := filepath.Join(storeDir, "generated.go")
	storeFile := filepath.Join(storeDir, "store.go")

	// Check for existing file
	if !config.ForceOverwrite && fileExists(generatedFile) {
		return "", fmt.Errorf("file already exists: %s (use -force to overwrite)", generatedFile)
	}

	// Create output directory
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	// Generate generated.go (ALWAYS regenerate - contains ALL generated SQL)
	if err := generateTemplateFile(generatedFile, GeneratedStoreTemplate, templateData); err != nil {
		return "", fmt.Errorf("generate generated file: %w", err)
	}

	// Generate store.go ONLY if it doesn't exist (never overwrite - custom file)
	if !fileExists(storeFile) {
		if err := generateTemplateFile(storeFile, StoreCustomTemplate, templateData); err != nil {
			return "", fmt.Errorf("generate store file: %w", err)
		}
	}

	return generatedFile, nil
}

// convertColumnsToFields converts schema.Column to Field
func convertColumnsToFields(columns []schema.Column) []Field {
	var fields []Field
	for _, col := range columns {
		field := Field{
			Name:       schema.ToPascalCase(col.Name),
			DBColumn:   col.Name,
			GoType:     col.GoType,
			IsPointer:  strings.HasPrefix(col.GoType, "*"),
			IsNullable: col.IsNullable,
		}
		fields = append(fields, field)
	}
	return fields
}

// buildCreateFields creates fields for Create struct (excludes auto-generated)
func buildCreateFields(columns []schema.Column, pk schema.PrimaryKeyInfo) []Field {
	var fields []Field
	for _, col := range columns {
		// Skip auto-generated PK
		if col.IsPrimaryKey && col.HasDefault {
			continue
		}

		// Skip auto-generated timestamps
		if (col.Name == "created_at" || col.Name == "updated_at") && col.HasDefault {
			continue
		}

		field := Field{
			Name:       schema.ToPascalCase(col.Name),
			DBColumn:   col.Name,
			GoType:     col.GoType,
			IsPointer:  strings.HasPrefix(col.GoType, "*"),
			IsNullable: col.IsNullable,
		}
		fields = append(fields, field)
	}
	return fields
}

// buildUpdateFields creates fields for Update struct (all optional)
func buildUpdateFields(columns []schema.Column, pk schema.PrimaryKeyInfo) []Field {
	var fields []Field
	for _, col := range columns {
		// Skip PK and auto-timestamps
		if col.IsPrimaryKey || col.Name == "created_at" || col.Name == "updated_at" {
			continue
		}

		goType := col.GoType
		if !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") {
			goType = "*" + goType
		}

		field := Field{
			Name:       schema.ToPascalCase(col.Name),
			DBColumn:   col.Name,
			GoType:     goType,
			IsPointer:  true,
			IsNullable: true,
		}
		fields = append(fields, field)
	}
	return fields
}

// buildFilterFields creates fields for Filter struct
func buildFilterFields(columns []schema.Column) []Field {
	var fields []Field
	for _, col := range columns {
		// Skip audit timestamps
		if col.Name == "created_at" || col.Name == "updated_at" {
			continue
		}

		goType := col.GoType
		if !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") {
			goType = "*" + goType
		}

		field := Field{
			Name:       schema.ToPascalCase(col.Name),
			DBColumn:   col.Name,
			GoType:     goType,
			IsPointer:  true,
			IsNullable: true,
		}
		fields = append(fields, field)
	}
	return fields
}

// FKMethodData holds data for generating FK methods
type FKMethodData struct {
	MethodName  string // "ListByTaskId"
	FKColumn    string // "task_id"
	FKGoType    string // "string"
	FKParamName string // "taskId"
}

// buildFKMethodData creates FK method data from foreign keys
func buildFKMethodData(foreignKeys []schema.ForeignKey, columns []schema.Column, naming *schema.NamingContext) []FKMethodData {
	var methods []FKMethodData
	for _, fk := range foreignKeys {
		// Look up the actual Go type for this FK column
		fkGoType := "string" // default fallback
		for _, col := range columns {
			if col.Name == fk.ColumnName {
				// Remove pointer prefix if present since FK parameters should not be pointers
				fkGoType = strings.TrimPrefix(col.GoType, "*")
				break
			}
		}

		method := FKMethodData{
			MethodName:  "List" + fk.MethodSuffix,
			FKColumn:    fk.ColumnName,
			FKGoType:    fkGoType,
			FKParamName: schema.ToCamelCase(fk.ColumnName),
		}
		methods = append(methods, method)
	}
	return methods
}

// needsTimeImport checks if any column needs time import
func needsTimeImport(columns []schema.Column) bool {
	for _, col := range columns {
		if strings.Contains(col.GoType, "time.Time") {
			return true
		}
	}
	return false
}

// needsJSONImport checks if any column needs json import
func needsJSONImport(columns []schema.Column) bool {
	for _, col := range columns {
		if strings.Contains(col.GoType, "json.RawMessage") {
			return true
		}
	}
	return false
}

// FilterFieldForFop represents a field for filter generation
type FilterFieldForFop struct {
	GoName     string // PascalCase name
	DBColumn   string // snake_case column name
	ParamName  string // camelCase parameter name
	FilterType string // "exact", "timestamp_range", "numeric_range", "has_value"
	Comment    string // Documentation comment
}

// OrderByField represents a field that can be used for ordering
type OrderByField struct {
	GoName   string // PascalCase constant name (e.g., "ProcessingStatus")
	DBColumn string // Database column name (e.g., "processing_status")
}

// buildFilterFieldsForFop creates filter field metadata for fop_gen.go
func buildFilterFieldsForFop(columns []schema.Column) []FilterFieldForFop {
	var fields []FilterFieldForFop

	for _, col := range columns {
		if col.IsPrimaryKey {
			continue
		}

		// Skip JSONB columns - filtering entire JSON blobs doesn't make sense
		if strings.Contains(col.GoType, "json.RawMessage") || strings.Contains(col.DBType, "jsonb") {
			continue
		}

		// Skip timestamp columns - they're handled specially
		if col.Name == "created_at" || col.Name == "updated_at" {
			fields = append(fields, FilterFieldForFop{
				GoName:     schema.ToPascalCase(col.Name),
				DBColumn:   col.Name,
				ParamName:  schema.ToCamelCase(col.Name),
				FilterType: "timestamp_range",
				Comment:    "Filter by " + col.Name,
			})
			continue
		}

		// Use exact match filters for all other fields to match the repository filter definition
		// The repository filter uses simple pointer fields like *string, *int, etc.
		fields = append(fields, FilterFieldForFop{
			GoName:     schema.ToPascalCase(col.Name),
			DBColumn:   col.Name,
			ParamName:  schema.ToCamelCase(col.Name),
			FilterType: "exact",
			Comment:    "Filter by " + col.Name,
		})
	}

	return fields
}

// buildOrderByFields creates a list of fields that can be used for ordering
func buildOrderByFields(columns []schema.Column) []OrderByField {
	var fields []OrderByField

	for _, col := range columns {
		// Skip PK and timestamps - they're already added as constants
		if col.IsPrimaryKey || col.Name == "created_at" || col.Name == "updated_at" {
			continue
		}

		fields = append(fields, OrderByField{
			GoName:   schema.ToPascalCase(col.Name),
			DBColumn: col.Name,
		})
	}

	return fields
}

// buildSearchableFields returns a list of text columns suitable for full-text search
func buildSearchableFields(columns []schema.Column) []string {
	var fields []string

	for _, col := range columns {
		if strings.Contains(col.DBType, "varchar") || strings.Contains(col.DBType, "text") {
			fields = append(fields, col.Name)
		}
	}

	return fields
}

// generateTemplateFile renders a template and writes it to a file
func generateTemplateFile(filepath string, tmplStr string, data interface{}) error {
	tmpl, err := template.New("gen").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	f.Close() // Close before running goimports

	// Format and fix imports using goimports
	if err := formatGoFile(filepath); err != nil {
		// Don't fail generation if formatting fails, just log it
		fmt.Printf("Warning: failed to format %s: %v\n", filepath, err)
	}

	return nil
}

// formatGoFile runs goimports on a Go file to fix imports and format code
func formatGoFile(filepath string) error {
	cmd := exec.Command("goimports", "-w", filepath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("goimports failed: %w, output: %s", err, string(output))
	}
	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
