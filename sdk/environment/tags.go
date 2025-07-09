// Package environment provides support for env vars.

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

// parseEnvTags fills a struct from environment variables using struct tags
func ParseEnvTags(prefix string, cfg any) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("cfg must be a pointer to a struct")
	}

	v = v.Elem() // dereference the pointer
	t := v.Type()

	for i := range v.NumField() {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanSet() {
			continue // skip unexported fields
		}

		envKey := fieldType.Tag.Get("env")
		defaultValue := fieldType.Tag.Get("default")
		separator := fieldType.Tag.Get("separator")
		required := fieldType.Tag.Get("required") == "true"

		if envKey == "" {
			continue // no env tag, skip this field
		}

		ek := GetEnvKeyPrefix(prefix, envKey)

		// Get value from environment or use default
		value := os.Getenv(ek)
		if value == "" {
			if required {
				return fmt.Errorf("required environment variable %s is not set", ek)
			}
			value = defaultValue
		}

		// Set the field value based on its type
		if err := setFieldValue(field, value, separator); err != nil {
			return fmt.Errorf("error setting field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// setFieldValue sets a reflect.Value based on its type
func setFieldValue(field reflect.Value, value, separator string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int64:
		// Check if this is a time.Duration (which is an int64 under the hood)
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			if value == "" {
				return nil // leave as zero value
			}
			duration, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("cannot parse duration: %w", err)
			}
			field.SetInt(int64(duration))
		} else {
			if value == "" {
				return nil // leave as zero value
			}
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("cannot parse int: %w", err)
			}
			field.SetInt(intVal)
		}

	case reflect.Bool:
		if value == "" {
			return nil // leave as zero value
		}
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("cannot parse bool: %w", err)
		}
		field.SetBool(boolVal)

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			if value == "" {
				return nil // leave as nil slice
			}
			if separator == "" {
				separator = "," // default separator
			}
			parts := strings.Split(value, separator)
			stringSlice := make([]string, len(parts))
			for i, part := range parts {
				stringSlice[i] = strings.TrimSpace(part)
			}
			field.Set(reflect.ValueOf(stringSlice))
		} else {
			return fmt.Errorf("unsupported slice type: %s", field.Type())
		}

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}
