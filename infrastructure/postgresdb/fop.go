package postgresdb

import (
	"bytes"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Set of directions for data ordering.
const (
	ASC  = "ASC"
	DESC = "DESC"
)

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
