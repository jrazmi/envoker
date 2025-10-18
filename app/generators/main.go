package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jrazmi/envoker/app/generators/bridgegen"
	"github.com/jrazmi/envoker/app/generators/orchestrator"
	"github.com/jrazmi/envoker/app/generators/pgxstores"
	"github.com/jrazmi/envoker/app/generators/repositorygen"
	"github.com/jrazmi/envoker/app/generators/schema"
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
	fmt.Println("Error: repositorygen command has been deprecated.")
	fmt.Println("The SQL parser has been removed. Please use the 'generate' command with JSON schema instead:")
	fmt.Println("  generator generate -json=schema/reflector/output/public.json -table=<table_name>")
	os.Exit(1)
}

func runStoreGen(args []string) {
	fmt.Println("Error: storegen command has been deprecated.")
	fmt.Println("The SQL parser has been removed. Please use the 'generate' command with JSON schema instead:")
	fmt.Println("  generator generate -json=schema/reflector/output/public.json -table=<table_name>")
	os.Exit(1)
}

func runBridgeGen(args []string) {
	fmt.Println("Error: bridgegen command has been deprecated.")
	fmt.Println("The SQL parser has been removed. Please use the 'generate' command with JSON schema instead:")
	fmt.Println("  generator generate -json=schema/reflector/output/public.json -table=<table_name>")
	os.Exit(1)
}

func runGenerateAll(args []string) {
	fs := flag.NewFlagSet("generate-all", flag.ExitOnError)

	sqlFile := fs.String("sql", "", "Path to SQL CREATE TABLE file")
	jsonFile := fs.String("json", "", "Path to reflected JSON schema file")
	tableName := fs.String("table", "", "Table name (required when using -json)")
	generateAll := fs.Bool("all", false, "Generate all tables from JSON file")
	outputDir := fs.String("output", ".", "Base output directory")
	modulePath := fs.String("module", "github.com/jrazmi/envoker", "Go module path")
	force := fs.Bool("force", false, "Overwrite existing files without prompting")
	layers := fs.String("layers", "all", "Comma-separated layers to generate: repository,store,bridge,all")

	fs.Parse(args)

	// Validate that either -sql or -json is provided (but not both)
	if *sqlFile == "" && *jsonFile == "" {
		fmt.Println("Error: either -sql or -json flag is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	if *sqlFile != "" && *jsonFile != "" {
		fmt.Println("Error: cannot specify both -sql and -json flags")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// If using JSON, table name is required unless -all is specified
	if *jsonFile != "" && *tableName == "" && !*generateAll {
		fmt.Println("Error: -table flag is required when using -json (or use -all to generate all tables)")
		fs.PrintDefaults()
		os.Exit(1)
	}

	var parseResults []*sqlparser.ParseResult

	// Load schema from SQL or JSON
	if *sqlFile != "" {
		// Read SQL file
		content, err := os.ReadFile(*sqlFile)
		if err != nil {
			fmt.Printf("Error reading SQL file: %v\n", err)
			os.Exit(1)
		}

		// Parse SQL
		parseResult, err := sqlparser.Parse(string(content))
		if err != nil {
			fmt.Printf("Error parsing SQL: %v\n", err)
			os.Exit(1)
		}
		parseResults = append(parseResults, parseResult)
	} else if *generateAll {
		// Load ALL tables from JSON
		fmt.Printf("📖 Loading ALL tables from JSON schema: %s\n", *jsonFile)
		tables, err := jsonschema.ListTables(*jsonFile)
		if err != nil {
			fmt.Printf("Error listing tables: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found %d tables: %v\n\n", len(tables), tables)
		for _, table := range tables {
			parseResult, err := jsonschema.LoadTableFromJSON(*jsonFile, table)
			if err != nil {
				fmt.Printf("Error loading table %s: %v\n", table, err)
				os.Exit(1)
			}
			parseResults = append(parseResults, parseResult)
		}
	} else {
		// Load single table from JSON
		fmt.Printf("📖 Loading table '%s' from JSON schema: %s\n", *tableName, *jsonFile)
		parseResult, err := jsonschema.LoadTableFromJSON(*jsonFile, *tableName)
		if err != nil {
			fmt.Printf("Error loading JSON schema: %v\n", err)
			os.Exit(1)
		}
		parseResults = append(parseResults, parseResult)
	}

	// Parse layers
	layerList := []string{"all"}
	if *layers != "all" {
		layerList = strings.Split(*layers, ",")
	}

	// Generate all layers for each table
	config := orchestrator.Config{
		ModulePath:     *modulePath,
		OutputDir:      *outputDir,
		ForceOverwrite: *force,
		Layers:         layerList,
	}

	fmt.Println("🚀 Starting full-stack generation...")
	fmt.Println()

	for _, parseResult := range parseResults {
		fmt.Printf("\n═══════════════════════════════════════════════════════════════\n")
		fmt.Printf("Generating: %s\n", parseResult.Schema.Name)
		fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

		result, err := orchestrator.GenerateAll(parseResult, config)
		if err != nil {
			fmt.Printf("\n❌ Generation failed for %s: %v\n", parseResult.Schema.Name, err)
			if result != nil && len(result.Errors) > 0 {
				fmt.Println("\nErrors:")
				for _, e := range result.Errors {
					fmt.Printf("  - %v\n", e)
				}
			}
			continue // Continue with next table
		}

		// Print summary
		orchestrator.PrintSummary(result, parseResult.Schema.Name)
	}

	fmt.Println("\n🎉 All tables generated successfully!")
}

func printUsage() {
	fmt.Println("Generator - Code generation tool for taskmaster")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  generator <command> [flags]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  generate       🚀 Generate ALL layers from SQL or JSON (recommended!)")
	fmt.Println("  repositorygen  Generate repository layer from SQL CREATE TABLE")
	fmt.Println("  storegen       Generate pgx store layer from SQL CREATE TABLE")
	fmt.Println("  bridgegen      Generate bridge/API layer from SQL CREATE TABLE")
	fmt.Println("  pgxstore       Generate a pgx-based PostgreSQL store (legacy)")
	fmt.Println("  help           Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate from SQL file (legacy)")
	fmt.Println("  generator generate -sql=schema/pgmigrations/001_initial_schema.sql")
	fmt.Println()
	fmt.Println("  # Generate single table from reflected JSON (recommended)")
	fmt.Println("  generator generate -json=schema/reflector/output/public.json -table=api_keys")
	fmt.Println()
	fmt.Println("  # Generate ALL tables from reflected JSON")
	fmt.Println("  generator generate -json=schema/reflector/output/public.json -all")
	fmt.Println()
	fmt.Println("  # With force overwrite")
	fmt.Println("  generator generate -json=schema/reflector/output/public.json -table=tasks -force")
	fmt.Println()
	fmt.Println("  # Generate only specific layers")
	fmt.Println("  generator generate -json=schema/reflector/output/public.json -table=tasks -layers=repository,store")
	fmt.Println()
	fmt.Println("  # Individual layer generation (still uses SQL)")
	fmt.Println("  generator repositorygen -sql=schema/tasks.sql")
	fmt.Println("  generator storegen -sql=schema/tasks.sql")
	fmt.Println("  generator bridgegen -sql=schema/tasks.sql")
	fmt.Println()
	fmt.Println("For command-specific help:")
	fmt.Println("  generator <command> -h")
}
