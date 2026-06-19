// Package graphql provides small helpers and scalar types for generated GraphQL
// clients: pointer constructors for optional (nullable) arguments, and scalar types
// that tolerate engine-specific JSON encodings.
package graphql

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Ptr returns a pointer to v. It works for any type, including generated enums, input
// structs, and the scalar types below (e.g. Where: graphql.Ptr(types.UsersBoolExp{})).
func Ptr[T any](v T) *T {
	return &v
}

// Int64 is a 64-bit integer GraphQL scalar. Engines commonly serialize 64-bit integers
// as JSON strings to preserve precision, but may return computed values (e.g. aggregate
// counts) as JSON numbers. Int64 decodes from either form and encodes as a string.
type Int64 int64

// UnmarshalJSON accepts a JSON number or a quoted JSON string.
func (v *Int64) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("Int64: %w", err)
	}
	*v = Int64(n)
	return nil
}

// MarshalJSON encodes as a quoted string so full 64-bit precision survives the wire.
func (v Int64) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(strconv.FormatInt(int64(v), 10))), nil
}

// Typed scalar pointer constructors for optional arguments.

// String returns a pointer to a string value (GraphQL String/String1/ID/Bigdecimal/
// Timestamp/Timestamptz, which all map to string).
func String(v string) *string { return &v }

// Bool returns a pointer to a bool value (GraphQL Boolean/Boolean1).
func Bool(v bool) *bool { return &v }

// Int returns a pointer to an int value (GraphQL Int).
func Int(v int) *int { return &v }

// Int32 returns a pointer to an int32 value (GraphQL Int32).
func Int32(v int32) *int32 { return &v }

// Float64 returns a pointer to a float64 value (GraphQL Float/Float64).
func Float64(v float64) *float64 { return &v }

// JSON returns a pointer to a json.RawMessage value (GraphQL Json).
func JSON(v json.RawMessage) *json.RawMessage { return &v }
