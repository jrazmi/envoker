// Package environment provides utilities for parsing environment variables
// into struct fields using reflection and struct tags.
package environment

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ParseEnvTags populates a struct's fields from environment variables based on struct tags.
// It uses reflection to dynamically map environment variables to struct fields,
// supporting various field types and configuration options through tags.
//
// This function implements a careful precedence system:
//  1. Environment variables (if set) - highest priority
//  2. Existing field values (from TOML or other sources)
//  3. Default values from struct tags - lowest priority
//
// The function will only modify a field if:
//   - An environment variable is explicitly set for that field, OR
//   - The field is at its zero value AND a default is specified
//
// Supported struct tags:
//   - `env:"KEY"` - The environment variable name to look for
//   - `default:"value"` - Default value if env var is not set and field is zero
//   - `separator:","` - Separator for slice values (default is comma)
//   - `required:"true"` - Makes the environment variable mandatory
//
// Parameters:
//   - namespace: Prefix for environment variable names (e.g., "MYAPP_SERVER")
//   - cfg: Pointer to a struct to populate
//
// Example struct with tags:
//
//	type Config struct {
//	    Host     string        `env:"HOST" default:"localhost"`
//	    Port     int           `env:"PORT" default:"8080"`
//	    Debug    bool          `env:"DEBUG" default:"false"`
//	    Timeout  time.Duration `env:"TIMEOUT" default:"30s"`
//	    Origins  []string      `env:"CORS_ORIGINS" separator:","`
//	    APIKey   string        `env:"API_KEY" required:"true"`
//	}
//
// Example usage:
//
//	cfg := &Config{
//	    Port: 9000,  // This value from TOML/JSON
//	}
//
//	// With namespace "MYAPP", this looks for MYAPP_HOST, MYAPP_PORT, etc.
//	err := ParseEnvTags("MYAPP", cfg)
//
//	// Result:
//	// - If MYAPP_PORT is set in env, it overrides the 9000
//	// - If MYAPP_PORT is NOT set, keeps 9000 (doesn't use default)
//	// - If MYAPP_HOST is NOT set, uses "localhost" (field was zero)
func ParseEnvTags(namespace string, cfg any) error {
	// Validate input - must be a pointer to a struct
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return errors.New("cfg must be a pointer to a struct")
	}

	v = v.Elem()  // Dereference to get the actual struct
	t := v.Type() // Get the struct's type information

	// Iterate through all fields in the struct
	for i := range v.NumField() {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields (lowercase first letter)
		if !field.CanSet() {
			continue
		}

		// Extract tag values
		envKey := fieldType.Tag.Get("env")
		defaultValue := fieldType.Tag.Get("default")
		separator := fieldType.Tag.Get("separator")
		required := fieldType.Tag.Get("required") == "true"

		// Skip fields without env tag
		if envKey == "" {
			continue
		}

		// Build the full environment variable name with namespace
		ek := GetNamespaceEnvKey(namespace, envKey)

		// Check if the environment variable exists
		// Using LookupEnv to distinguish between unset and empty
		value, exists := os.LookupEnv(ek)

		if !exists {
			// Environment variable is not set

			// Check if this field is required
			if required {
				return fmt.Errorf("required environment variable %s is not set", ek)
			}

			// IMPORTANT: Preserve existing non-zero values
			// This allows TOML/JSON config to work without env var overrides
			// Only apply defaults to zero-valued fields
			if isZeroValue(field) && defaultValue != "" {
				value = defaultValue
			} else {
				// Keep existing value (from TOML, JSON, or previous configuration)
				continue
			}
		}

		// Parse and set the field value based on its type
		if err := setFieldValue(field, value, separator); err != nil {
			return fmt.Errorf("error setting field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// isZeroValue determines if a reflect.Value contains the zero value for its type.
// This is used to decide whether to apply default values from struct tags.
//
// Zero values by type:
//   - string: ""
//   - numeric: 0
//   - bool: false
//   - slice: nil or empty
//   - pointer: nil
//   - struct: all fields are zero
//
// Example:
//
//	var s string        // isZeroValue = true (empty string)
//	s = "hello"         // isZeroValue = false
//	var n int          // isZeroValue = true (0)
//	n = 0              // isZeroValue = true (still 0)
//	var slice []string // isZeroValue = true (nil)
//	slice = []string{} // isZeroValue = true (empty)
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int64:
		return v.Int() == 0
	case reflect.Bool:
		return !v.Bool() // false is zero value
	case reflect.Slice:
		return v.IsNil() || v.Len() == 0
	default:
		// For complex types (structs, maps, etc.), use reflection
		// to compare with a zero-initialized value of the same type
		zero := reflect.Zero(v.Type())
		return reflect.DeepEqual(v.Interface(), zero.Interface())
	}
}

// setFieldValue sets a struct field's value from a string representation.
// It handles type conversion for common Go types used in configuration.
//
// Supported types:
//   - string: Direct assignment
//   - int, int64: Parsed as base-10 integers
//   - time.Duration: Parsed using time.ParseDuration (e.g., "5s", "1h30m")
//   - bool: Parsed using strconv.ParseBool ("true", "false", "1", "0", etc.)
//   - []string: Split by separator and trimmed
//
// Parameters:
//   - field: The reflect.Value of the field to set
//   - value: String value to parse and set
//   - separator: For slices, the delimiter to split on (default ",")
//
// Examples:
//
//	// String field
//	setFieldValue(field, "hello", "")           // field = "hello"
//
//	// Duration field
//	setFieldValue(field, "30s", "")             // field = 30 * time.Second
//	setFieldValue(field, "1h30m", "")           // field = 90 * time.Minute
//
//	// Bool field
//	setFieldValue(field, "true", "")            // field = true
//	setFieldValue(field, "1", "")               // field = true
//	setFieldValue(field, "false", "")           // field = false
//
//	// Slice field
//	setFieldValue(field, "a,b,c", ",")          // field = []string{"a", "b", "c"}
//	setFieldValue(field, "x|y|z", "|")          // field = []string{"x", "y", "z"}
//	setFieldValue(field, "one, two, three", ",") // field = []string{"one", "two", "three"}
func setFieldValue(field reflect.Value, value, separator string) error {
	switch field.Kind() {
	case reflect.String:
		// Strings are straightforward - direct assignment
		field.SetString(value)

	case reflect.Int, reflect.Int64:
		// Special handling for time.Duration (which is int64 internally)
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			if value == "" {
				return nil // Keep zero value for empty string
			}
			// Parse duration string (e.g., "5s", "10m", "1h30m")
			duration, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("cannot parse duration: %w", err)
			}
			field.SetInt(int64(duration))
		} else {
			// Regular integer parsing
			if value == "" {
				return nil // Keep zero value for empty string
			}
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("cannot parse int: %w", err)
			}
			field.SetInt(intVal)
		}

	case reflect.Bool:
		if value == "" {
			return nil // Keep false for empty string
		}
		// ParseBool accepts: "true", "false", "1", "0", "t", "f", "T", "F"
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("cannot parse bool: %w", err)
		}
		field.SetBool(boolVal)

	case reflect.Slice:
		// Currently only supporting []string slices
		if field.Type().Elem().Kind() == reflect.String {
			if value == "" {
				return nil // Keep nil/empty slice for empty string
			}
			if separator == "" {
				separator = "," // Default to comma separation
			}
			// Split the string and trim whitespace from each element
			parts := strings.Split(value, separator)
			stringSlice := make([]string, len(parts))
			for i, part := range parts {
				stringSlice[i] = strings.TrimSpace(part)
			}
			field.Set(reflect.ValueOf(stringSlice))
		} else {
			// Could be extended to support []int, []bool, etc.
			return fmt.Errorf("unsupported slice type: %s", field.Type())
		}

	default:
		// Unsupported field type
		// Could be extended to support custom types with encoding.TextUnmarshaler
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// Common usage patterns and best practices:
//
// 1. Configuration struct with all tag options:
//
//	type ServerConfig struct {
//	    // Simple fields with defaults
//	    Host string `env:"HOST" default:"0.0.0.0"`
//	    Port int    `env:"PORT" default:"8080"`
//
//	    // Required field - will error if not set
//	    APIKey string `env:"API_KEY" required:"true"`
//
//	    // Duration with time parsing
//	    Timeout time.Duration `env:"TIMEOUT" default:"30s"`
//
//	    // Slice with custom separator
//	    AllowedIPs []string `env:"ALLOWED_IPS" separator:"|"`
//
//	    // Boolean flags
//	    Debug   bool `env:"DEBUG" default:"false"`
//	    Verbose bool `env:"VERBOSE"` // Defaults to false if not set
//	}
//
// 2. Loading with namespace:
//
//	cfg := &ServerConfig{}
//
//	// This will look for: MYAPP_HOST, MYAPP_PORT, MYAPP_API_KEY, etc.
//	if err := ParseEnvTags("MYAPP", cfg); err != nil {
//	    log.Fatal("Configuration error:", err)
//	}
//
// 3. Combining with TOML/JSON loading:
//
//	// First, load from config file
//	cfg := loadFromTOML("config.toml")
//
//	// Then, apply environment overrides
//	// This preserves TOML values unless explicitly overridden
//	ParseEnvTags("MYAPP", cfg)
//
// 4. Testing with different environments:
//
//	// Development
//	os.Setenv("MYAPP_DEBUG", "true")
//	os.Setenv("MYAPP_HOST", "localhost")
//
//	// Production
//	os.Setenv("MYAPP_DEBUG", "false")
//	os.Setenv("MYAPP_HOST", "0.0.0.0")
//	os.Setenv("MYAPP_API_KEY", "secret-key-here")
