package pgxstores

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Config struct {
	Entity      string
	Table       string
	Schema      string
	PK          string
	PackageName string
	ModulePath  string
	// Derived fields
	Create    string
	Update    string
	Filter    string
	StoreType string
}

type Field struct {
	Name       string
	DBColumn   string
	GoType     string
	IsPointer  bool
	IsNullable bool
}

func Generate(entity, table, schema, pk, modulePath string) error {
	cfg := Config{
		Entity:     entity,
		Table:      table,
		Schema:     schema,
		PK:         pk,
		ModulePath: modulePath,
		Create:     fmt.Sprintf("Create%s", entity),
		Update:     fmt.Sprintf("Update%s", entity),
		Filter:     "QueryFilter",
		StoreType:  "Store",
	}

	fmt.Printf("Generating store for: %s\n", cfg.Entity)
	fmt.Printf("  Table: %s.%s\n", cfg.Schema, cfg.Table)
	fmt.Printf("  Primary key: %s\n", cfg.PK)
	fmt.Printf("  Derived types:\n")
	fmt.Printf("    Create: %s\n", cfg.Create)
	fmt.Printf("    Update: %s\n", cfg.Update)
	fmt.Printf("    Filter: %s\n", cfg.Filter)

	// Extract fields from parent repository package
	fields, err := extractFieldsFromRepo(cfg)
	if err != nil {
		return fmt.Errorf("extract fields: %w", err)
	}

	// Generate store_gen.go
	if err := generateStore(cfg, fields); err != nil {
		return fmt.Errorf("generate store: %w", err)
	}

	return nil
}

func extractFieldsFromRepo(cfg Config) (map[string][]Field, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	fmt.Printf("Current directory: %s\n", cwd)

	// From core/repositories/usersrepo/stores/userspgxstore -> core/repositories/usersrepo
	repoDir := "../.."
	absRepoDir, err := filepath.Abs(repoDir)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Looking for repository package in: %s\n", absRepoDir)

	if _, err := os.Stat(absRepoDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository directory does not exist: %s", absRepoDir)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, repoDir, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing repo package %s: %w", repoDir, err)
	}

	fmt.Printf("Found %d packages\n", len(pkgs))

	result := make(map[string][]Field)

	// Find our structs
	for pkgName, pkg := range pkgs {
		fmt.Printf("Scanning package: %s (%d files)\n", pkgName, len(pkg.Files))
		for fileName, file := range pkg.Files {
			fmt.Printf("  Scanning file: %s\n", fileName)
			ast.Inspect(file, func(n ast.Node) bool {
				typeSpec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					return true
				}

				typeName := typeSpec.Name.Name
				fmt.Printf("    Found struct: %s\n", typeName)

				switch typeName {
				case cfg.Entity, cfg.Create, cfg.Update:
					fmt.Printf("      -> Matches target type!\n")
					result[typeName] = parseStructFields(structType)
				}

				return true
			})
		}
	}

	return result, nil
}

func parseStructFields(structType *ast.StructType) []Field {
	var fields []Field

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue // Skip embedded fields
		}

		dbTag := extractDBTag(field.Tag)
		if dbTag == "" || dbTag == "-" {
			continue
		}

		f := Field{
			Name:     field.Names[0].Name,
			DBColumn: dbTag,
			GoType:   formatType(field.Type),
		}

		// Check if pointer
		if _, ok := field.Type.(*ast.StarExpr); ok {
			f.IsPointer = true
			f.IsNullable = true
		}

		fields = append(fields, f)
	}

	return fields
}

func filterFieldsExcludingPK(fields []Field, pk string) []Field {
	var result []Field
	for _, f := range fields {
		if f.DBColumn != pk {
			result = append(result, f)
		}
	}
	return result
}

func extractDBTag(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}

	tagStr := tag.Value[1 : len(tag.Value)-1] // Remove backticks

	for _, part := range strings.Split(tagStr, " ") {
		if strings.HasPrefix(part, `db:"`) {
			return strings.Trim(part[4:], `"`)
		}
	}

	return ""
}

func formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + formatType(t.X)
	case *ast.SelectorExpr:
		return formatType(t.X) + "." + t.Sel.Name
	default:
		return "interface{}"
	}
}

func generateStore(cfg Config, fields map[string][]Field) error {
	// Validate we found the fields
	if len(fields[cfg.Entity]) == 0 {
		return fmt.Errorf("no fields found for entity type: %s", cfg.Entity)
	}
	if len(fields[cfg.Create]) == 0 {
		return fmt.Errorf("no fields found for create type: %s", cfg.Create)
	}
	if len(fields[cfg.Update]) == 0 {
		return fmt.Errorf("no fields found for update type: %s", cfg.Update)
	}

	// Determine package names
	repoPackage, err := determineRepoPackage()
	if err != nil {
		return err
	}
	storePackage, err := determineStorePackage()
	if err != nil {
		return err
	}

	fmt.Printf("  Repository package: %s\n", repoPackage)
	fmt.Printf("  Store package: %s\n", storePackage)

	outFile := "store_gen.go"

	tmpl := template.Must(template.New("store").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(storeTemplate))

	// Check if PK is in CreateFields (if so, we don't auto-generate it)
	pkInCreate := false
	var pkField Field
	for _, f := range fields[cfg.Create] {
		if f.DBColumn == cfg.PK {
			pkInCreate = true
			pkField = f
			break
		}
	}

	// If PK is NOT in Create struct, we'll generate it, so exclude it from CreateFields
	// If PK IS in Create struct, keep it in CreateFields (caller provides it or DB generates)
	createFields := fields[cfg.Create]
	if !pkInCreate {
		createFields = filterFieldsExcludingPK(fields[cfg.Create], cfg.PK)
	}

	data := struct {
		Config
		RepoPackage  string
		StorePackage string
		EntityFields []Field
		CreateFields []Field
		UpdateFields []Field
		PKInCreate   bool
		PKField      Field
	}{
		Config:       cfg,
		RepoPackage:  repoPackage,
		StorePackage: storePackage,
		EntityFields: fields[cfg.Entity],
		CreateFields: createFields,
		UpdateFields: fields[cfg.Update],
		PKInCreate:   pkInCreate,
		PKField:      pkField,
	}

	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tmpl.Execute(f, data)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully generated: %s\n", outFile)
	return nil
}

func determineRepoPackage() (string, error) {
	// Get parent parent directory name (e.g., "usersrepo")
	// From stores/userspgxstore -> usersrepo
	parentDir, err := filepath.Abs("../..")
	if err != nil {
		return "", err
	}
	return filepath.Base(parentDir), nil
}

func determineStorePackage() (string, error) {
	// Get current directory name (e.g., "userspgxstore")
	currentDir, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	return filepath.Base(currentDir), nil
}
