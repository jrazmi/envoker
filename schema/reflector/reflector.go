package reflector

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Reflector is the repository layer that orchestrates schema reflection
// It uses a Store (dependency injected) to query the database
type Reflector struct {
	store Store
}

// NewReflector creates a new Reflector with the given store
func NewReflector(store Store) *Reflector {
	return &Reflector{
		store: store,
	}
}

// Reflect queries the database via the store and returns a complete schema reflection
func (r *Reflector) Reflect(schemaName string) (*ReflectedSchema, error) {
	schema := &ReflectedSchema{
		Version:     "1.0",
		Source:      r.store.GetSourceType(),
		Database:    r.store.GetDatabaseName(),
		SchemaName:  schemaName,
		ReflectedAt: time.Now(),
		Tables:      make(map[string]*TableInfo),
	}

	// Get all tables in schema
	tables, err := r.store.GetTables(schemaName)
	if err != nil {
		return nil, fmt.Errorf("get tables: %w", err)
	}

	// For each table, extract complete metadata
	for _, tableName := range tables {
		tableInfo := &TableInfo{
			TableName:   tableName,
			Schema:      schemaName,
			Columns:     []ColumnInfo{},
			ForeignKeys: []ForeignKeyInfo{},
			Indexes:     []IndexInfo{},
			Constraints: []ConstraintInfo{},
		}

		// Get columns
		columns, err := r.store.GetColumns(schemaName, tableName)
		if err != nil {
			return nil, fmt.Errorf("get columns for %s: %w", tableName, err)
		}
		tableInfo.Columns = columns

		// Get primary key
		pk, err := r.store.GetPrimaryKey(schemaName, tableName, columns)
		if err != nil {
			// Primary key is optional, just log and continue
			fmt.Printf("Warning: could not get primary key for %s.%s: %v\n", schemaName, tableName, err)
		} else {
			tableInfo.PrimaryKey = pk
		}

		// Mark primary key column
		if pk != nil {
			for i := range tableInfo.Columns {
				if tableInfo.Columns[i].Name == pk.Column {
					tableInfo.Columns[i].IsPrimaryKey = true
				}
			}
		}

		// Get foreign keys
		fks, err := r.store.GetForeignKeys(schemaName, tableName)
		if err != nil {
			return nil, fmt.Errorf("get foreign keys for %s: %w", tableName, err)
		}
		tableInfo.ForeignKeys = fks

		// Mark foreign key columns
		for i := range tableInfo.Columns {
			for _, fk := range fks {
				if tableInfo.Columns[i].Name == fk.ColumnName {
					tableInfo.Columns[i].IsForeignKey = true
				}
			}
		}

		// Get indexes
		indexes, err := r.store.GetIndexes(schemaName, tableName)
		if err != nil {
			return nil, fmt.Errorf("get indexes for %s: %w", tableName, err)
		}
		tableInfo.Indexes = indexes

		// Get constraints
		constraints, err := r.store.GetConstraints(schemaName, tableName)
		if err != nil {
			return nil, fmt.Errorf("get constraints for %s: %w", tableName, err)
		}
		tableInfo.Constraints = constraints

		// Get table comment
		comment, _ := r.store.GetTableComment(schemaName, tableName)
		tableInfo.Comment = comment

		schema.Tables[tableName] = tableInfo
	}

	return schema, nil
}

// WriteJSON writes the schema to a JSON file
func WriteJSON(schema *ReflectedSchema, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(schema); err != nil {
		return err
	}

	return nil
}

// WriteSQL writes the schema to an SQL file (documentation format)
func WriteSQL(schema *ReflectedSchema, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "-- =============================================================================\n")
	fmt.Fprintf(file, "-- Schema Reflection: %s.%s\n", schema.Database, schema.SchemaName)
	fmt.Fprintf(file, "-- Reflected at: %s\n", schema.ReflectedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "-- Tables: %d\n", len(schema.Tables))
	fmt.Fprintf(file, "-- =============================================================================\n\n")

	// Write each table
	for _, tableName := range sortedTableNames(schema.Tables) {
		table := schema.Tables[tableName]
		if err := writeTableSQL(file, table); err != nil {
			return fmt.Errorf("write table %s: %w", tableName, err)
		}
		fmt.Fprintln(file) // Blank line between tables
	}

	return nil
}

// writeTableSQL writes a single table's SQL definition
func writeTableSQL(file *os.File, table *TableInfo) error {
	fmt.Fprintf(file, "-- -----------------------------------------------------------------------------\n")
	fmt.Fprintf(file, "-- Table: %s\n", table.TableName)
	if table.Comment != "" {
		fmt.Fprintf(file, "-- %s\n", table.Comment)
	}
	fmt.Fprintf(file, "-- -----------------------------------------------------------------------------\n")

	fmt.Fprintf(file, "CREATE TABLE %s.%s (\n", table.Schema, table.TableName)

	// Write columns
	columnLines := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		line := fmt.Sprintf("    %s %s", col.Name, col.DBType)

		if !col.IsNullable {
			line += " NOT NULL"
		}

		if col.HasDefault && col.DefaultValue != "" {
			line += fmt.Sprintf(" DEFAULT %s", col.DefaultValue)
		}

		columnLines = append(columnLines, line)
	}

	// Write primary key constraint
	if table.PrimaryKey != nil {
		columnLines = append(columnLines, fmt.Sprintf("    PRIMARY KEY (%s)", table.PrimaryKey.Column))
	}

	// Write foreign key constraints
	for _, fk := range table.ForeignKeys {
		fkLine := fmt.Sprintf("    FOREIGN KEY (%s) REFERENCES %s.%s(%s)",
			fk.ColumnName,
			fk.RefSchema,
			fk.RefTable,
			fk.RefColumn,
		)

		if fk.OnDelete != "" && fk.OnDelete != "NO_ACTION" {
			fkLine += fmt.Sprintf(" ON DELETE %s", strings.ReplaceAll(fk.OnDelete, "_", " "))
		}

		if fk.OnUpdate != "" && fk.OnUpdate != "NO_ACTION" {
			fkLine += fmt.Sprintf(" ON UPDATE %s", strings.ReplaceAll(fk.OnUpdate, "_", " "))
		}

		columnLines = append(columnLines, fkLine)
	}

	// Write CHECK constraints
	for _, constraint := range table.Constraints {
		if constraint.Type == "CHECK" {
			columnLines = append(columnLines, fmt.Sprintf("    CONSTRAINT %s %s", constraint.Name, constraint.Definition))
		}
	}

	// Join all lines
	fmt.Fprint(file, strings.Join(columnLines, ",\n"))
	fmt.Fprintln(file)
	fmt.Fprintln(file, ");")

	// Write indexes
	for _, idx := range table.Indexes {
		uniqueStr := ""
		if idx.Unique {
			uniqueStr = "UNIQUE "
		}

		fmt.Fprintf(file, "CREATE %sINDEX %s ON %s.%s USING %s (%s);\n",
			uniqueStr,
			idx.Name,
			table.Schema,
			table.TableName,
			idx.Method,
			strings.Join(idx.Columns, ", "),
		)
	}

	// Write table comment
	if table.Comment != "" {
		fmt.Fprintf(file, "\nCOMMENT ON TABLE %s.%s IS '%s';\n",
			table.Schema,
			table.TableName,
			escapeSQLString(table.Comment),
		)
	}

	// Write column comments
	for _, col := range table.Columns {
		if col.Comment != "" {
			fmt.Fprintf(file, "COMMENT ON COLUMN %s.%s.%s IS '%s';\n",
				table.Schema,
				table.TableName,
				col.Name,
				escapeSQLString(col.Comment),
			)
		}
	}

	return nil
}

// sortedTableNames returns table names sorted alphabetically
func sortedTableNames(tables map[string]*TableInfo) []string {
	names := make([]string, 0, len(tables))
	for name := range tables {
		names = append(names, name)
	}

	// Simple bubble sort (fine for small lists)
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	return names
}

// escapeSQLString escapes single quotes in SQL strings
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
