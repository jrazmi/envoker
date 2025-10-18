package bridgegen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jrazmi/envoker/app/generators/schema"
)

// Generate creates bridge files from a parsed SQL schema
func Generate(parseResult *schema.TableDefinition, config Config) (*GenerateResult, error) {
	result := &GenerateResult{}

	// Prepare template data
	templateData, err := prepareTemplateData(parseResult, config)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result, err
	}

	// Determine output paths
	bridgeDir := filepath.Join(config.OutputDir, parseResult.Naming.BridgePath)
	bridgeInitFile := filepath.Join(bridgeDir, "bridge.go")
	httpRoutesFile := filepath.Join(bridgeDir, "http.go") // User-editable route registration (never overwrite)
	httpFile := filepath.Join(bridgeDir, "http_gen.go")   // Generated HTTP handlers
	modelFile := filepath.Join(bridgeDir, "model_gen.go")
	marshalFile := filepath.Join(bridgeDir, "marshal_gen.go")
	fopFile := filepath.Join(bridgeDir, "fop_gen.go")

	// Check for existing files
	if !config.ForceOverwrite {
		for _, file := range []string{httpFile, modelFile, marshalFile, fopFile} {
			if fileExists(file) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("File exists: %s (use -force to overwrite)", file))
				return result, fmt.Errorf("file already exists: %s", file)
			}
		}
	}

	// Create output directory
	if err := os.MkdirAll(bridgeDir, 0755); err != nil {
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("create directory: %w", err)
	}

	// Generate bridge.go ONLY if it doesn't exist (never overwrite)
	if !fileExists(bridgeInitFile) {
		if err := generateFile(bridgeInitFile, BridgeInitTemplate, templateData); err != nil {
			result.Errors = append(result.Errors, err)
			return result, fmt.Errorf("generate bridge init file: %w", err)
		}
		result.BridgeFile = bridgeInitFile
	} else {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Skipped: %s (already exists, will not overwrite)", bridgeInitFile))
		result.BridgeFile = bridgeInitFile
	}

	// Generate http.go ONLY if it doesn't exist (never overwrite, even with -force)
	if !fileExists(httpRoutesFile) {
		if err := generateFile(httpRoutesFile, HTTPRoutesTemplate, templateData); err != nil {
			result.Errors = append(result.Errors, err)
			return result, fmt.Errorf("generate http routes file: %w", err)
		}
		result.HTTPRoutesFile = httpRoutesFile
	} else {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Skipped: %s (already exists, will not overwrite)", httpRoutesFile))
		result.HTTPRoutesFile = httpRoutesFile
	}

	// Generate http_gen.go (can be overwritten with -force)
	if err := generateFile(httpFile, HTTPTemplate, templateData); err != nil {
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("generate http file: %w", err)
	}
	result.HTTPFile = httpFile

	if err := generateFile(modelFile, ModelTemplate, templateData); err != nil {
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("generate model file: %w", err)
	}
	result.ModelFile = modelFile

	if err := generateFile(marshalFile, MarshalTemplate, templateData); err != nil {
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("generate marshal file: %w", err)
	}
	result.MarshalFile = marshalFile

	if err := generateFile(fopFile, FOPTemplate, templateData); err != nil {
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("generate fop file: %w", err)
	}
	result.FOPFile = fopFile

	return result, nil
}

// prepareTemplateData converts parsed schema to template data
func prepareTemplateData(parseResult *schema.TableDefinition, config Config) (*TemplateData, error) {
	schema := parseResult.Schema
	naming := parseResult.Naming

	data := &TemplateData{
		PackageName:      naming.BridgePackage,
		RepoPackage:      naming.PackageName,
		Entity:           naming.EntityName,
		EntityName:       naming.EntityName,
		EntityNamePlural: naming.EntityNamePlural,
		EntityNameLower:  naming.EntityNameLower,
		EntityNameCamel:  toCamelCase(naming.EntityNameLower),
		HTTPBasePath:     naming.HTTPBasePath,
		HTTPSingular:     naming.HTTPSingular,
		PKColumn:         naming.PKColumn,
		PKGoName:         naming.PKGoName,
		PKJSONName:       toCamelCase(naming.PKColumn),
		PKGoType:         strings.TrimPrefix(schema.PrimaryKey.GoType, "*"),
		PKParamName:      naming.PKParamName,
		PKURLParam:       naming.PKURLParam,
		ModulePath:       config.ModulePath,
		BridgePackage:    naming.BridgePackage,
		HasStatusColumn:  schema.HasStatusColumn(schema),
	}

	// Build field lists
	data.EntityFields = buildBridgeFields(schema.Columns, false)
	data.CreateFields = buildCreateBridgeFields(schema.Columns, schema.PrimaryKey)
	data.UpdateFields = buildUpdateBridgeFields(schema.Columns, schema.PrimaryKey)
	data.FilterFields = buildFilterBridgeFields(schema.Columns)

	// Build FK methods
	data.ForeignKeys = buildFKBridgeMethods(schema.ForeignKeys, naming)

	// Check if we need time import
	data.NeedsTimeImport = needsTimeImport(schema.Columns)

	return data, nil
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

// buildBridgeFields creates bridge fields from columns
func buildBridgeFields(columns []schema.Column, omitEmpty bool) []BridgeField {
	var fields []BridgeField
	for _, col := range columns {
		field := BridgeField{
			RepoName:   schema.ToPascalCase(col.Name),
			BridgeName: schema.ToPascalCase(col.Name),
			JSONName:   toCamelCase(col.Name),
			GoType:     col.GoType,
			DBColumn:   col.Name,
			IsPointer:  strings.HasPrefix(col.GoType, "*"),
			OmitEmpty:  omitEmpty || col.IsNullable,
			IsTime:     strings.Contains(col.GoType, "time.Time"),
			IsJSON:     strings.Contains(col.GoType, "json.RawMessage"),
		}
		fields = append(fields, field)
	}
	return fields
}

// buildCreateBridgeFields creates bridge fields for Create input
func buildCreateBridgeFields(columns []schema.Column, pk schema.PrimaryKeyInfo) []BridgeField {
	var fields []BridgeField
	for _, col := range columns {
		// Skip auto-generated PK
		if col.IsPrimaryKey && col.HasDefault {
			continue
		}

		// Skip auto-generated timestamps
		if (col.Name == "created_at" || col.Name == "updated_at") && col.HasDefault {
			continue
		}

		field := BridgeField{
			RepoName:   schema.ToPascalCase(col.Name),
			BridgeName: schema.ToPascalCase(col.Name),
			JSONName:   toCamelCase(col.Name),
			GoType:     col.GoType,
			DBColumn:   col.Name,
			IsPointer:  strings.HasPrefix(col.GoType, "*"),
			OmitEmpty:  col.IsNullable || col.HasDefault,
			IsTime:     strings.Contains(col.GoType, "time.Time"),
			IsJSON:     strings.Contains(col.GoType, "json.RawMessage"),
		}
		fields = append(fields, field)
	}
	return fields
}

// buildUpdateBridgeFields creates bridge fields for Update input
func buildUpdateBridgeFields(columns []schema.Column, pk schema.PrimaryKeyInfo) []BridgeField {
	var fields []BridgeField
	for _, col := range columns {
		// Skip PK and auto-timestamps
		if col.IsPrimaryKey || col.Name == "created_at" || col.Name == "updated_at" {
			continue
		}

		goType := col.GoType
		if !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") {
			goType = "*" + goType
		}

		field := BridgeField{
			RepoName:   schema.ToPascalCase(col.Name),
			BridgeName: schema.ToPascalCase(col.Name),
			JSONName:   toCamelCase(col.Name),
			GoType:     goType,
			DBColumn:   col.Name,
			IsPointer:  true,
			OmitEmpty:  true,
			IsTime:     strings.Contains(col.GoType, "time.Time"),
			IsJSON:     strings.Contains(col.GoType, "json.RawMessage"),
		}
		fields = append(fields, field)
	}
	return fields
}

// buildFilterBridgeFields creates bridge fields for Filter
func buildFilterBridgeFields(columns []schema.Column) []BridgeField {
	var fields []BridgeField
	for _, col := range columns {
		// Skip audit timestamps
		if col.Name == "created_at" || col.Name == "updated_at" {
			continue
		}

		// Skip primary keys (they're not filterable in the repository filter)
		if col.IsPrimaryKey {
			continue
		}

		// Skip JSONB fields (not filterable - no schema)
		if strings.Contains(col.GoType, "json.RawMessage") || strings.Contains(col.DBType, "jsonb") {
			continue
		}

		goType := col.GoType
		if !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") {
			goType = "*" + goType
		}

		field := BridgeField{
			RepoName:   schema.ToPascalCase(col.Name),
			BridgeName: schema.ToPascalCase(col.Name),
			JSONName:   toCamelCase(col.Name),
			GoType:     goType,
			DBColumn:   col.Name,
			IsPointer:  true,
			OmitEmpty:  true,
			IsTime:     strings.Contains(col.GoType, "time.Time"),
			IsJSON:     strings.Contains(col.GoType, "json.RawMessage"),
		}
		fields = append(fields, field)
	}
	return fields
}

// buildFKBridgeMethods creates FK method data from foreign keys
func buildFKBridgeMethods(foreignKeys []schema.ForeignKey, naming *schema.NamingContext) []FKBridgeMethod {
	var methods []FKBridgeMethod
	for _, fk := range foreignKeys {
		method := FKBridgeMethod{
			MethodName:    "httpListBy" + schema.ToPascalCase(fk.ColumnName),
			RoutePath:     fmt.Sprintf("/%s/{%s}%s", toKebabCase(fk.RefTable), fk.ColumnName, naming.HTTPBasePath),
			FKColumn:      fk.ColumnName,
			FKGoName:      schema.ToPascalCase(fk.ColumnName),
			FKParamName:   schema.ToCamelCase(fk.ColumnName),
			FKURLParam:    fk.ColumnName,
			FKGoType:      "string", // Assume string for UUIDs
			RefEntityName: fk.EntityName,
		}
		methods = append(methods, method)
	}
	return methods
}

// toCamelCase converts snake_case to camelCase
func toCamelCase(s string) string {
	return schema.ToCamelCase(s)
}

// toKebabCase converts snake_case to kebab-case
func toKebabCase(s string) string {
	return strings.ReplaceAll(s, "_", "-")
}

// generateFile renders a template and writes it to a file
func generateFile(filepath string, tmplStr string, data interface{}) error {
	funcMap := template.FuncMap{
		"Contains": strings.Contains,
	}

	tmpl, err := template.New("gen").Funcs(funcMap).Parse(tmplStr)
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
