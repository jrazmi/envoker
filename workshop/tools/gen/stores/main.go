package main

import (
	"flag"
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
	// Derived fields
	Create    string
	Update    string
	Filter    string
	StoreType string
}

func main() {
	cfg := parseFlags()

	fmt.Printf("Generating store for: %s\n", cfg.Entity)
	fmt.Printf("  Table: %s.%s\n", cfg.Schema, cfg.Table)
	fmt.Printf("  Primary key: %s\n", cfg.PK)

	// Extract fields from parent repository package
	fields := extractFieldsFromRepo(cfg)

	// Generate store_gen.go
	generateStore(cfg, fields)
}

func parseFlags() Config {
	cfg := Config{}

	flag.StringVar(&cfg.Entity, "entity", "", "Entity type name (e.g., User)")
	flag.StringVar(&cfg.Table, "table", "", "Database table name")
	flag.StringVar(&cfg.Schema, "schema", "public", "Database schema")
	flag.StringVar(&cfg.PK, "pk", "id", "Primary key field name")

	flag.Parse()

	// Validate required flags
	if cfg.Entity == "" {
		panic("missing required flag: -entity")
	}
	if cfg.Table == "" {
		panic("missing required flag: -table")
	}
	if cfg.PK == "" {
		panic("missing required flag: -pk")
	}

	// Derive conventional names
	cfg.Create = fmt.Sprintf("Create%s", cfg.Entity)
	cfg.Update = fmt.Sprintf("Update%s", cfg.Entity)
	cfg.Filter = fmt.Sprintf("%sFilter", cfg.Entity)
	cfg.StoreType = "Store"
	cfg.PackageName = fmt.Sprintf("%spgxstore")

	fmt.Printf("  Derived types:\n")
	fmt.Printf("    Create: %s\n", cfg.Create)
	fmt.Printf("    Update: %s\n", cfg.Update)
	fmt.Printf("    Filter: %s\n", cfg.Filter)

	return cfg
}

func extractFieldsFromRepo(cfg Config) map[string][]Field {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Current directory: %s\n", cwd)

	// From core/repositories/usersrepo/stores/userspgxstore -> core/repositories/usersrepo
	repoDir := "../.."
	absRepoDir, err := filepath.Abs(repoDir)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Looking for repository package in: %s\n", absRepoDir)

	if _, err := os.Stat(absRepoDir); os.IsNotExist(err) {
		panic(fmt.Errorf("repository directory does not exist: %s", absRepoDir))
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, repoDir, nil, parser.ParseComments)
	if err != nil {
		panic(fmt.Errorf("parsing repo package %s: %w", repoDir, err))
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

	return result
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

type Field struct {
	Name       string
	DBColumn   string
	GoType     string
	IsPointer  bool
	IsNullable bool
}

func generateStore(cfg Config, fields map[string][]Field) {
	// Validate we found the fields
	if len(fields[cfg.Entity]) == 0 {
		panic(fmt.Errorf("no fields found for entity type: %s", cfg.Entity))
	}
	if len(fields[cfg.Create]) == 0 {
		panic(fmt.Errorf("no fields found for create type: %s", cfg.Create))
	}
	if len(fields[cfg.Update]) == 0 {
		panic(fmt.Errorf("no fields found for update type: %s", cfg.Update))
	}

	// Determine package names
	repoPackage := determineRepoPackage()
	storePackage := determineStorePackage()

	fmt.Printf("  Repository package: %s\n", repoPackage)
	fmt.Printf("  Store package: %s\n", storePackage)

	outFile := "store_gen.go"

	tmpl := template.Must(template.New("store").Parse(storeTemplate))

	data := struct {
		Config
		RepoPackage  string
		StorePackage string
		EntityFields []Field
		CreateFields []Field
		UpdateFields []Field
	}{
		Config:       cfg,
		RepoPackage:  repoPackage,
		StorePackage: storePackage,
		EntityFields: fields[cfg.Entity],
		CreateFields: fields[cfg.Create],
		UpdateFields: fields[cfg.Update],
	}

	f, err := os.Create(outFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = tmpl.Execute(f, data)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Successfully generated: %s\n", outFile)
}

func determineRepoPackage() string {
	// Get parent parent directory name (e.g., "usersrepo")
	// From stores/userspgxstore -> usersrepo
	parentDir, err := filepath.Abs("../..")
	if err != nil {
		panic(err)
	}
	return filepath.Base(parentDir)
}

func determineStorePackage() string {
	// Get current directory name (e.g., "userspgxstore")
	currentDir, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}
	return filepath.Base(currentDir)
}

const storeTemplate = `// Code generated by storegen. DO NOT EDIT.

package {{.Table}}pgxstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jrazmi/envoker/core/repositories/{{.RepoPackage}}"
	"github.com/jrazmi/envoker/core/scaffolding/fop"
	"github.com/jrazmi/envoker/infrastructure/databases/postgresdb"
	"github.com/jrazmi/envoker/sdk/cryptids"
)

// Get retrieves a single {{.Entity}} by ID
func (s *{{.StoreType}}) Get(ctx context.Context, ID string, filter {{.RepoPackage}}.{{.Filter}}) ({{.RepoPackage}}.{{.Entity}}, error) {
	query := ` + "`SELECT {{range $i, $f := .EntityFields}}{{if $i}}, {{end}}{{$f.DBColumn}}{{end}} FROM {{.Schema}}.{{.Table}} WHERE {{.PK}} = @{{.PK}}`" + `
	
	args := pgx.NamedArgs{
		"{{.PK}}": ID,
	}
	
	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return {{.RepoPackage}}.{{.Entity}}{}, postgresdb.HandlePgError(err)
	}
	defer rows.Close()
	
	record, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[{{.RepoPackage}}.{{.Entity}}])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return {{.RepoPackage}}.{{.Entity}}{}, {{.RepoPackage}}.Err{{.Entity}}NotFound
		}
		return {{.RepoPackage}}.{{.Entity}}{}, postgresdb.HandlePgError(err)
	}
	
	return record, nil
}

// Create inserts a new {{.Entity}}
func (s *{{.StoreType}}) Create(ctx context.Context, input *{{.RepoPackage}}.{{.Create}}) ({{.RepoPackage}}.{{.Entity}}, error) {
	// Generate ID using cryptids
	id, err := cryptids.GenerateID()
	if err != nil {
		return {{.RepoPackage}}.{{.Entity}}{}, fmt.Errorf("generate id: %w", err)
	}

	query := ` + "`INSERT INTO {{.Schema}}.{{.Table}} ({{.PK}}, {{range $i, $f := .CreateFields}}{{if $i}}, {{end}}{{$f.DBColumn}}{{end}}) VALUES (@{{.PK}}, {{range $i, $f := .CreateFields}}{{if $i}}, {{end}}@{{$f.DBColumn}}{{end}}) RETURNING *`" + `

	args := pgx.NamedArgs{
		"{{.PK}}": id,
{{- range .CreateFields}}
		"{{.DBColumn}}": input.{{.Name}},
{{- end}}
	}

	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return {{.RepoPackage}}.{{.Entity}}{}, postgresdb.HandlePgError(err)
	}
	defer rows.Close()

	record, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[{{.RepoPackage}}.{{.Entity}}])
	if err != nil {
		return {{.RepoPackage}}.{{.Entity}}{}, postgresdb.HandlePgError(err)
	}

	return record, nil
}

// Update modifies an existing {{.Entity}}
func (s *{{.StoreType}}) Update(ctx context.Context, ID string, input *{{.RepoPackage}}.{{.Update}}) error {
	var fields []string
	data := pgx.NamedArgs{
		"{{.PK}}": ID,
	}
	
{{range .UpdateFields}}
	{{if eq .DBColumn "updated_at"}}// Handle updated_at specially - always set it
	now := time.Now()
	if input.{{.Name}} != nil {
		data["{{.DBColumn}}"] = *input.{{.Name}}
	} else {
		data["{{.DBColumn}}"] = now
	}
	fields = append(fields, "{{.DBColumn}} = @{{.DBColumn}}")
	{{else}}{{if .IsNullable}}if input.{{.Name}} != nil {
		fields = append(fields, "{{.DBColumn}} = @{{.DBColumn}}")
		data["{{.DBColumn}}"] = input.{{.Name}}
	}
	{{else}}fields = append(fields, "{{.DBColumn}} = @{{.DBColumn}}")
	data["{{.DBColumn}}"] = input.{{.Name}}
	{{end}}{{end}}
{{end}}
	
	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}
	
	query := fmt.Sprintf(` + "`UPDATE {{.Schema}}.{{.Table}} SET %s WHERE {{.PK}} = @{{.PK}}`" + `, strings.Join(fields, ", "))
	
	result, err := s.pool.Exec(ctx, query, data)
	if err != nil {
		return postgresdb.HandlePgError(err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("{{.Entity}} not found: %s", ID)
	}
	
	return nil
}

// Delete removes a {{.Entity}}
func (s *{{.StoreType}}) Delete(ctx context.Context, ID string) error {
	query := ` + "`DELETE FROM {{.Schema}}.{{.Table}} WHERE {{.PK}} = @{{.PK}}`" + `

	args := pgx.NamedArgs{
		"{{.PK}}": ID,
	}

	result, err := s.pool.Exec(ctx, query, args)
	if err != nil {
		return postgresdb.HandlePgError(err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("{{.Entity}} not found: %s", ID)
	}

	return nil
}

// List retrieves multiple {{.Entity}} records with filtering, ordering, and pagination
func (s *{{.StoreType}}) List(ctx context.Context, filter {{.RepoPackage}}.{{.Filter}}, orderBy fop.By, page fop.PageStringCursor) ([]{{.RepoPackage}}.{{.Entity}}, error) {
	// TODO: Implement filtering, ordering, and pagination logic
	query := ` + "`SELECT {{range $i, $f := .EntityFields}}{{if $i}}, {{end}}{{$f.DBColumn}}{{end}} FROM {{.Schema}}.{{.Table}} LIMIT 100`" + `

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, postgresdb.HandlePgError(err)
	}
	defer rows.Close()

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[{{.RepoPackage}}.{{.Entity}}])
	if err != nil {
		return nil, postgresdb.HandlePgError(err)
	}

	return records, nil
}

// Archive soft-deletes a {{.Entity}} by setting archived_at
func (s *{{.StoreType}}) Archive(ctx context.Context, ID string) error {
	query := ` + "`UPDATE {{.Schema}}.{{.Table}} SET archived_at = @archived_at WHERE {{.PK}} = @{{.PK}}`" + `

	args := pgx.NamedArgs{
		"{{.PK}}":        ID,
		"archived_at": time.Now(),
	}

	result, err := s.pool.Exec(ctx, query, args)
	if err != nil {
		return postgresdb.HandlePgError(err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("{{.Entity}} not found: %s", ID)
	}

	return nil
}
`
