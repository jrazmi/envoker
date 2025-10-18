package schema

import (
	"fmt"
	"strings"
	"unicode"
)

// Analyze enriches a ParseResult with derived naming conventions and relationship analysis
func Analyze(result *ParseResult) error {
	// Derive naming context
	naming := deriveNamingContext(result.Schema)
	result.Naming = naming

	// Analyze and enrich foreign keys with derived names
	for i := range result.Schema.ForeignKeys {
		enrichForeignKey(&result.Schema.ForeignKeys[i])
	}

	return nil
}

// deriveNamingContext generates all naming conventions from the table name
func deriveNamingContext(schema *TableSchema) *NamingContext {
	naming := &NamingContext{
		TableName: schema.Name,
	}

	// Generate singular form (basic pluralization)
	naming.TableNameSingular = singularize(schema.Name)

	// Entity names (PascalCase)
	naming.EntityName = ToPascalCase(naming.TableNameSingular)
	naming.EntityNamePlural = ToPascalCase(schema.Name)

	// Lowercase variants
	naming.EntityNameLower = ToCamelCase(naming.TableNameSingular)

	// Snake case
	naming.EntityNameSnake = toSnakeCase(naming.TableNameSingular)

	// Package names (lowercase, no underscores)
	naming.PackageName = strings.ReplaceAll(schema.Name, "_", "") + "repo"
	naming.StorePackage = strings.ReplaceAll(schema.Name, "_", "") + "pgxstore"
	naming.BridgePackage = strings.ReplaceAll(schema.Name, "_", "") + "repobridge"

	// Paths
	naming.RepoPath = fmt.Sprintf("core/repositories/%s", naming.PackageName)
	naming.StorePath = fmt.Sprintf("core/repositories/%s/stores/%s", naming.PackageName, naming.StorePackage)
	naming.BridgePath = fmt.Sprintf("bridge/repositories/%s", naming.BridgePackage)

	// HTTP paths (kebab-case)
	naming.HTTPBasePath = "/" + toKebabCase(schema.Name)
	naming.HTTPSingular = "/" + toKebabCase(naming.TableNameSingular)

	// Primary key naming
	if schema.PrimaryKey.ColumnName != "" {
		naming.PKColumn = schema.PrimaryKey.ColumnName
		naming.PKGoName = ToPascalCase(schema.PrimaryKey.ColumnName)
		naming.PKParamName = ToCamelCase(schema.PrimaryKey.ColumnName)
		naming.PKURLParam = schema.PrimaryKey.ColumnName // Keep as snake_case for URLs
	}

	return naming
}

// enrichForeignKey derives naming information for a foreign key relationship
func enrichForeignKey(fk *ForeignKey) {
	// Entity name from referenced table (singular, PascalCase)
	singularTable := singularize(fk.RefTable)
	fk.EntityName = ToPascalCase(singularTable)

	// Repository package name
	fk.RepoPackageName = strings.ReplaceAll(fk.RefTable, "_", "") + "repo"

	// Method suffix for generated methods (e.g., "ByTaskID")
	fk.MethodSuffix = "By" + ToPascalCase(fk.ColumnName)

	// HTTP path segment (e.g., "tasks/{task_id}/executions")
	fk.HTTPPathSegment = toKebabCase(fk.RefTable) + "/{" + fk.ColumnName + "}/" + toKebabCase(fk.RefTable)

	// Go parameter names
	fk.GoParamName = ToCamelCase(fk.ColumnName)
	fk.GoParamNameLower = strings.ToLower(string(fk.GoParamName[0])) + fk.GoParamName[1:]
}

// singularize converts a plural table name to singular (basic implementation)
func singularize(word string) string {
	word = strings.TrimSpace(word)

	// Handle common patterns
	if strings.HasSuffix(word, "ies") {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "sses") || strings.HasSuffix(word, "xes") || strings.HasSuffix(word, "zes") {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") {
		return word[:len(word)-1]
	}

	return word
}

// pluralize converts a singular word to plural (basic implementation)
func pluralize(word string) string {
	word = strings.TrimSpace(word)

	// Handle common patterns
	if strings.HasSuffix(word, "y") && len(word) > 1 {
		// Check if preceding letter is consonant
		if !isVowel(rune(word[len(word)-2])) {
			return word[:len(word)-1] + "ies"
		}
	}
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") || strings.HasSuffix(word, "z") ||
		strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh") {
		return word + "es"
	}

	return word + "s"
}

// ToPascalCase converts snake_case to PascalCase
func ToPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

// ToCamelCase converts snake_case to camelCase
func ToCamelCase(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 0 {
		return ""
	}

	// First part lowercase
	result := strings.ToLower(parts[0])

	// Rest PascalCase
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(string(parts[i][0])) + strings.ToLower(parts[i][1:])
		}
	}

	return result
}

// toSnakeCase converts PascalCase or camelCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// toKebabCase converts snake_case to kebab-case
func toKebabCase(s string) string {
	return strings.ReplaceAll(s, "_", "-")
}

// isVowel checks if a rune is a vowel
func isVowel(r rune) bool {
	r = unicode.ToLower(r)
	return r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u'
}

// DeriveFilterSpecs generates filter specifications for searchable/filterable columns
func DeriveFilterSpecs(schema *TableSchema) []FilterSpec {
	var specs []FilterSpec

	for _, col := range schema.Columns {
		// Skip primary key and audit fields
		if col.IsPrimaryKey || col.Name == "created_at" || col.Name == "updated_at" {
			continue
		}

		spec := FilterSpec{
			Name:          ToPascalCase(col.Name),
			GoType:        strings.TrimPrefix(col.GoType, "*"),
			DBColumn:      col.Name,
			ValidationTag: col.ValidationTags,
		}

		// Determine filter type based on column type
		if strings.Contains(col.DBType, "timestamp") || strings.Contains(col.DBType, "date") {
			spec.IsRange = true
			spec.IsExactMatch = false
			spec.IsSearch = false
		} else if strings.Contains(col.DBType, "int") || strings.Contains(col.DBType, "numeric") || strings.Contains(col.DBType, "float") {
			spec.IsRange = true
			spec.IsExactMatch = true
			spec.IsSearch = false
		} else if strings.Contains(col.DBType, "varchar") || strings.Contains(col.DBType, "text") {
			spec.IsRange = false
			spec.IsExactMatch = true
			spec.IsSearch = true
		} else {
			spec.IsExactMatch = true
			spec.IsSearch = false
		}

		specs = append(specs, spec)
	}

	return specs
}

// DeriveListByFKMethods generates method specifications for foreign key list operations
func DeriveListByFKMethods(schema *TableSchema) []FKListMethod {
	var methods []FKListMethod

	for _, fk := range schema.ForeignKeys {
		method := FKListMethod{
			MethodName:       "List" + fk.MethodSuffix,
			FKColumn:         fk.ColumnName,
			FKGoType:         strings.TrimPrefix(getFKGoType(schema, fk.ColumnName), "*"),
			FKParamName:      fk.GoParamName,
			RefEntityName:    fk.EntityName,
			RefRepoPackage:   fk.RepoPackageName,
			HTTPPath:         fmt.Sprintf("%s/%s", toKebabCase(fk.RefTable), toKebabCase(schema.Name)),
			HTTPHandlerName:  "List" + ToPascalCase(schema.Name) + fk.MethodSuffix,
		}
		methods = append(methods, method)
	}

	return methods
}

// FKListMethod represents a generated ListByFK method
type FKListMethod struct {
	MethodName      string // "ListByTaskID"
	FKColumn        string // "task_id"
	FKGoType        string // "string"
	FKParamName     string // "taskID"
	RefEntityName   string // "Task"
	RefRepoPackage  string // "tasksrepo"
	HTTPPath        string // "tasks/task-executions"
	HTTPHandlerName string // "ListTaskExecutionsByTaskID"
}

// getFKGoType gets the Go type for a foreign key column
func getFKGoType(schema *TableSchema, columnName string) string {
	for _, col := range schema.Columns {
		if col.Name == columnName {
			return col.GoType
		}
	}
	return "string" // default fallback
}

// ValidateSchema performs validation checks on the parsed schema
func ValidateSchema(schema *TableSchema) []error {
	var errors []error

	// Must have a table name
	if schema.Name == "" {
		errors = append(errors, fmt.Errorf("table name is required"))
	}

	// Must have at least one column
	if len(schema.Columns) == 0 {
		errors = append(errors, fmt.Errorf("table must have at least one column"))
	}

	// Must have a primary key
	if schema.PrimaryKey.ColumnName == "" {
		errors = append(errors, fmt.Errorf("table must have a primary key"))
	}

	// Validate foreign key references
	for _, fk := range schema.ForeignKeys {
		if fk.RefTable == "" {
			errors = append(errors, fmt.Errorf("foreign key %s has no reference table", fk.ColumnName))
		}
		if fk.RefColumn == "" {
			errors = append(errors, fmt.Errorf("foreign key %s has no reference column", fk.ColumnName))
		}

		// Check that FK column exists
		found := false
		for _, col := range schema.Columns {
			if col.Name == fk.ColumnName {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Errorf("foreign key references non-existent column: %s", fk.ColumnName))
		}
	}

	return errors
}

// ValidateTimestamps checks if the schema has required timestamp columns
func ValidateTimestamps(schema *TableSchema) []string {
	var warnings []string

	hasCreatedAt := false
	hasUpdatedAt := false

	for _, col := range schema.Columns {
		if col.Name == "created_at" {
			hasCreatedAt = true
		}
		if col.Name == "updated_at" {
			hasUpdatedAt = true
		}
	}

	if !hasCreatedAt {
		warnings = append(warnings, "table missing 'created_at' timestamp column (recommended)")
	}
	if !hasUpdatedAt {
		warnings = append(warnings, "table missing 'updated_at' timestamp column (recommended)")
	}

	return warnings
}

// HasStatusColumn checks if the schema has a status column for archive support
func HasStatusColumn(schema *TableSchema) bool {
	for _, col := range schema.Columns {
		if col.Name == "status" {
			return true
		}
	}
	return false
}

// HasDeletedAtColumn checks if the schema has a deleted_at column for soft delete support
func HasDeletedAtColumn(schema *TableSchema) bool {
	for _, col := range schema.Columns {
		if col.Name == "deleted_at" {
			return true
		}
	}
	return false
}
