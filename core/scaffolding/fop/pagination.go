package fop

import (
	"fmt"
	"strconv"
)

// PageLimit represents the requested items per page.
type PageStringCursor struct {
	Limit  int
	Cursor string
}

// Pagination returns pagination data. Every slice query should return pagination info.
type Pagination struct {
	HasPrev        bool   `json:"has_prev,omitempty"`
	Limit          int    `json:"limit,omitempty"`
	PreviousCursor string `json:"previous_cursor,omitempty"`
	NextCursor     string `json:"next_cursor,omitempty"`
	PageTotal      int    `json:"page_total,omitempty"`
}

func ParsePageStringCursor(pageLimit string, cursor string) (PageStringCursor, error) {
	limit := 25

	if pageLimit != "" {
		var err error
		limit, err = strconv.Atoi(pageLimit)
		if err != nil {
			return PageStringCursor{}, fmt.Errorf("page limit conversion: %w", err)
		}
	}

	if limit <= 0 {
		return PageStringCursor{}, fmt.Errorf("rows value too small, must be larger than 0")
	}

	if limit > 100 {
		return PageStringCursor{}, fmt.Errorf("rows value too large, must be less than 100")
	}

	return PageStringCursor{
		Limit:  limit,
		Cursor: cursor,
	}, nil
}
