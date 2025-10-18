package jsonschema

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jrazmi/envoker/app/generators/sqlparser"
	"github.com/jrazmi/envoker/schema/reflector"
)

// LoadTableFromJSON reads a reflected JSON schema file and extracts a single table's definition,
// converting it into the ParseResult format expected by the generators
func LoadTableFromJSON(jsonPath, tableName string) (*sqlparser.ParseResult, error) {
	// Read JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("read JSON file: %w", err)
	}

	// Parse JSON into ReflectedSchema
	var reflected reflector.ReflectedSchema
	if err := json.Unmarshal(data, &reflected); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	// Find the requested table
	tableInfo, ok := reflected.Tables[tableName]
	if !ok {
		return nil, fmt.Errorf("table %s not found in schema (available tables: %v)",
			tableName, getTableNames(reflected.Tables))
	}

	// Convert to ParseResult
	parseResult := convertToParseResult(&reflected, tableInfo)

	return parseResult, nil
}

// convertToParseResult converts a ReflectedSchema table to sqlparser.ParseResult
func convertToParseResult(reflected *reflector.ReflectedSchema, tableInfo *reflector.TableInfo) *sqlparser.ParseResult {
	schema := &sqlparser.TableSchema{
		Name:        tableInfo.TableName,
		Schema:      tableInfo.Schema,
		Columns:     convertColumns(tableInfo.Columns),
		PrimaryKey:  convertPrimaryKey(tableInfo.PrimaryKey),
		ForeignKeys: convertForeignKeys(tableInfo.ForeignKeys),
		Indexes:     convertIndexes(tableInfo.Indexes),
		Constraints: convertConstraints(tableInfo.Constraints),
		Comments:    extractComments(tableInfo.Columns),
	}

	return &sqlparser.ParseResult{
		Schema:    schema,
		Timestamp: time.Now(),
		SQLSource: fmt.Sprintf("-- Loaded from JSON: %s.%s", reflected.SchemaName, tableInfo.TableName),
	}
}

// convertColumns converts reflector.ColumnInfo to sqlparser.Column
func convertColumns(cols []reflector.ColumnInfo) []sqlparser.Column {
	result := make([]sqlparser.Column, len(cols))

	for i, col := range cols {
		result[i] = sqlparser.Column{
			Name:           col.Name,
			DBType:         col.DBType,
			GoType:         col.GoType,
			GoImportPath:   col.GoImport,
			IsNullable:     col.IsNullable,
			IsPrimaryKey:   col.IsPrimaryKey,
			IsForeignKey:   col.IsForeignKey,
			DefaultValue:   col.DefaultValue,
			HasDefault:     col.HasDefault,
			MaxLength:      col.MaxLength,
			Precision:      col.Precision,
			Scale:          col.Scale,
			ValidationTags: col.ValidationTags,
			Comment:        col.Comment,
		}
	}

	return result
}

// convertPrimaryKey converts reflector.PrimaryKeyInfo to sqlparser.PrimaryKeyInfo
func convertPrimaryKey(pk *reflector.PrimaryKeyInfo) sqlparser.PrimaryKeyInfo {
	if pk == nil {
		return sqlparser.PrimaryKeyInfo{}
	}

	return sqlparser.PrimaryKeyInfo{
		ColumnName:  pk.Column,
		GoType:      pk.GoType,
		HasDefault:  pk.HasDefault,
		DefaultExpr: pk.DefaultExpr,
	}
}

// convertForeignKeys converts reflector.ForeignKeyInfo to sqlparser.ForeignKey
func convertForeignKeys(fks []reflector.ForeignKeyInfo) []sqlparser.ForeignKey {
	result := make([]sqlparser.ForeignKey, len(fks))

	for i, fk := range fks {
		result[i] = sqlparser.ForeignKey{
			ColumnName: fk.ColumnName,
			RefTable:   fk.RefTable,
			RefSchema:  fk.RefSchema,
			RefColumn:  fk.RefColumn,
			OnDelete:   fk.OnDelete,
			OnUpdate:   fk.OnUpdate,
		}
	}

	return result
}

// convertIndexes converts reflector.IndexInfo to sqlparser.Index
func convertIndexes(indexes []reflector.IndexInfo) []sqlparser.Index {
	result := make([]sqlparser.Index, len(indexes))

	for i, idx := range indexes {
		result[i] = sqlparser.Index{
			Name:    idx.Name,
			Columns: idx.Columns,
			Unique:  idx.Unique,
			Method:  idx.Method,
		}
	}

	return result
}

// convertConstraints converts reflector.ConstraintInfo to sqlparser.Constraint
func convertConstraints(constraints []reflector.ConstraintInfo) []sqlparser.Constraint {
	result := make([]sqlparser.Constraint, len(constraints))

	for i, c := range constraints {
		result[i] = sqlparser.Constraint{
			Name:       c.Name,
			Type:       c.Type,
			Definition: c.Definition,
		}
	}

	return result
}

// extractComments builds a map of column name -> comment
func extractComments(cols []reflector.ColumnInfo) map[string]string {
	comments := make(map[string]string)

	for _, col := range cols {
		if col.Comment != "" {
			comments[col.Name] = col.Comment
		}
	}

	return comments
}

// getTableNames returns a list of table names from the map
func getTableNames(tables map[string]*reflector.TableInfo) []string {
	names := make([]string, 0, len(tables))
	for name := range tables {
		names = append(names, name)
	}
	return names
}

// ListTables returns all table names in a reflected JSON schema file
func ListTables(jsonPath string) ([]string, error) {
	// Read JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("read JSON file: %w", err)
	}

	// Parse JSON into ReflectedSchema
	var reflected reflector.ReflectedSchema
	if err := json.Unmarshal(data, &reflected); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	return getTableNames(reflected.Tables), nil
}
