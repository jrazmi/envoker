package pgxstores

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jrazmi/envoker/app/envoker/generators/sqlparser"
)

// SQLConfig holds configuration for SQL-based generation
type SQLConfig struct {
	ModulePath     string
	OutputDir      string
	ForceOverwrite bool
}

// GenerateFromSQL creates a pgx store from parsed SQL schema
func GenerateFromSQL(parseResult *sqlparser.ParseResult, config SQLConfig) (string, error) {
	schema := parseResult.Schema
	naming := parseResult.Naming

	// Build template configuration
	cfg := Config{
		Entity:      naming.EntityName,
		Table:       schema.Name,
		Schema:      schema.Schema,
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
	fields[cfg.Entity] = convertColumnsToFields(schema.Columns)
	fields[cfg.Create] = buildCreateFields(schema.Columns, schema.PrimaryKey)
	fields[cfg.Update] = buildUpdateFields(schema.Columns, schema.PrimaryKey)
	fields[cfg.Filter] = buildFilterFields(schema.Columns)

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
		"ModulePath":   cfg.ModulePath,
		"PackageName":  cfg.PackageName,
		"RepoPackage":  naming.PackageName,
		"Entity":       cfg.Entity,
		"StoreType":    cfg.StoreType,
		"Schema":       cfg.Schema,
		"Table":        cfg.Table,
		"PK":           cfg.PK,
		"PKGoName":     naming.PKGoName,
		"PKGoType":     strings.TrimPrefix(schema.PrimaryKey.GoType, "*"),
		"Create":       cfg.Create,
		"Update":       cfg.Update,
		"Filter":       cfg.Filter,
		"EntityFields": fields[cfg.Entity],
		"CreateFields": fields[cfg.Create],
		"UpdateFields": fields[cfg.Update],
		"FilterFields": fields[cfg.Filter],
		"PKInCreate":   pkInCreate,
		"ForeignKeys":  buildFKMethodData(schema.ForeignKeys, naming),
		"NeedsTime":    needsTimeImport(schema.Columns),
		"NeedsJSON":    needsJSONImport(schema.Columns),
	}

	// Determine output path
	storeDir := filepath.Join(config.OutputDir, naming.StorePath)
	storeFile := filepath.Join(storeDir, "store_gen.go")

	// Check for existing file
	if !config.ForceOverwrite && fileExists(storeFile) {
		return "", fmt.Errorf("file already exists: %s (use -force to overwrite)", storeFile)
	}

	// Create output directory
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	// Generate store_gen.go
	if err := generateStoreFile(storeFile, templateData); err != nil {
		return "", fmt.Errorf("generate store file: %w", err)
	}

	return storeFile, nil
}

// convertColumnsToFields converts sqlparser.Column to Field
func convertColumnsToFields(columns []sqlparser.Column) []Field {
	var fields []Field
	for _, col := range columns {
		field := Field{
			Name:       sqlparser.ToPascalCase(col.Name),
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
func buildCreateFields(columns []sqlparser.Column, pk sqlparser.PrimaryKeyInfo) []Field {
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
			Name:       sqlparser.ToPascalCase(col.Name),
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
func buildUpdateFields(columns []sqlparser.Column, pk sqlparser.PrimaryKeyInfo) []Field {
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
			Name:       sqlparser.ToPascalCase(col.Name),
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
func buildFilterFields(columns []sqlparser.Column) []Field {
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
			Name:       sqlparser.ToPascalCase(col.Name),
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
func buildFKMethodData(foreignKeys []sqlparser.ForeignKey, naming *sqlparser.NamingContext) []FKMethodData {
	var methods []FKMethodData
	for _, fk := range foreignKeys {
		method := FKMethodData{
			MethodName:  "List" + fk.MethodSuffix,
			FKColumn:    fk.ColumnName,
			FKGoType:    "string", // Assume string for UUIDs
			FKParamName: sqlparser.ToCamelCase(fk.ColumnName),
		}
		methods = append(methods, method)
	}
	return methods
}

// needsTimeImport checks if any column needs time import
func needsTimeImport(columns []sqlparser.Column) bool {
	for _, col := range columns {
		if strings.Contains(col.GoType, "time.Time") {
			return true
		}
	}
	return false
}

// needsJSONImport checks if any column needs json import
func needsJSONImport(columns []sqlparser.Column) bool {
	for _, col := range columns {
		if strings.Contains(col.GoType, "json.RawMessage") {
			return true
		}
	}
	return false
}

// generateStoreFile renders the store template to a file
func generateStoreFile(filepath string, data interface{}) error {
	tmpl, err := template.New("store").Parse(StoreTemplate)
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

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
