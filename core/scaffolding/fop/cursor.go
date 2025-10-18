package fop

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

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

type Cursor[PK any, OrderValue any] struct {
	OrderValue OrderValue `json:"order_value"`
	PK         PK         `json:"pk"`
}

func (c Cursor[PK, OrderValue]) Encode() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

func DecodeCursor[PK any, OrderValue any](token string) (*Cursor[PK, OrderValue], error) {
	if token == "" {
		return nil, nil
	}

	data, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor Cursor[PK, OrderValue]
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}

	return &cursor, nil
}
