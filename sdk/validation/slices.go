package validation

// SliceOverlap returns only the items that are in the allowed list
func StringSliceOverlap(requested []string, allowed []string) []string {
	// Create a map for O(1) lookup of allowed categories
	allowedMap := make(map[string]bool)
	for _, cat := range allowed {
		allowedMap[cat] = true
	}

	// Filter requested categories
	var valid []string
	for _, cat := range requested {
		if allowedMap[cat] {
			valid = append(valid, cat)
		}
	}

	return valid
}
