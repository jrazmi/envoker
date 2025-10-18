package repositorygen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jrazmi/envoker/app/generators/sqlparser"
)

// Generate creates repository files from a parsed SQL schema
func Generate(parseResult *sqlparser.ParseResult, config Config) (*GenerateResult, error) {
	result := &GenerateResult{}

	// Prepare template data
	templateData, err := prepareTemplateData(parseResult, config)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result, err
	}

	// Determine output paths
	repoDir := filepath.Join(config.OutputDir, parseResult.Naming.RepoPath)
	modelFile := filepath.Join(repoDir, "model_gen.go")
	repoFile := filepath.Join(repoDir, "repo.go")
	repoGenFile := filepath.Join(repoDir, "repo_gen.go")
	fopGenFile := filepath.Join(repoDir, "fop_gen.go")

	templateData.ModelFilePath = modelFile
	templateData.RepositoryFilePath = repoFile

	// Check for existing files and prompt if needed
	if !config.ForceOverwrite {
		for _, file := range []string{modelFile, repoGenFile, fopGenFile} {
			if fileExists(file) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("File exists: %s (use -force to overwrite)", file))
				return result, fmt.Errorf("file already exists: %s", file)
			}
		}
	}

	// Create output directory
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("create directory: %w", err)
	}

	// Generate model_gen.go
	if err := generateFile(modelFile, ModelTemplate, templateData); err != nil {
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("generate model file: %w", err)
	}
	result.ModelFile = modelFile

	// Generate repo.go ONLY if it doesn't exist (never overwrite)
	if !fileExists(repoFile) {
		if err := generateFile(repoFile, RepoTemplate, templateData); err != nil {
			result.Errors = append(result.Errors, err)
			return result, fmt.Errorf("generate repository file: %w", err)
		}
		result.RepositoryFile = repoFile
	} else {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Skipped: %s (already exists, will not overwrite)", repoFile))
		result.RepositoryFile = repoFile
	}

	// Generate repo_gen.go (always regenerate)
	if err := generateFile(repoGenFile, RepoGenTemplate, templateData); err != nil {
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("generate repo_gen file: %w", err)
	}

	// Generate fop_gen.go (always regenerate)
	if err := generateFile(fopGenFile, RepoFopGenTemplate, templateData); err != nil {
		result.Errors = append(result.Errors, err)
		return result, fmt.Errorf("generate fop_gen file: %w", err)
	}

	return result, nil
}

// prepareTemplateData converts parsed schema to template data
func prepareTemplateData(parseResult *sqlparser.ParseResult, config Config) (*TemplateData, error) {
	schema := parseResult.Schema
	naming := parseResult.Naming

	data := &TemplateData{
		PackageName:         naming.PackageName,
		EntityName:          naming.EntityName,
		EntityNamePlural:    naming.EntityNamePlural,
		EntityNameLower:     naming.EntityNameLower,
		CreateStructName:    "Create" + naming.EntityName,
		UpdateStructName:    "Update" + naming.EntityName,
		FilterStructName:    "Filter" + naming.EntityName,
		PKColumn:            naming.PKColumn,
		PKGoName:            naming.PKGoName,
		PKGoType:            strings.TrimPrefix(schema.PrimaryKey.GoType, "*"),
		PKParamName:         naming.PKParamName,
		StorerInterfaceName: "Storer",
		Columns:             schema.Columns,
		HasStatusColumn:     sqlparser.HasStatusColumn(schema),
		HasDeletedAt:        sqlparser.HasDeletedAtColumn(schema),
	}

	// Build field lists
	data.EntityFields = buildEntityFields(schema.Columns)
	data.CreateFields = buildCreateFields(schema.Columns, schema.PrimaryKey)
	data.UpdateFields = buildUpdateFields(schema.Columns, schema.PrimaryKey)
	data.FilterFields = buildFilterFields(schema.Columns, schema.PrimaryKey)

	// Check if PK is in Create struct
	data.PKInCreate = false
	for _, f := range data.CreateFields {
		if f.DBColumn == data.PKColumn {
			data.PKInCreate = true
			break
		}
	}

	// Check if CreatedAt exists and if it's a pointer
	data.HasCreatedAt = false
	data.CreatedAtIsPointer = false
	for _, col := range schema.Columns {
		if col.Name == "created_at" {
			data.HasCreatedAt = true
			if strings.HasPrefix(col.GoType, "*") {
				data.CreatedAtIsPointer = true
			}
			break
		}
	}

	// Build FK method info
	data.ForeignKeys = buildFKMethods(schema.ForeignKeys, naming.EntityNamePlural)

	// Collect imports
	imports := collectImports(schema.Columns)
	data.Imports = imports

	return data, nil
}

// buildEntityFields creates fields for the main entity struct
func buildEntityFields(columns []sqlparser.Column) []FieldInfo {
	var fields []FieldInfo
	for _, col := range columns {
		field := FieldInfo{
			Name:         sqlparser.ToPascalCase(col.Name),
			GoType:       col.GoType,
			DBColumn:     col.Name,
			JSONTag:      col.Name,
			DBTag:        col.Name,
			ValidateTag:  col.ValidationTags,
			Comment:      col.Comment,
			IsPointer:    strings.HasPrefix(col.GoType, "*"),
			IsTime:       strings.Contains(col.GoType, "time.Time"),
			IsJSON:       strings.Contains(col.GoType, "json.RawMessage"),
			IsPrimaryKey: col.IsPrimaryKey,
			IsForeignKey: col.IsForeignKey,
			HasDefault:   col.HasDefault,
		}
		fields = append(fields, field)
	}
	return fields
}

// buildCreateFields creates fields for the Create struct (excludes auto-generated PKs and timestamps)
func buildCreateFields(columns []sqlparser.Column, pk sqlparser.PrimaryKeyInfo) []FieldInfo {
	var fields []FieldInfo
	for _, col := range columns {
		// Skip auto-generated PK
		if col.IsPrimaryKey && col.HasDefault {
			continue
		}

		// Skip auto-generated timestamps
		if col.Name == "created_at" || col.Name == "updated_at" {
			if col.HasDefault {
				continue
			}
		}

		field := FieldInfo{
			Name:         sqlparser.ToPascalCase(col.Name),
			GoType:       col.GoType,
			DBColumn:     col.Name,
			JSONTag:      col.Name,
			DBTag:        col.Name,
			ValidateTag:  col.ValidationTags,
			Comment:      col.Comment,
			IsPointer:    strings.HasPrefix(col.GoType, "*"),
			IsTime:       strings.Contains(col.GoType, "time.Time"),
			IsJSON:       strings.Contains(col.GoType, "json.RawMessage"),
			IsPrimaryKey: col.IsPrimaryKey,
			IsForeignKey: col.IsForeignKey,
		}
		fields = append(fields, field)
	}
	return fields
}

// buildUpdateFields creates fields for the Update struct (all fields optional/pointer)
func buildUpdateFields(columns []sqlparser.Column, pk sqlparser.PrimaryKeyInfo) []FieldInfo {
	var fields []FieldInfo
	for _, col := range columns {
		// Skip PK and auto-timestamps
		if col.IsPrimaryKey {
			continue
		}
		if col.Name == "created_at" || col.Name == "updated_at" {
			continue
		}

		// Make all fields pointers for optional updates
		goType := col.GoType
		if !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") {
			goType = "*" + goType
		}

		field := FieldInfo{
			Name:         sqlparser.ToPascalCase(col.Name),
			GoType:       goType,
			DBColumn:     col.Name,
			JSONTag:      col.Name,
			DBTag:        col.Name,
			ValidateTag:  "", // No validation on update fields (all optional)
			Comment:      col.Comment,
			IsPointer:    true,
			IsTime:       strings.Contains(col.GoType, "time.Time"),
			IsJSON:       strings.Contains(col.GoType, "json.RawMessage"),
			IsForeignKey: col.IsForeignKey,
		}
		fields = append(fields, field)
	}
	return fields
}

// buildFilterFields creates fields for the Filter struct (all optional)
func buildFilterFields(columns []sqlparser.Column, pk sqlparser.PrimaryKeyInfo) []FieldInfo {
	var fields []FieldInfo
	for _, col := range columns {
		// Skip audit timestamps
		if col.Name == "created_at" || col.Name == "updated_at" {
			continue
		}

		// Make all fields pointers for optional filtering
		goType := col.GoType
		if !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") {
			goType = "*" + goType
		}

		field := FieldInfo{
			Name:     sqlparser.ToPascalCase(col.Name),
			GoType:   goType,
			DBColumn: col.Name,
			JSONTag:  col.Name,
			Comment:  "Filter by " + col.Name,
		}
		fields = append(fields, field)
	}
	return fields
}

// buildFKMethods creates FK method info from foreign keys
func buildFKMethods(foreignKeys []sqlparser.ForeignKey, entityNamePlural string) []FKMethodInfo {
	var methods []FKMethodInfo
	for _, fk := range foreignKeys {
		method := FKMethodInfo{
			MethodName:     "List" + fk.MethodSuffix,
			FKColumn:       fk.ColumnName,
			FKGoType:       strings.TrimPrefix(fk.RefColumn, "*"), // Assume string for now
			FKParamName:    sqlparser.ToCamelCase(fk.ColumnName),
			FKGoName:       sqlparser.ToPascalCase(fk.ColumnName),
			RefEntityName:  fk.EntityName,
			RefRepoPackage: fk.RepoPackageName,
			Comment:        fmt.Sprintf("Retrieves %s for a given %s", entityNamePlural, fk.EntityName),
		}

		// Get actual Go type from column
		// For now, default to string for UUIDs
		method.FKGoType = "string"

		methods = append(methods, method)
	}
	return methods
}

// collectImports gathers all necessary imports from columns
func collectImports(columns []sqlparser.Column) []string {
	importSet := make(map[string]bool)

	for _, col := range columns {
		if col.GoImportPath != "" {
			importSet[col.GoImportPath] = true
		}
	}

	imports := []string{}
	for imp := range importSet {
		imports = append(imports, imp)
	}

	return imports
}

// generateFile renders a template and writes it to a file
func generateFile(filepath string, tmplStr string, data interface{}) error {
	funcMap := template.FuncMap{
		"ToPascalCase": sqlparser.ToPascalCase,
		"ToCamelCase":  sqlparser.ToCamelCase,
		"TrimPrefix":   strings.TrimPrefix,
		"Contains":     strings.Contains,
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
