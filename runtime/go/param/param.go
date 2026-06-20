// Package param provides Opt, a presence-tracking optional value for generated input
// (parameter) types, mirroring the design used by Stainless-generated SDKs.
//
// An Opt distinguishes "set" from "unset" without a pointer. Unset Opts are omitted
// from JSON via the encoding/json "omitzero" struct tag (Go 1.24+), so callers never
// take addresses or wrap values in pointers. A set Opt encodes its value even when that
// value is the type's zero (e.g. an explicit empty string or a zero count), which a
// plain "omitempty" field cannot express.
//
// Generated code sets Opt fields with value-taking constructors (e.g. graphql.String,
// graphql.Int) rather than &v, keeping pointer bookkeeping out of call sites.
package param

import (
	"encoding/json"
	"reflect"
)

// Opt is an optional value of a comparable type T. The zero Opt is "unset".
type Opt[T comparable] struct {
	value   T
	present bool
}

// NewOpt returns a set Opt holding v.
func NewOpt[T comparable](v T) Opt[T] { return Opt[T]{value: v, present: true} }

// Get returns the wrapped value and whether the Opt is set.
func (o Opt[T]) Get() (T, bool) { return o.value, o.present }

// Value returns the wrapped value (the zero value of T when unset).
func (o Opt[T]) Value() T { return o.value }

// IsPresent reports whether the Opt has been set.
func (o Opt[T]) IsPresent() bool { return o.present }

// Or returns the wrapped value when set, otherwise def.
func (o Opt[T]) Or(def T) T {
	if o.present {
		return o.value
	}
	return def
}

// IsZero reports whether the Opt is unset, so encoding/json's "omitzero" tag omits it.
func (o Opt[T]) IsZero() bool { return !o.present }

// MarshalJSON encodes the wrapped value. Unset Opts are omitted by "omitzero" before
// this runs, but a set Opt holding the zero value still encodes explicitly.
func (o Opt[T]) MarshalJSON() ([]byte, error) {
	if !o.present {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

// UnmarshalJSON decodes a value and marks the Opt set; a JSON null leaves it unset.
func (o *Opt[T]) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	if err := json.Unmarshal(b, &o.value); err != nil {
		return err
	}
	o.present = true
	return nil
}

// IsOmitted reports whether v is its type's zero value. Generated operation code uses it
// to decide whether to send an optional argument whose type is not wrapped in Opt
// (nested input structs and slices, which "omitzero" handles the same way).
func IsOmitted(v any) bool {
	rv := reflect.ValueOf(v)
	return !rv.IsValid() || rv.IsZero()
}
