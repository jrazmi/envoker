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

// PageInfoStringCursor returns pagination data. Every slice query should return page info.
type PageInfoStringCursor struct {
	HasPrev        bool   `json:"hasPrev,omitempty"`
	Limit          int    `json:"limit,omitempty"`
	PreviousCursor string `json:"previousCursor,omitempty"`
	NextCursor     string `json:"nextCursor,omitempty"`
	PageTotal      int    `json:"pageTotal,omitempty"`
}

// PageLimit represents the requested items per page.
type PageIntCursor struct {
	Limit  int
	Cursor int
}

// PageInfoInt64Offset returns pagination data. Every slice query should return page info.
type PageInfoIntCursor struct {
	HasPrev        bool `json:"hasPrev,omitempty"`
	Limit          int  `json:"limit,omitempty"`
	PreviousCursor int  `json:"previousCursor,omitempty"`
	NextCursor     int  `json:"nextCursor,omitempty"`
	PageTotal      int  `json:"pageTotal,omitempty"`
}

type PageInt64Cursor struct {
	Limit  int
	Cursor int64
}

func ParsePageStringCursor(pageLimit string, cursor string) (PageStringCursor, error) {
	limit := 20

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

func ParsePageIntCursor(pageLimit string, cursor string) (PageIntCursor, error) {
	limit := 20

	if pageLimit != "" {
		var err error
		limit, err = strconv.Atoi(pageLimit)
		if err != nil {
			return PageIntCursor{}, fmt.Errorf("page limit conversion: %w", err)
		}
	}
	if limit <= 0 {
		return PageIntCursor{}, fmt.Errorf("rows value too small, must be larger than 0")
	}

	if limit > 100 {
		return PageIntCursor{}, fmt.Errorf("rows value too large, must be less than 100")
	}
	offset := int(0)
	if cursor != "" {
		var err error
		offset, err = strconv.Atoi(cursor)
		if err != nil {
			return PageIntCursor{}, fmt.Errorf("cursor conversion: %w", err)
		}
	}

	return PageIntCursor{
		Limit:  limit,
		Cursor: offset,
	}, nil
}

func ParsePageInt64Cursor(pageLimit string, cursor string) (PageInt64Cursor, error) {
	limit := 20

	if pageLimit != "" {
		var err error
		limit, err = strconv.Atoi(pageLimit)
		if err != nil {
			return PageInt64Cursor{}, fmt.Errorf("page limit conversion: %w", err)
		}
	}
	if limit <= 0 {
		return PageInt64Cursor{}, fmt.Errorf("rows value too small, must be larger than 0")
	}

	if limit > 100 {
		return PageInt64Cursor{}, fmt.Errorf("rows value too large, must be less than 100")
	}
	offset := int64(0)
	if cursor != "" {
		var err error
		offset, err = strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return PageInt64Cursor{}, fmt.Errorf("cursor conversion: %w", err)
		}
	}

	return PageInt64Cursor{
		Limit:  limit,
		Cursor: offset,
	}, nil
}
