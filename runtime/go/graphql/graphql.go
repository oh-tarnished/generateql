// Package graphql provides small helpers for working with generated GraphQL clients,
// chiefly constructors that turn a value into a pointer for optional (nullable)
// arguments on generated Params structs, e.g. ListParams{Limit: graphql.Int(5)}.
package graphql

import "encoding/json"

// Ptr returns a pointer to v. It works for any type, including generated enums and
// input structs, e.g. Where: graphql.Ptr(types.UsersBoolExp{...}).
func Ptr[T any](v T) *T {
	return &v
}

// Typed scalar constructors return a pointer to the value, mirroring the Go types the
// generator maps GraphQL scalars to. Use them for optional scalar arguments.

// String returns a pointer to a string value (GraphQL String/String1/ID/Bigdecimal/
// Timestamp/Timestamptz, which all map to string).
func String(v string) *string { return &v }

// Bool returns a pointer to a bool value (GraphQL Boolean/Boolean1).
func Bool(v bool) *bool { return &v }

// Int returns a pointer to an int value (GraphQL Int).
func Int(v int) *int { return &v }

// Int32 returns a pointer to an int32 value (GraphQL Int32).
func Int32(v int32) *int32 { return &v }

// Int64 returns a pointer to an int64 value (GraphQL Int64).
func Int64(v int64) *int64 { return &v }

// Float64 returns a pointer to a float64 value (GraphQL Float/Float64).
func Float64(v float64) *float64 { return &v }

// JSON returns a pointer to a json.RawMessage value (GraphQL Json).
func JSON(v json.RawMessage) *json.RawMessage { return &v }
