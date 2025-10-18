package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jrazmi/envoker/app/generators/orchestrator"
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

func runGenerateAll(args []string) {
	fs := flag.NewFlagSet("generate-all", flag.ExitOnError)

	// Auto-detect module path from go.mod
	defaultModulePath := getModulePath()

	jsonFile := fs.String("json", "", "Path to reflected JSON schema file (required)")
	tableName := fs.String("table", "", "Table name (required unless using -all)")
	generateAll := fs.Bool("all", false, "Generate all tables from JSON file")
	outputDir := fs.String("output", ".", "Base output directory")
	modulePath := fs.String("module", defaultModulePath, "Go module path (auto-detected from go.mod)")
	force := fs.Bool("force", false, "Overwrite existing files without prompting")
	layers := fs.String("layers", "all", "Comma-separated layers to generate: repository,store,bridge,all")

	fs.Parse(args)

	// Validate that -json is provided
	if *jsonFile == "" {
		fmt.Println("Error: -json flag is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// If using JSON, table name is required unless -all is specified
	if *tableName == "" && !*generateAll {
		fmt.Println("Error: -table flag is required when using -json (or use -all to generate all tables)")
		fs.PrintDefaults()
		os.Exit(1)
	}

	var tableDefinitions []*schema.TableDefinition

	// Load schema from JSON
	if *generateAll {
		// Load ALL tables from JSON
		fmt.Printf("📖 Loading ALL tables from JSON schema: %s\n", *jsonFile)
		tables, err := schema.ListTables(*jsonFile)
		if err != nil {
			fmt.Printf("Error listing tables: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found %d tables: %v\n\n", len(tables), tables)
		for _, table := range tables {
			tableDef, err := schema.LoadTableFromJSON(*jsonFile, table)
			if err != nil {
				fmt.Printf("Error loading table %s: %v\n", table, err)
				os.Exit(1)
			}
			tableDefinitions = append(tableDefinitions, tableDef)
		}
	} else {
		// Load single table from JSON
		fmt.Printf("📖 Loading table '%s' from JSON schema: %s\n", *tableName, *jsonFile)
		tableDef, err := schema.LoadTableFromJSON(*jsonFile, *tableName)
		if err != nil {
			fmt.Printf("Error loading JSON schema: %v\n", err)
			os.Exit(1)
		}
		tableDefinitions = append(tableDefinitions, tableDef)
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

	for _, tableDef := range tableDefinitions {
		fmt.Printf("\n═══════════════════════════════════════════════════════════════\n")
		fmt.Printf("Generating: %s\n", tableDef.Schema.Name)
		fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

		result, err := orchestrator.GenerateAll(tableDef, config)
		if err != nil {
			fmt.Printf("\n❌ Generation failed for %s: %v\n", tableDef.Schema.Name, err)
			if result != nil && len(result.Errors) > 0 {
				fmt.Println("\nErrors:")
				for _, e := range result.Errors {
					fmt.Printf("  - %v\n", e)
				}
			}
			continue // Continue with next table
		}

		// Print summary
		orchestrator.PrintSummary(result, tableDef.Schema.Name)
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
	fmt.Println("  generate       🚀 Generate ALL layers from JSON schema (recommended!)")
	fmt.Println("  help           Show this help message")
	fmt.Println()
	fmt.Println("Note: pgxstore, repositorygen, storegen, and bridgegen have been deprecated.")
	fmt.Println("Use 'generate' command with JSON schema instead.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate single table from reflected JSON")
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
	fmt.Println("For command-specific help:")
	fmt.Println("  generator <command> -h")
}

// getModulePath reads the module path from go.mod file
func getModulePath() string {
	// Default fallback
	defaultPath := "github.com/jrazmi/envoker"

	// Try to find go.mod in current directory or parent directories
	goModPath, err := findGoMod()
	if err != nil {
		return defaultPath
	}

	file, err := os.Open(goModPath)
	if err != nil {
		return defaultPath
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			// Extract module path (everything after "module ")
			modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			if modulePath != "" {
				return modulePath
			}
		}
	}

	return defaultPath
}

// findGoMod searches for go.mod file in current and parent directories
func findGoMod() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Search up to 5 levels up
	for i := 0; i < 5; i++ {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return goModPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found")
}
