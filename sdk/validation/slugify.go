package validation

import (
	"regexp"
	"strings"
)

// slugify converts a string to a URL-safe slug
func Slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Remove accents and normalize unicode characters
	s = removeAccents(s)

	// Replace spaces and other separators with hyphens
	separators := regexp.MustCompile(`[\s\-_]+`)
	s = separators.ReplaceAllString(s, "-")

	// Remove all characters that aren't alphanumeric or hyphens
	nonAlphaNumeric := regexp.MustCompile(`[^a-z0-9\-]`)
	s = nonAlphaNumeric.ReplaceAllString(s, "")

	// Remove leading/trailing hyphens and multiple consecutive hyphens
	s = strings.Trim(s, "-")
	multipleHyphens := regexp.MustCompile(`-+`)
	s = multipleHyphens.ReplaceAllString(s, "-")

	return s
}

// removeAccents removes accents from characters
func removeAccents(s string) string {
	// Map of accented characters to their base equivalents
	accentMap := map[rune]rune{
		'à': 'a', 'á': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'å': 'a',
		'è': 'e', 'é': 'e', 'ê': 'e', 'ë': 'e',
		'ì': 'i', 'í': 'i', 'î': 'i', 'ï': 'i',
		'ò': 'o', 'ó': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
		'ù': 'u', 'ú': 'u', 'û': 'u', 'ü': 'u',
		'ý': 'y', 'ÿ': 'y',
		'ñ': 'n', 'ç': 'c',
		'ß': 's',
	}

	var result strings.Builder
	for _, r := range s {
		if replacement, exists := accentMap[r]; exists {
			result.WriteRune(replacement)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
