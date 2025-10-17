package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jrazmi/envoker/app/envoker/generators/bridgegen"
	"github.com/jrazmi/envoker/app/envoker/generators/orchestrator"
	"github.com/jrazmi/envoker/app/envoker/generators/pgxstores"
	"github.com/jrazmi/envoker/app/envoker/generators/repositorygen"
	"github.com/jrazmi/envoker/app/envoker/generators/sqlparser"
)

func main() {
	// Define subcommands
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "pgxstore":
		runPgxStore(os.Args[2:])
	case "repositorygen":
		runRepositoryGen(os.Args[2:])
	case "storegen":
		runStoreGen(os.Args[2:])
	case "bridgegen":
		runBridgeGen(os.Args[2:])
	case "generate", "generate-all", "all":
		runGenerateAll(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func runPgxStore(args []string) {
	fs := flag.NewFlagSet("pgxstore", flag.ExitOnError)

	entity := fs.String("entity", "", "Entity type name (e.g., User)")
	table := fs.String("table", "", "Database table name")
	schema := fs.String("schema", "public", "Database schema")
	pk := fs.String("pk", "id", "Primary key field name")
	modulePath := fs.String("module", "github.com/jrazmi/envoker", "Go module path")

	fs.Parse(args)

	// Validate required flags
	if *entity == "" {
		fmt.Println("Error: -entity flag is required")
		fs.PrintDefaults()
		os.Exit(1)
	}
	if *table == "" {
		fmt.Println("Error: -table flag is required")
		fs.PrintDefaults()
		os.Exit(1)
	}
	if *pk == "" {
		fmt.Println("Error: -pk flag is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	err := pgxstores.Generate(*entity, *table, *schema, *pk, *modulePath)
	if err != nil {
		fmt.Printf("Error generating store: %v\n", err)
		os.Exit(1)
	}
}

func runRepositoryGen(args []string) {
	fs := flag.NewFlagSet("repositorygen", flag.ExitOnError)

	sqlFile := fs.String("sql", "", "Path to SQL CREATE TABLE file")
	outputDir := fs.String("output", ".", "Base output directory")
	modulePath := fs.String("module", "github.com/jrazmi/envoker", "Go module path")
	force := fs.Bool("force", false, "Overwrite existing files without prompting")

	fs.Parse(args)

	// Validate required flags
	if *sqlFile == "" {
		fmt.Println("Error: -sql flag is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// Read SQL file
	sqlContent, err := os.ReadFile(*sqlFile)
	if err != nil {
		fmt.Printf("Error reading SQL file: %v\n", err)
		os.Exit(1)
	}

	// Parse SQL
	parseResult, err := sqlparser.Parse(string(sqlContent))
	if err != nil {
		fmt.Printf("Error parsing SQL: %v\n", err)
		os.Exit(1)
	}

	// Analyze and enrich
	if err := sqlparser.Analyze(parseResult); err != nil {
		fmt.Printf("Error analyzing SQL: %v\n", err)
		os.Exit(1)
	}

	// Generate repository files
	config := repositorygen.Config{
		ModulePath:     *modulePath,
		OutputDir:      *outputDir,
		ForceOverwrite: *force,
	}

	result, err := repositorygen.Generate(parseResult, config)
	if err != nil {
		fmt.Printf("Error generating repository: %v\n", err)
		for _, e := range result.Errors {
			fmt.Printf("  - %v\n", e)
		}
		os.Exit(1)
	}

	// Print success
	fmt.Println("Repository generated successfully:")
	fmt.Printf("  Model:      %s\n", result.ModelFile)
	fmt.Printf("  Repository: %s\n", result.RepositoryFile)

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
}

func runStoreGen(args []string) {
	fs := flag.NewFlagSet("storegen", flag.ExitOnError)

	sqlFile := fs.String("sql", "", "Path to SQL CREATE TABLE file")
	outputDir := fs.String("output", ".", "Base output directory")
	modulePath := fs.String("module", "github.com/jrazmi/envoker", "Go module path")
	force := fs.Bool("force", false, "Overwrite existing files without prompting")

	fs.Parse(args)

	// Validate required flags
	if *sqlFile == "" {
		fmt.Println("Error: -sql flag is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// Read SQL file
	sqlContent, err := os.ReadFile(*sqlFile)
	if err != nil {
		fmt.Printf("Error reading SQL file: %v\n", err)
		os.Exit(1)
	}

	// Parse SQL
	parseResult, err := sqlparser.Parse(string(sqlContent))
	if err != nil {
		fmt.Printf("Error parsing SQL: %v\n", err)
		os.Exit(1)
	}

	// Analyze and enrich
	if err := sqlparser.Analyze(parseResult); err != nil {
		fmt.Printf("Error analyzing SQL: %v\n", err)
		os.Exit(1)
	}

	// Generate store files
	config := pgxstores.SQLConfig{
		ModulePath:     *modulePath,
		OutputDir:      *outputDir,
		ForceOverwrite: *force,
	}

	storeFile, err := pgxstores.GenerateFromSQL(parseResult, config)
	if err != nil {
		fmt.Printf("Error generating store: %v\n", err)
		os.Exit(1)
	}

	// Print success
	fmt.Println("Store generated successfully:")
	fmt.Printf("  Store: %s\n", storeFile)
}

func runBridgeGen(args []string) {
	fs := flag.NewFlagSet("bridgegen", flag.ExitOnError)

	sqlFile := fs.String("sql", "", "Path to SQL CREATE TABLE file")
	outputDir := fs.String("output", ".", "Base output directory")
	modulePath := fs.String("module", "github.com/jrazmi/envoker", "Go module path")
	force := fs.Bool("force", false, "Overwrite existing files without prompting")

	fs.Parse(args)

	// Validate required flags
	if *sqlFile == "" {
		fmt.Println("Error: -sql flag is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// Read SQL file
	sqlContent, err := os.ReadFile(*sqlFile)
	if err != nil {
		fmt.Printf("Error reading SQL file: %v\n", err)
		os.Exit(1)
	}

	// Parse SQL
	parseResult, err := sqlparser.Parse(string(sqlContent))
	if err != nil {
		fmt.Printf("Error parsing SQL: %v\n", err)
		os.Exit(1)
	}

	// Analyze and enrich
	if err := sqlparser.Analyze(parseResult); err != nil {
		fmt.Printf("Error analyzing SQL: %v\n", err)
		os.Exit(1)
	}

	// Generate bridge files
	config := bridgegen.Config{
		ModulePath:     *modulePath,
		OutputDir:      *outputDir,
		ForceOverwrite: *force,
	}

	result, err := bridgegen.Generate(parseResult, config)
	if err != nil {
		fmt.Printf("Error generating bridge: %v\n", err)
		for _, e := range result.Errors {
			fmt.Printf("  - %v\n", e)
		}
		os.Exit(1)
	}

	// Print success
	fmt.Println("Bridge generated successfully:")
	fmt.Printf("  Bridge:  %s\n", result.BridgeFile)
	fmt.Printf("  HTTP:    %s\n", result.HTTPFile)
	fmt.Printf("  Model:   %s\n", result.ModelFile)
	fmt.Printf("  Marshal: %s\n", result.MarshalFile)
	fmt.Printf("  FOP:     %s\n", result.FOPFile)

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
}

func runGenerateAll(args []string) {
	fs := flag.NewFlagSet("generate-all", flag.ExitOnError)

	sqlFile := fs.String("sql", "", "Path to SQL CREATE TABLE file")
	outputDir := fs.String("output", ".", "Base output directory")
	modulePath := fs.String("module", "github.com/jrazmi/envoker", "Go module path")
	force := fs.Bool("force", false, "Overwrite existing files without prompting")
	layers := fs.String("layers", "all", "Comma-separated layers to generate: repository,store,bridge,all")

	fs.Parse(args)

	// Validate required flags
	if *sqlFile == "" {
		fmt.Println("Error: -sql flag is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// Read SQL file
	sqlContent, err := os.ReadFile(*sqlFile)
	if err != nil {
		fmt.Printf("Error reading SQL file: %v\n", err)
		os.Exit(1)
	}

	// Parse layers
	layerList := []string{"all"}
	if *layers != "all" {
		layerList = strings.Split(*layers, ",")
	}

	// Generate all layers
	config := orchestrator.Config{
		ModulePath:     *modulePath,
		OutputDir:      *outputDir,
		ForceOverwrite: *force,
		Layers:         layerList,
	}

	fmt.Println("🚀 Starting full-stack generation...")
	fmt.Println()

	result, err := orchestrator.GenerateAll(string(sqlContent), config)
	if err != nil {
		fmt.Printf("\n❌ Generation failed: %v\n", err)
		if result != nil && len(result.Errors) > 0 {
			fmt.Println("\nErrors:")
			for _, e := range result.Errors {
				fmt.Printf("  - %v\n", e)
			}
		}
		os.Exit(1)
	}

	// Parse result to get table name
	parseResult, _ := sqlparser.Parse(string(sqlContent))
	tableName := parseResult.Schema.Name

	// Print summary
	orchestrator.PrintSummary(result, tableName)
}

func printUsage() {
	fmt.Println("Generator - Code generation tool for taskmaster")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  generator <command> [flags]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  generate       🚀 Generate ALL layers from SQL (recommended!)")
	fmt.Println("  repositorygen  Generate repository layer from SQL CREATE TABLE")
	fmt.Println("  storegen       Generate pgx store layer from SQL CREATE TABLE")
	fmt.Println("  bridgegen      Generate bridge/API layer from SQL CREATE TABLE")
	fmt.Println("  pgxstore       Generate a pgx-based PostgreSQL store (legacy)")
	fmt.Println("  help           Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate complete full-stack (Repository + Store + Bridge)")
	fmt.Println("  generator generate -sql=schema/tasks.sql")
	fmt.Println()
	fmt.Println("  # With force overwrite")
	fmt.Println("  generator generate -sql=schema/tasks.sql -force")
	fmt.Println()
	fmt.Println("  # Generate only specific layers")
	fmt.Println("  generator generate -sql=schema/tasks.sql -layers=repository,store")
	fmt.Println()
	fmt.Println("  # Individual layer generation")
	fmt.Println("  generator repositorygen -sql=schema/tasks.sql")
	fmt.Println("  generator storegen -sql=schema/tasks.sql")
	fmt.Println("  generator bridgegen -sql=schema/tasks.sql")
	fmt.Println()
	fmt.Println("For command-specific help:")
	fmt.Println("  generator <command> -h")
}
