package validation

import (
	"strings"
	"unicode"
)

// camelCaseToTitleCase converts a camelCase string to Title Case with spaces
// Example: "caseStudy" -> "Case Study"
// Example: "XMLParser" -> "XML Parser"
// Example: "iOSApp" -> "iOS App"
func CamelCaseToTitleCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	runes := []rune(s)

	for i, r := range runes {
		// First character should be uppercase
		if i == 0 {
			result.WriteRune(unicode.ToUpper(r))
			continue
		}

		// Check if current rune is uppercase
		if unicode.IsUpper(r) {
			// Check if we need to add a space
			// Add space if:
			// 1. Previous character is lowercase (camelCase boundary)
			// 2. Previous character is uppercase but next is lowercase (acronym boundary like "XMLParser")
			prevIsLower := i > 0 && unicode.IsLower(runes[i-1])
			prevIsUpper := i > 0 && unicode.IsUpper(runes[i-1])
			nextIsLower := i < len(runes)-1 && unicode.IsLower(runes[i+1])

			if prevIsLower || (prevIsUpper && nextIsLower) {
				result.WriteRune(' ')
			}
			result.WriteRune(r)
		} else {
			// Lowercase character - just add it
			result.WriteRune(r)
		}
	}

	return result.String()
}
