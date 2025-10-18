package postgresdb

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// StringCursorConfig holds configuration for string-based cursor pagination
type StringCursorConfig struct {
	Cursor     string
	OrderField string
	PKField    string
	TableName  string
	Direction  string
	Limit      int
}

// Set of directions for data ordering.
const (
	ASC  = "ASC"
	DESC = "DESC"
)

// decodeBase64JSON decodes a base64-encoded JSON string into the target type
func decodeBase64JSON[T any](encoded string) (*T, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	return &result, nil
}

// ApplyStringCursorPagination applies string cursor pagination to a query
// This is a generic function that decodes the cursor and extracts the pk and order_value
func ApplyStringCursorPagination[OrderValue any](
	buf *bytes.Buffer,
	data pgx.NamedArgs,
	config StringCursorConfig,
	forPrevious bool,
) error {
	// If no cursor, nothing to apply
	if config.Cursor == "" {
		return nil
	}

	// Import the fop package for cursor decoding
	// The cursor is a base64-encoded JSON containing {pk: string, order_value: OrderValue}
	// We need to decode it to extract these values

	// Decode the cursor - we need to use a local struct since we can't import fop here
	// to avoid circular dependencies
	type cursorData struct {
		OrderValue OrderValue `json:"order_value"`
		PK         string     `json:"pk"`
	}

	// Decode from base64
	decoded, err := decodeBase64JSON[cursorData](config.Cursor)
	if err != nil {
		return fmt.Errorf("decode cursor: %w", err)
	}

	// Use the generic ApplyCursorPagination function that already exists
	return ApplyCursorPagination(
		buf, data,
		config.OrderField, config.PKField,
		&decoded.OrderValue, &decoded.PK,
		config.Direction, forPrevious,
	)
}

// AddOrderByClause adds ORDER BY clause to the query buffer
func AddOrderByClause(buf *bytes.Buffer, orderField, pkField, direction string, forPrevious bool) error {
	// Validate and quote identifiers
	quotedOrderField, err := QuoteIdentifier(orderField)
	if err != nil {
		return fmt.Errorf("invalid order field name: %w", err)
	}
	quotedPKField, err := QuoteIdentifier(pkField)
	if err != nil {
		return fmt.Errorf("invalid pk field name: %w", err)
	}

	actualDirection := direction

	// Reverse direction for previous page to get results in reverse order
	if forPrevious {
		if direction == ASC {
			actualDirection = DESC
		} else {
			actualDirection = ASC
		}
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s %s", quotedOrderField, actualDirection))

	// Add primary key as secondary sort for consistency (if not already the order field)
	if orderField != pkField {
		buf.WriteString(fmt.Sprintf(", %s %s", quotedPKField, actualDirection))
	}

	return nil
}

// AddLimitClause adds LIMIT clause to the query buffer
func AddLimitClause(limit int, data pgx.NamedArgs, buf *bytes.Buffer) {
	buf.WriteString(" LIMIT @limit")
	data["limit"] = limit
}

// AliasedOrderField creates an aliased field name for queries with table aliases
func AliasedOrderField(field string, alias string) string {
	return fmt.Sprintf("%s.%s", alias, field)
}

// IntCursorConfig holds configuration for integer-based cursor pagination
type IntCursorConfig struct {
	Cursor     int
	OrderField string
	PKField    string
	TableName  string
	Direction  string
	Limit      int
}

// ApplyIntCursorPagination adds cursor-based WHERE conditions for integer cursors
func ApplyIntCursorPagination(buf *bytes.Buffer, data pgx.NamedArgs, config IntCursorConfig, forPrevious bool) error {
	if config.Cursor == 0 {
		return nil
	}

	// Validate and quote identifiers
	order, err := QuoteIdentifier(config.OrderField)
	if err != nil {
		return fmt.Errorf("invalid order field name: %w", err)
	}
	tableName, err := QuoteIdentifier(config.TableName)
	if err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	pkField, err := QuoteIdentifier(config.PKField)
	if err != nil {
		return fmt.Errorf("invalid pk field name: %w", err)
	}

	// Add cursor to query parameters
	data["cursor"] = config.Cursor

	// Determine if we need to add WHERE or AND
	needsWhere := !strings.Contains(buf.String(), "WHERE")

	if needsWhere {
		buf.WriteString(" WHERE ")
	} else {
		buf.WriteString(" AND ")
	}

	// For previous page, invert the comparison
	operator := ">"
	if forPrevious {
		operator = "<"
	}

	// Adjust the operator based on sort direction
	if config.Direction == DESC && !forPrevious {
		operator = "<"
	} else if config.Direction == DESC && forPrevious {
		operator = ">"
	}

	// Only allow specific operators (whitelist approach)
	validOperators := map[string]bool{"<": true, ">": true, "<=": true, ">=": true}
	if !validOperators[operator] {
		return fmt.Errorf("invalid operator: %s", operator)
	}

	// Build the cursor comparison condition using buffer operations
	buf.WriteString("(")

	// First condition: OrderField comparison with subquery
	buf.WriteString(order)
	buf.WriteString(" ")
	buf.WriteString(operator)
	buf.WriteString(" (SELECT ")
	buf.WriteString(order)
	buf.WriteString(" FROM ")
	buf.WriteString(tableName)
	buf.WriteString(" WHERE ")
	buf.WriteString(pkField)
	buf.WriteString(" = @cursor)")

	buf.WriteString(" OR (")

	// Second condition: OrderField equality with subquery AND PKField comparison
	buf.WriteString(order)
	buf.WriteString(" = (SELECT ")
	buf.WriteString(order)
	buf.WriteString(" FROM ")
	buf.WriteString(tableName)
	buf.WriteString(" WHERE ")
	buf.WriteString(pkField)
	buf.WriteString(" = @cursor) AND ")
	buf.WriteString(pkField)
	buf.WriteString(" ")
	buf.WriteString(operator)
	buf.WriteString(" @cursor")

	buf.WriteString("))")

	return nil
}
