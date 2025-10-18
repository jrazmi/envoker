package schema

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// typeMap contains PostgreSQL to Go type mappings
var typeMap = map[string]TypeMapping{
	// UUIDs
	"uuid": {
		GoType:       "string",
		Import:       "",
		Validation:   "uuid",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "string",
		DefaultValue: `""`,
	},

	// Strings
	"text": {
		GoType:       "string",
		Import:       "",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "string",
		DefaultValue: `""`,
	},
	"varchar": {
		GoType:       "string",
		Import:       "",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "string",
		DefaultValue: `""`,
	},
	"character varying": {
		GoType:       "string",
		Import:       "",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "string",
		DefaultValue: `""`,
	},
	"char": {
		GoType:       "string",
		Import:       "",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "string",
		DefaultValue: `""`,
	},

	// Integers
	"integer": {
		GoType:       "int",
		Import:       "",
		Validation:   "",
		JSONType:     "integer",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "int",
		DefaultValue: "0",
	},
	"int": {
		GoType:       "int",
		Import:       "",
		Validation:   "",
		JSONType:     "integer",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "int",
		DefaultValue: "0",
	},
	"int4": {
		GoType:       "int",
		Import:       "",
		Validation:   "",
		JSONType:     "integer",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "int",
		DefaultValue: "0",
	},
	"smallint": {
		GoType:       "int16",
		Import:       "",
		Validation:   "",
		JSONType:     "integer",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "int16",
		DefaultValue: "0",
	},
	"int2": {
		GoType:       "int16",
		Import:       "",
		Validation:   "",
		JSONType:     "integer",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "int16",
		DefaultValue: "0",
	},
	"bigint": {
		GoType:       "int64",
		Import:       "",
		Validation:   "",
		JSONType:     "integer",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "int64",
		DefaultValue: "0",
	},
	"int8": {
		GoType:       "int64",
		Import:       "",
		Validation:   "",
		JSONType:     "integer",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "int64",
		DefaultValue: "0",
	},

	// Floating point
	"real": {
		GoType:       "float32",
		Import:       "",
		Validation:   "",
		JSONType:     "number",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "float32",
		DefaultValue: "0.0",
	},
	"float4": {
		GoType:       "float32",
		Import:       "",
		Validation:   "",
		JSONType:     "number",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "float32",
		DefaultValue: "0.0",
	},
	"double precision": {
		GoType:       "float64",
		Import:       "",
		Validation:   "",
		JSONType:     "number",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "float64",
		DefaultValue: "0.0",
	},
	"float8": {
		GoType:       "float64",
		Import:       "",
		Validation:   "",
		JSONType:     "number",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "float64",
		DefaultValue: "0.0",
	},
	"numeric": {
		GoType:       "float64",
		Import:       "",
		Validation:   "",
		JSONType:     "number",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "float64",
		DefaultValue: "0.0",
	},
	"decimal": {
		GoType:       "float64",
		Import:       "",
		Validation:   "",
		JSONType:     "number",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "float64",
		DefaultValue: "0.0",
	},

	// Boolean
	"boolean": {
		GoType:       "bool",
		Import:       "",
		Validation:   "",
		JSONType:     "boolean",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "bool",
		DefaultValue: "false",
	},
	"bool": {
		GoType:       "bool",
		Import:       "",
		Validation:   "",
		JSONType:     "boolean",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "bool",
		DefaultValue: "false",
	},

	// Date/Time
	"timestamp": {
		GoType:       "time.Time",
		Import:       "time",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       true,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "time.Time",
		DefaultValue: "time.Time{}",
	},
	"timestamp without time zone": {
		GoType:       "time.Time",
		Import:       "time",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       true,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "time.Time",
		DefaultValue: "time.Time{}",
	},
	"timestamp with time zone": {
		GoType:       "time.Time",
		Import:       "time",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       true,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "time.Time",
		DefaultValue: "time.Time{}",
	},
	"timestamptz": {
		GoType:       "time.Time",
		Import:       "time",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       true,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "time.Time",
		DefaultValue: "time.Time{}",
	},
	"date": {
		GoType:       "time.Time",
		Import:       "time",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       true,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "time.Time",
		DefaultValue: "time.Time{}",
	},
	"time": {
		GoType:       "time.Time",
		Import:       "time",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    true,
		IsTime:       true,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "time.Time",
		DefaultValue: "time.Time{}",
	},

	// JSON
	"json": {
		GoType:       "json.RawMessage",
		Import:       "encoding/json",
		Validation:   "",
		JSONType:     "object",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      true,
		IsNumeric:    false,
		BaseType:     "json.RawMessage",
		DefaultValue: "nil",
	},
	"jsonb": {
		GoType:       "json.RawMessage",
		Import:       "encoding/json",
		Validation:   "",
		JSONType:     "object",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      true,
		IsNumeric:    false,
		BaseType:     "json.RawMessage",
		DefaultValue: "nil",
	},

	// Arrays
	"text[]": {
		GoType:       "[]string",
		Import:       "",
		Validation:   "",
		JSONType:     "array",
		IsPointer:    false,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "[]string",
		DefaultValue: "nil",
	},
	"varchar[]": {
		GoType:       "[]string",
		Import:       "",
		Validation:   "",
		JSONType:     "array",
		IsPointer:    false,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "[]string",
		DefaultValue: "nil",
	},
	"integer[]": {
		GoType:       "[]int",
		Import:       "",
		Validation:   "",
		JSONType:     "array",
		IsPointer:    false,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    true,
		BaseType:     "[]int",
		DefaultValue: "nil",
	},

	// Binary
	"bytea": {
		GoType:       "[]byte",
		Import:       "",
		Validation:   "",
		JSONType:     "string",
		IsPointer:    false,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "[]byte",
		DefaultValue: "nil",
	},
}

// MapPostgreSQLType converts a PostgreSQL type to Go type information
func MapPostgreSQLType(dbType string) (TypeMapping, error) {
	// Normalize the type (lowercase, trim spaces)
	normalized := strings.ToLower(strings.TrimSpace(dbType))

	// Handle parameterized types: varchar(100), numeric(10,2), etc.
	baseType := extractBaseType(normalized)

	// Look up in map
	if mapping, ok := typeMap[baseType]; ok {
		// Clone the mapping to avoid modifying the original
		result := mapping

		// Add validation for varchar max length
		if strings.HasPrefix(baseType, "varchar") || strings.HasPrefix(baseType, "character varying") {
			if maxLen := extractMaxLength(normalized); maxLen > 0 {
				if result.Validation != "" {
					result.Validation += fmt.Sprintf(",max=%d", maxLen)
				} else {
					result.Validation = fmt.Sprintf("max=%d", maxLen)
				}
			}
		}

		return result, nil
	}

	// Default fallback for unknown types
	return TypeMapping{
		GoType:       "interface{}",
		Import:       "",
		Validation:   "",
		JSONType:     "object",
		IsPointer:    true,
		IsTime:       false,
		IsJSONB:      false,
		IsNumeric:    false,
		BaseType:     "interface{}",
		DefaultValue: "nil",
	}, fmt.Errorf("unknown PostgreSQL type: %s (using interface{} as fallback)", dbType)
}

// extractBaseType removes parameters from type definition
// Examples: "varchar(100)" -> "varchar", "numeric(10,2)" -> "numeric"
func extractBaseType(dbType string) string {
	if idx := strings.Index(dbType, "("); idx != -1 {
		return strings.TrimSpace(dbType[:idx])
	}
	return dbType
}

// extractMaxLength extracts max length from varchar(n) or character varying(n)
func extractMaxLength(dbType string) int {
	re := regexp.MustCompile(`\((\d+)\)`)
	matches := re.FindStringSubmatch(dbType)
	if len(matches) > 1 {
		if n, err := strconv.Atoi(matches[1]); err == nil {
			return n
		}
	}
	return 0
}

// extractPrecisionScale extracts precision and scale from numeric(p,s)
func extractPrecisionScale(dbType string) (precision int, scale int) {
	re := regexp.MustCompile(`\((\d+),\s*(\d+)\)`)
	matches := re.FindStringSubmatch(dbType)
	if len(matches) > 2 {
		if p, err := strconv.Atoi(matches[1]); err == nil {
			precision = p
		}
		if s, err := strconv.Atoi(matches[2]); err == nil {
			scale = s
		}
	}
	return
}

// GetImportsForColumns returns unique imports needed for a set of columns
func GetImportsForColumns(columns []Column) []string {
	importSet := make(map[string]bool)

	for _, col := range columns {
		if col.GoImportPath != "" {
			importSet[col.GoImportPath] = true
		}
	}

	imports := make([]string, 0, len(importSet))
	for imp := range importSet {
		imports = append(imports, imp)
	}

	return imports
}

// DeriveValidationTag creates validation tag for a column
func DeriveValidationTag(col Column) string {
	var tags []string

	// Required if NOT NULL and not a primary key (PKs have their own handling)
	if !col.IsNullable && !col.IsPrimaryKey {
		tags = append(tags, "required")
	}

	// UUID validation
	if col.DBType == "uuid" {
		tags = append(tags, "uuid")
	}

	// Max length for strings
	if col.MaxLength > 0 {
		tags = append(tags, fmt.Sprintf("max=%d", col.MaxLength))
	}

	// Min validation for numbers
	if strings.Contains(col.DBType, "int") || strings.Contains(col.DBType, "numeric") {
		if !col.IsNullable {
			tags = append(tags, "min=0")
		}
	}

	if len(tags) == 0 {
		return ""
	}

	return strings.Join(tags, ",")
}
