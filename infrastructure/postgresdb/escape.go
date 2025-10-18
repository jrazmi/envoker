package postgresdb

import (
	"fmt"
	"regexp"
	"strings"
)

// QuoteIdentifier validates and quotes SQL identifiers
// Returns the properly quoted identifier or an error if invalid
func QuoteIdentifier(name string) (string, error) {
	// First, check against obviously dangerous characters
	dangerousChars := regexp.MustCompile(`[;'"\\()]`)
	if dangerousChars.MatchString(name) {
		return "", fmt.Errorf("identifier contains dangerous characters: %s", name)
	}

	// Split the name to handle potential aliases
	parts := strings.Split(name, " ")

	// If there are more than 2 parts (column_name alias), that's invalid
	if len(parts) > 2 {
		return "", fmt.Errorf("invalid identifier format (too many parts): %s", name)
	}

	// Check the column/table name part (possibly including schema)
	identifierPattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)?$`)

	// For simple identifier without a space (no alias)
	if len(parts) == 1 {
		// Allow for schema.table format
		if strings.Contains(parts[0], ".") {
			segments := strings.Split(parts[0], ".")
			if len(segments) > 2 { // More than schema.table is invalid
				return "", fmt.Errorf("invalid identifier format (too many segments): %s", name)
			}
			// Each part must be a valid identifier
			for i, segment := range segments {
				if !identifierPattern.MatchString(segment) {
					return "", fmt.Errorf("invalid identifier segment at position %d: %s", i, segment)
				}
			}

			// Quote each segment
			quotedSegments := make([]string, len(segments))
			for i, segment := range segments {
				quotedSegments[i] = fmt.Sprintf(`"%s"`, segment)
			}
			return strings.Join(quotedSegments, "."), nil
		}

		// Simple column name
		if identifierPattern.MatchString(parts[0]) {
			return fmt.Sprintf(`"%s"`, parts[0]), nil
		}
		return "", fmt.Errorf("invalid identifier format: %s", parts[0])
	}

	// If we have two parts (an alias): both must be valid identifiers
	quoted := ""

	// First part can be schema.table format
	if strings.Contains(parts[0], ".") {
		segments := strings.Split(parts[0], ".")
		if len(segments) > 2 {
			return "", fmt.Errorf("invalid identifier format (too many segments in first part): %s", parts[0])
		}

		for i, segment := range segments {
			if !identifierPattern.MatchString(segment) {
				return "", fmt.Errorf("invalid identifier segment at position %d: %s", i, segment)
			}
		}

		// Quote each segment of the first part
		quotedSegments := make([]string, len(segments))
		for i, segment := range segments {
			quotedSegments[i] = fmt.Sprintf(`"%s"`, segment)
		}
		quoted = strings.Join(quotedSegments, ".")
	} else {
		if !identifierPattern.MatchString(parts[0]) {
			return "", fmt.Errorf("invalid identifier first part: %s", parts[0])
		}
		quoted = fmt.Sprintf(`"%s"`, parts[0])
	}

	// Alias must be a simple identifier (no schema)
	if !identifierPattern.MatchString(parts[1]) {
		return "", fmt.Errorf("invalid identifier alias: %s", parts[1])
	}

	// Add the quoted alias
	quoted += fmt.Sprintf(` "%s"`, parts[1])

	return quoted, nil
}
