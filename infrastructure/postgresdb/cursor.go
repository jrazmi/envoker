// cursor.go - Generic cursor implementation

package postgresdb

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ApplyCursorPagination is a generic pagination function that works with any cursor type
func ApplyCursorPagination[K any, O any](
	buf *bytes.Buffer,
	data pgx.NamedArgs,
	orderField string,
	pkField string,
	orderValue *O,
	keyValue *K,
	direction string,
	forPrevious bool,
) error {
	if keyValue == nil || orderValue == nil {
		return nil
	}

	quotedOrder, err := QuoteIdentifier(orderField)
	if err != nil {
		return fmt.Errorf("invalid order field: %w", err)
	}
	quotedPK, err := QuoteIdentifier(pkField)
	if err != nil {
		return fmt.Errorf("invalid pk field: %w", err)
	}

	needsWhere := !strings.Contains(buf.String(), "WHERE")
	if needsWhere {
		buf.WriteString(" WHERE ")
	} else {
		buf.WriteString(" AND ")
	}

	operator := determineOperator(direction, forPrevious)

	// Tuple comparison
	// e.g. ("created_at", "id") > ("2025-08-01", 4)
	fmt.Fprintf(buf, "(%s, %s) %s (@cursor_order_value, @cursor_pk)", quotedOrder, quotedPK, operator)

	data["cursor_order_value"] = *orderValue
	data["cursor_pk"] = *keyValue

	return nil
}

func determineOperator(direction string, forPrevious bool) string {
	operator := ">"
	if forPrevious {
		operator = "<"
	}
	if direction == DESC && !forPrevious {
		operator = "<"
	} else if direction == DESC && forPrevious {
		operator = ">"
	}
	return operator
}
