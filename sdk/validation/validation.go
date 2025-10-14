package validation

import (
	"encoding/json"
	"time"
)

func StringPtr(s string) *string {
	return &s
}

func StringPtrValue(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// stringPtrIfNotEmpty returns a pointer to string if not empty, otherwise nil
func StringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func TimePtr(t time.Time) *time.Time {
	return &t
}

func BoolPtr(b bool) *bool {
	return &b
}

func TimeValid(t any) bool {
	if _, ok := t.(time.Time); ok {
		return true
	}
	return false
}

func TsInt32(i any) bool {
	if _, ok := i.(int32); ok {
		return true
	}
	return false
}

func IntPtr(s int) *int {
	return &s
}

func GetTimeOrEmpty(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// Helper functions for nullable fields

// getStringOrEmpty returns the string value or an empty string if nil
func GetStringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// getStringOrDefault returns the string value or a default value if nil
func GetStringOrDefault(s *string, defaultValue string) string {
	if s == nil {
		return defaultValue
	}
	return *s
}

// getTimeOrNow returns the time value or current time if nil
func GetTimeOrNow(t *time.Time) time.Time {
	if t == nil {
		return time.Now().UTC()
	}
	return t.UTC() // Ensure UTC time
}

// getBoolOrFalse returns the bool value or false if nil
func GetBoolOrFalse(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// getInt32OrZero returns the int32 value or 0 if nil
func GetInt32OrZero(i *int) int32 {
	if i == nil {
		return 0
	}
	return int32(*i)
}

// getJSONOrEmpty returns the JSON as bytes or empty bytes if nil
func GetJSONOrEmpty(j *json.RawMessage) []byte {
	if j == nil {
		return []byte("{}")
	}
	return []byte(*j)
}

// GetJSONOrEmptyObject returns the JSON as bytes or an empty JSON object if nil
// This ensures the returned value is always a valid JSON object
func GetJSONOrEmptyObject(j *json.RawMessage) json.RawMessage {
	if j == nil {
		return json.RawMessage("{}")
	}

	// Check if the JSON is valid and not null
	if len(*j) == 0 || string(*j) == "null" {
		return json.RawMessage("{}")
	}

	return *j
}

// getIntOrZero returns the int value or 0 if nil
func GetIntOrZero(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// Helper functions
func FormatTimePtrToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
