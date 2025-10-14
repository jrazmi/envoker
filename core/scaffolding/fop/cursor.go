package fop

// StringCursorConfig holds configuration for string-based cursor pagination
type StringCursorConfig struct {
	// The cursor value (string)
	Cursor string

	// Field to use for cursor ordering
	OrderField string

	// Primary key field for tie-breaking
	PKField string

	// Table name for subqueries
	TableName string

	// Order direction (ASC or DESC)
	Direction string

	// Maximum number of records to return
	Limit int
}
