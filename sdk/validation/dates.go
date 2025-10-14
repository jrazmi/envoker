package validation

import (
	"fmt"
	"time"
)

// parseFlexibleDate tries to parse a date string using multiple common formats
func ParseFlexibleDate(dateStr string) (time.Time, error) {
	formats := []string{
		"01/02/2006",  // MM/DD/YYYY (4-digit year)
		"01/02/06",    // MM/DD/YY (2-digit year)
		"2006-01-02",  // YYYY-MM-DD (ISO date)
		"01-02-2006",  // MM-DD-YYYY
		"01-02-06",    // MM-DD-YY (2-digit year)
		"2006/01/02",  // YYYY/MM/DD
		"06/01/02",    // YY/MM/DD (2-digit year)
		"02/01/2006",  // DD/MM/YYYY (European)
		"02/01/06",    // DD/MM/YY (European, 2-digit year)
		"02-01-2006",  // DD-MM-YYYY
		"02-01-06",    // DD-MM-YY (European, 2-digit year)
		time.RFC3339,  // ISO 8601 with time
		time.DateOnly, // Go 1.20+ constant for YYYY-MM-DD
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
