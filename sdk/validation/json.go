package validation

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Generic JSON type that implements Scanner/Valuer
type JSONField[T any] struct {
	Data T
}

func (j *JSONField[T]) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &j.Data)
	case string:
		return json.Unmarshal([]byte(v), &j.Data)
	default:
		return fmt.Errorf("cannot scan %T into JSONField", value)
	}
}

func (j JSONField[T]) Value() (driver.Value, error) {
	return json.Marshal(j.Data)
}
