package sqlparser

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Parse parses a CREATE TABLE SQL statement and returns a ParseResult
func Parse(sql string) (*ParseResult, error) {
	schema := &TableSchema{
		Comments: make(map[string]string),
	}

	// Extract table name and schema
	tableName, tableSchema, err := extractTableName(sql)
	if err != nil {
		return nil, fmt.Errorf("extract table name: %w", err)
	}
	schema.Name = tableName
	schema.Schema = tableSchema

	// Extract columns
	columns, err := extractColumns(sql)
	if err != nil {
		return nil, fmt.Errorf("extract columns: %w", err)
	}
	schema.Columns = columns

	// Extract primary key
	pkInfo, err := extractPrimaryKey(sql, columns)
	if err != nil {
		return nil, fmt.Errorf("extract primary key: %w", err)
	}
	schema.PrimaryKey = pkInfo

	// Mark primary key column
	for i := range schema.Columns {
		if schema.Columns[i].Name == pkInfo.ColumnName {
			schema.Columns[i].IsPrimaryKey = true
			schema.Columns[i].GoType = pkInfo.GoType
		}
	}

	// Extract foreign keys
	foreignKeys, err := extractForeignKeys(sql)
	if err != nil {
		return nil, fmt.Errorf("extract foreign keys: %w", err)
	}
	schema.ForeignKeys = foreignKeys

	// Mark foreign key columns and attach references
	for i := range schema.Columns {
		for j := range foreignKeys {
			if schema.Columns[i].Name == foreignKeys[j].ColumnName {
				schema.Columns[i].IsForeignKey = true
				schema.Columns[i].References = &foreignKeys[j]
			}
		}
	}

	// Extract indexes
	indexes, err := extractIndexes(sql)
	if err != nil {
		return nil, fmt.Errorf("extract indexes: %w", err)
	}
	schema.Indexes = indexes

	// Extract constraints
	constraints, err := extractConstraints(sql)
	if err != nil {
		return nil, fmt.Errorf("extract constraints: %w", err)
	}
	schema.Constraints = constraints

	// Extract comments
	comments := extractComments(sql)
	schema.Comments = comments
	for i := range schema.Columns {
		if comment, ok := comments[schema.Columns[i].Name]; ok {
			schema.Columns[i].Comment = comment
		}
	}

	return &ParseResult{
		Schema:    schema,
		Timestamp: time.Now(),
		SQLSource: sql,
	}, nil
}

// extractTableName extracts the table name and schema from CREATE TABLE statement
func extractTableName(sql string) (string, string, error) {
	// Match: CREATE TABLE [schema.]table_name
	re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:(\w+)\.)?(\w+)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 3 {
		return "", "", fmt.Errorf("could not find table name in SQL")
	}

	schema := matches[1]
	if schema == "" {
		schema = "public" // default schema
	}
	tableName := matches[2]

	return tableName, schema, nil
}

// extractColumns parses column definitions from CREATE TABLE statement
func extractColumns(sql string) ([]Column, error) {
	var columns []Column

	// Find the content between parentheses
	start := strings.Index(sql, "(")
	end := strings.LastIndex(sql, ")")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("could not find column definitions")
	}

	content := sql[start+1 : end]

	// Split by commas, but respect nested parentheses and constraints
	lines := splitColumnDefinitions(content)

	for _, line := range lines {
		// Handle multi-line chunks that may contain comments
		// Split by newlines and process only the non-comment lines
		subLines := strings.Split(line, "\n")
		var cleanLine string
		for _, subLine := range subLines {
			subLine = strings.TrimSpace(subLine)
			// Skip empty lines and comments
			if subLine == "" || strings.HasPrefix(subLine, "--") {
				continue
			}
			// Accumulate non-comment content
			if cleanLine != "" {
				cleanLine += " "
			}
			cleanLine += subLine
		}

		line = strings.TrimSpace(cleanLine)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip table-level constraints (PRIMARY KEY, FOREIGN KEY, CONSTRAINT, CHECK, UNIQUE)
		if isTableConstraint(line) {
			continue
		}

		// Parse column
		col, err := parseColumnDefinition(line)
		if err != nil {
			// Log warning but continue
			continue
		}

		columns = append(columns, col)
	}

	return columns, nil
}

// parseColumnDefinition parses a single column definition
func parseColumnDefinition(def string) (Column, error) {
	col := Column{}

	// Split by whitespace
	parts := strings.Fields(def)
	if len(parts) < 2 {
		return col, fmt.Errorf("invalid column definition: %s", def)
	}

	// Column name is first token
	col.Name = strings.Trim(parts[0], `"`)

	// Type is second token (may include parameters like varchar(100))
	col.DBType = parts[1]

	// Handle type parameters (e.g., varchar(100), numeric(10,2))
	// But stop at keywords like NOT, NULL, REFERENCES, DEFAULT
	if strings.HasSuffix(parts[1], "(") || (len(parts) > 2 && parts[2] == "(") {
		// Type has parameters, extract them
		typeStart := strings.Index(def, parts[1])
		remaining := def[typeStart:]

		// Find the matching closing paren for this type
		parenCount := 0
		foundStart := false
		for i, ch := range remaining {
			if ch == '(' {
				foundStart = true
				parenCount++
			} else if ch == ')' {
				parenCount--
				if foundStart && parenCount == 0 {
					col.DBType = remaining[:i+1]
					break
				}
			}
		}
	}

	// Map to Go type
	typeMapping, err := MapPostgreSQLType(col.DBType)
	if err != nil {
		// Use mapping but log the error (already has fallback)
	}
	col.GoType = typeMapping.GoType
	col.GoImportPath = typeMapping.Import

	// Parse constraints and modifiers
	defUpper := strings.ToUpper(def)

	// PRIMARY KEY
	col.IsPrimaryKey = strings.Contains(defUpper, "PRIMARY KEY")

	// NOT NULL
	col.IsNullable = !strings.Contains(defUpper, "NOT NULL") && !col.IsPrimaryKey

	// Apply pointer for nullable types (but not for slices)
	if col.IsNullable && !strings.HasPrefix(col.GoType, "*") && !strings.HasPrefix(col.GoType, "[]") {
		col.GoType = "*" + col.GoType
	}

	// DEFAULT
	col.HasDefault = strings.Contains(defUpper, "DEFAULT")
	if col.HasDefault {
		col.DefaultValue = extractDefaultValue(def)
	}

	// Max length for varchar
	if strings.HasPrefix(strings.ToLower(col.DBType), "varchar") {
		col.MaxLength = extractMaxLength(col.DBType)
	}

	// Precision and scale for numeric
	if strings.HasPrefix(strings.ToLower(col.DBType), "numeric") || strings.HasPrefix(strings.ToLower(col.DBType), "decimal") {
		col.Precision, col.Scale = extractPrecisionScale(col.DBType)
	}

	// Derive validation tags
	col.ValidationTags = DeriveValidationTag(col)

	return col, nil
}

// splitColumnDefinitions splits column definitions respecting nested parentheses
func splitColumnDefinitions(content string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, ch := range content {
		switch ch {
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	// Add last item
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// isTableConstraint checks if a line is a table-level constraint
func isTableConstraint(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	return strings.HasPrefix(upper, "PRIMARY KEY") ||
		strings.HasPrefix(upper, "FOREIGN KEY") ||
		strings.HasPrefix(upper, "CONSTRAINT") ||
		strings.HasPrefix(upper, "UNIQUE") ||
		strings.HasPrefix(upper, "CHECK") ||
		strings.HasPrefix(upper, "EXCLUDE")
}

// extractPrimaryKey extracts primary key information
func extractPrimaryKey(sql string, columns []Column) (PrimaryKeyInfo, error) {
	pkInfo := PrimaryKeyInfo{}

	// First, check if any column was already marked as primary key during parsing
	for _, col := range columns {
		if col.IsPrimaryKey {
			pkInfo.ColumnName = col.Name
			pkInfo.GoType = col.GoType
			pkInfo.HasDefault = col.HasDefault
			pkInfo.DefaultExpr = col.DefaultValue
			return pkInfo, nil
		}
	}

	// Try table-level PRIMARY KEY
	re := regexp.MustCompile(`(?i)PRIMARY KEY\s*\(([^)]+)\)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) >= 2 {
		// Take first column if composite
		pkColumns := strings.Split(matches[1], ",")
		columnName := strings.TrimSpace(pkColumns[0])
		// Remove quotes if present
		columnName = strings.Trim(columnName, `"`)
		pkInfo.ColumnName = columnName

		// Find the column to get its type
		for _, col := range columns {
			if col.Name == pkInfo.ColumnName {
				pkInfo.GoType = col.GoType
				pkInfo.HasDefault = col.HasDefault
				pkInfo.DefaultExpr = col.DefaultValue
				break
			}
		}

		return pkInfo, nil
	}

	return pkInfo, fmt.Errorf("no primary key found")
}

// extractForeignKeys extracts all foreign key relationships
func extractForeignKeys(sql string) ([]ForeignKey, error) {
	var foreignKeys []ForeignKey

	// Match explicit: FOREIGN KEY (column) REFERENCES schema.table (ref_column)
	explicitRe := regexp.MustCompile(`(?i)FOREIGN KEY\s*\(([^)]+)\)\s*REFERENCES\s+(?:(\w+)\.)?(\w+)\s*\(([^)]+)\)(?:\s+ON\s+DELETE\s+(CASCADE|SET NULL|RESTRICT|NO ACTION))?(?:\s+ON\s+UPDATE\s+(CASCADE|SET NULL|RESTRICT|NO ACTION))?`)

	matches := explicitRe.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		fk := ForeignKey{
			ColumnName: strings.TrimSpace(match[1]),
			RefSchema:  match[2],
			RefTable:   match[3],
			RefColumn:  strings.TrimSpace(match[4]),
			OnDelete:   match[5],
			OnUpdate:   match[6],
		}

		if fk.RefSchema == "" {
			fk.RefSchema = "public"
		}

		if fk.OnDelete == "" {
			fk.OnDelete = "NO ACTION"
		}

		if fk.OnUpdate == "" {
			fk.OnUpdate = "NO ACTION"
		}

		foreignKeys = append(foreignKeys, fk)
	}

	// Match inline: column_name type REFERENCES schema.table(ref_column)
	// Example: task_id uuid NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE
	inlineRe := regexp.MustCompile(`(?i)(\w+)\s+\w+(?:\([^)]*\))?\s+(?:NOT\s+NULL\s+)?REFERENCES\s+(?:(\w+)\.)?(\w+)\s*\(([^)]+)\)(?:\s+ON\s+DELETE\s+(CASCADE|SET NULL|RESTRICT|NO ACTION))?(?:\s+ON\s+UPDATE\s+(CASCADE|SET NULL|RESTRICT|NO ACTION))?`)

	inlineMatches := inlineRe.FindAllStringSubmatch(sql, -1)
	for _, match := range inlineMatches {
		fk := ForeignKey{
			ColumnName: strings.TrimSpace(match[1]),
			RefSchema:  match[2],
			RefTable:   match[3],
			RefColumn:  strings.TrimSpace(match[4]),
			OnDelete:   match[5],
			OnUpdate:   match[6],
		}

		if fk.RefSchema == "" {
			fk.RefSchema = "public"
		}

		if fk.OnDelete == "" {
			fk.OnDelete = "NO ACTION"
		}

		if fk.OnUpdate == "" {
			fk.OnUpdate = "NO ACTION"
		}

		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}

// extractIndexes extracts index definitions (from comments or separate statements)
func extractIndexes(sql string) ([]Index, error) {
	var indexes []Index

	// For now, we'll parse indexes from CREATE INDEX statements if they're in comments
	// This is a simplified version - real implementation would need more sophistication

	return indexes, nil
}

// extractConstraints extracts CHECK and other constraints
func extractConstraints(sql string) ([]Constraint, error) {
	var constraints []Constraint

	// Match: CONSTRAINT name CHECK (condition)
	re := regexp.MustCompile(`(?i)CONSTRAINT\s+(\w+)\s+(CHECK|UNIQUE|EXCLUDE)\s*\(([^)]+)\)`)
	matches := re.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		constraint := Constraint{
			Name:       match[1],
			Type:       strings.ToUpper(match[2]),
			Definition: match[3],
		}
		constraints = append(constraints, constraint)
	}

	return constraints, nil
}

// extractComments extracts column comments from COMMENT ON COLUMN statements
func extractComments(sql string) map[string]string {
	comments := make(map[string]string)

	// Match: COMMENT ON COLUMN table.column IS 'comment'
	re := regexp.MustCompile(`(?i)COMMENT ON COLUMN\s+\w+\.(\w+)\s+IS\s+'([^']+)'`)
	matches := re.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		columnName := match[1]
		comment := match[2]
		comments[columnName] = comment
	}

	return comments
}

// extractDefaultValue extracts the default value expression from a column definition
func extractDefaultValue(def string) string {
	// Match: DEFAULT expression
	re := regexp.MustCompile(`(?i)DEFAULT\s+([^,\s]+(?:\([^)]*\))?)`)
	matches := re.FindStringSubmatch(def)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
