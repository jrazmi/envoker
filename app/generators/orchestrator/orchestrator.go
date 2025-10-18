package orchestrator

import (
	"fmt"
	"strings"
	"time"

	"github.com/jrazmi/envoker/app/generators/bridgegen"
	"github.com/jrazmi/envoker/app/generators/pgxstores"
	"github.com/jrazmi/envoker/app/generators/repositorygen"
	"github.com/jrazmi/envoker/app/generators/schema"
)

// Config holds configuration for the orchestrator
type Config struct {
	ModulePath     string
	OutputDir      string
	ForceOverwrite bool
	Layers         []string // Which layers to generate: "repository", "store", "bridge", or "all"
}

// Result holds the complete generation results
type Result struct {
	RepositoryResult *repositorygen.GenerateResult
	StoreResult      string // Store file path
	BridgeResult     *bridgegen.GenerateResult
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
	Errors           []error
	Warnings         []string
}

// GenerateAll orchestrates the generation of all layers from a TableDefinition
func GenerateAll(tableDef *schema.TableDefinition, config Config) (*Result, error) {
	result := &Result{
		StartTime: time.Now(),
	}

	// Analyze and enrich (if not already done)
	fmt.Println("🧠 Analyzing schema and deriving metadata...")
	if err := schema.AnalyzeTableDefinition(tableDef); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("analyze schema: %w", err))
		return result, err
	}

	// Validate
	if errs := schema.ValidateSchema(tableDef.Schema); len(errs) > 0 {
		for _, e := range errs {
			result.Errors = append(result.Errors, e)
		}
		return result, fmt.Errorf("schema validation failed")
	}

	fmt.Printf("✅ Schema validated: %s (PK: %s)\n",
		tableDef.Schema.Name,
		tableDef.Schema.PrimaryKey.ColumnName,
	)

	// Determine which layers to generate
	layers := config.Layers
	if len(layers) == 0 || contains(layers, "all") {
		layers = []string{"repository", "store", "bridge"}
	}

	// Generate Repository Layer
	if contains(layers, "repository") {
		fmt.Println("\n📦 Generating Repository Layer...")
		repoConfig := repositorygen.Config{
			ModulePath:     config.ModulePath,
			OutputDir:      config.OutputDir,
			ForceOverwrite: config.ForceOverwrite,
		}

		repoResult, err := repositorygen.Generate(tableDef, repoConfig)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("generate repository: %w", err))
			result.RepositoryResult = repoResult
			return result, err
		}

		result.RepositoryResult = repoResult
		fmt.Printf("  ✅ Model:      %s\n", repoResult.ModelFile)
		fmt.Printf("  ✅ Repository: %s\n", repoResult.RepositoryFile)
	}

	// Generate Store Layer
	if contains(layers, "store") {
		fmt.Println("\n🗄️  Generating Store Layer...")
		storeConfig := pgxstores.GenerateConfig{
			ModulePath:     config.ModulePath,
			OutputDir:      config.OutputDir,
			ForceOverwrite: config.ForceOverwrite,
		}

		storeFile, err := pgxstores.GenerateFromSchema(tableDef, storeConfig)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("generate store: %w", err))
			return result, err
		}

		result.StoreResult = storeFile
		fmt.Printf("  ✅ Store: %s\n", storeFile)
	}

	// Generate Bridge Layer
	if contains(layers, "bridge") {
		fmt.Println("\n🌉 Generating Bridge Layer...")
		bridgeConfig := bridgegen.Config{
			ModulePath:     config.ModulePath,
			OutputDir:      config.OutputDir,
			ForceOverwrite: config.ForceOverwrite,
		}

		bridgeResult, err := bridgegen.Generate(tableDef, bridgeConfig)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("generate bridge: %w", err))
			result.BridgeResult = bridgeResult
			return result, err
		}

		result.BridgeResult = bridgeResult
		fmt.Printf("  ✅ Bridge:  %s\n", bridgeResult.BridgeFile)
		fmt.Printf("  ✅ Routes:  %s\n", bridgeResult.HTTPRoutesFile)
		fmt.Printf("  ✅ HTTP:    %s\n", bridgeResult.HTTPFile)
		fmt.Printf("  ✅ Model:   %s\n", bridgeResult.ModelFile)
		fmt.Printf("  ✅ Marshal: %s\n", bridgeResult.MarshalFile)
		fmt.Printf("  ✅ FOP:     %s\n", bridgeResult.FOPFile)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// PrintSummary prints a summary of the generation results
func PrintSummary(result *Result, tableName string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("🎉 GENERATION COMPLETE for table: %s\n", tableName)
	fmt.Println(strings.Repeat("=", 60))

	if result.RepositoryResult != nil {
		fmt.Println("\n📦 Repository Layer:")
		fmt.Printf("   • %s\n", result.RepositoryResult.ModelFile)
		fmt.Printf("   • %s\n", result.RepositoryResult.RepositoryFile)
	}

	if result.StoreResult != "" {
		fmt.Println("\n🗄️  Store Layer:")
		fmt.Printf("   • %s\n", result.StoreResult)
	}

	if result.BridgeResult != nil {
		fmt.Println("\n🌉 Bridge Layer:")
		fmt.Printf("   • %s\n", result.BridgeResult.BridgeFile)
		fmt.Printf("   • %s\n", result.BridgeResult.HTTPRoutesFile)
		fmt.Printf("   • %s\n", result.BridgeResult.HTTPFile)
		fmt.Printf("   • %s\n", result.BridgeResult.ModelFile)
		fmt.Printf("   • %s\n", result.BridgeResult.MarshalFile)
		fmt.Printf("   • %s\n", result.BridgeResult.FOPFile)
	}

	fmt.Printf("\n⏱️  Duration: %v\n", result.Duration)

	if len(result.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("   • %s\n", w)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for _, e := range result.Errors {
			fmt.Printf("   • %s\n", e)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🚀 Ready to use! Don't forget to:")
	fmt.Println("   1. Review the generated files")
	fmt.Println("   2. Add any custom business logic")
	fmt.Println("   3. Wire up the routes in your main.go")
	fmt.Println(strings.Repeat("=", 60))
}
