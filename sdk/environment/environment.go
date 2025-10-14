// Package environment provides utilities for managing environment variables
// and configuration loading with support for namespacing and defaults.
package environment

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from a .env file at the specified path.
// This is typically called at application startup to load local development
// environment variables.
//
// Example:
//
//	// Load from .env in current directory
//	if err := LoadEnv(".env"); err != nil {
//	    log.Printf("Warning: .env file not found: %v", err)
//	}
//
//	// Load from specific path
//	LoadEnv("/config/.env.production")
func LoadEnv() error {

	return godotenv.Load()
}

func LoadPath(p string) error {
	if p != "" {
		return godotenv.Load(p)
	}
	return godotenv.Load()
}

// GetEnvOrDefault retrieves an environment variable value, returning a fallback
// value if the variable is not set. This is useful for configuration values
// that have sensible defaults.
//
// Example:
//
//	port := GetEnvOrDefault("PORT", "8080")
//	dbHost := GetEnvOrDefault("DB_HOST", "localhost")
func GetEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// GetNamespaceEnvKey constructs a namespaced environment variable key by
// combining a namespace prefix with the actual key name using an underscore.
// If no namespace is provided, it returns the key unchanged.
//
// This is useful for applications that need to avoid environment variable
// naming conflicts or when running multiple instances of the same service.
//
// Example:
//
//	key := GetNamespaceEnvKey("MYAPP", "DATABASE_URL")
//	// Returns: "MYAPP_DATABASE_URL"
//
//	key := GetNamespaceEnvKey("", "DATABASE_URL")
//	// Returns: "DATABASE_URL"
func GetNamespaceEnvKey(namespace, key string) string {
	if namespace == "" {
		return key
	}
	return fmt.Sprintf("%s_%s", namespace, key)
}

// GetNamespaceEnvOrDefault retrieves a namespaced environment variable value,
// returning a fallback value if the variable is not set. It combines the
// functionality of GetNamespaceEnvKey and GetEnvOrDefault.
//
// Example:
//
//	// Looks for MYAPP_PORT, returns "8080" if not found
//	port := GetNamespaceEnvOrDefault("MYAPP", "PORT", "8080")
//
//	// Looks for SERVICE_TIMEOUT, returns "30s" if not found
//	timeout := GetNamespaceEnvOrDefault("SERVICE", "TIMEOUT", "30s")
//
//	// With empty namespace, looks for PORT directly
//	port := GetNamespaceEnvOrDefault("", "PORT", "8080")
func GetNamespaceEnvOrDefault(namespace, key, fallback string) string {
	if namespace == "" {
		return GetEnvOrDefault(key, fallback)
	}
	return GetEnvOrDefault(fmt.Sprintf("%s_%s", namespace, key), fallback)
}

// GetNamespaceEnvValue retrieves the value of a namespaced environment variable.
// Unlike GetNamespaceEnvOrDefault, this returns an empty string if the variable
// is not set (no fallback value).
//
// Note: This function cannot distinguish between an unset variable and a
// variable set to an empty string. Use os.LookupEnv directly if you need
// to make this distinction.
//
// Example:
//
//	// Looks for MYAPP_API_KEY
//	apiKey := GetNamespaceEnvValue("MYAPP", "API_KEY")
//
//	// Looks for API_KEY directly (no namespace)
//	apiKey := GetNamespaceEnvValue("", "API_KEY")
//
//	// Check if value exists
//	if apiKey := GetNamespaceEnvValue("MYAPP", "API_KEY"); apiKey != "" {
//	    // Use the API key
//	}
func GetNamespaceEnvValue(namespace, key string) string {
	if namespace == "" {
		return os.Getenv(key)
	}
	return os.Getenv(fmt.Sprintf("%s_%s", namespace, key))
}

// Common usage patterns:
//
// 1. Application with namespace prefix:
//
//	const AppNamespace = "MYAPP"
//
//	func GetConfig() Config {
//	    return Config{
//	        Port:     GetNamespaceEnvOrDefault(AppNamespace, "PORT", "8080"),
//	        DBHost:   GetNamespaceEnvOrDefault(AppNamespace, "DB_HOST", "localhost"),
//	        LogLevel: GetNamespaceEnvOrDefault(AppNamespace, "LOG_LEVEL", "info"),
//	    }
//	}
//
// 2. Service with optional namespace:
//
//	type Service struct {
//	    namespace string
//	}
//
//	func (s *Service) GetSetting(key, defaultVal string) string {
//	    return GetNamespaceEnvOrDefault(s.namespace, key, defaultVal)
//	}
//
// 3. Multi-tenant configuration:
//
//	func GetTenantConfig(tenantID string) TenantConfig {
//	    namespace := fmt.Sprintf("TENANT_%s", strings.ToUpper(tenantID))
//	    return TenantConfig{
//	        APIKey: GetNamespaceEnvValue(namespace, "API_KEY"),
//	        Limit:  GetNamespaceEnvOrDefault(namespace, "RATE_LIMIT", "100"),
//	    }
//	}
