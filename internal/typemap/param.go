package typemap

import "github.com/oh-tarnished/generateql/internal/ir"

// GoParamType returns the Go type for an INPUT field or optional operation argument,
// using the Stainless-style param design instead of pointers:
//
//   - required (non-null) leaf/input  -> plain value (string, graphql.Int64, inputs.X)
//   - nullable comparable scalar/enum -> param.Opt[T]   (presence without a pointer)
//   - nullable nested input or Json   -> plain value    (absence handled by "omitzero")
//   - list                            -> []elem         (nil slice omitted by "omitzero")
//
// Pairing every nullable field with the json:",omitzero" tag lets unset values disappear
// from the wire while a deliberately-set zero value (e.g. _eq: "") is still sent.
func (m *Mapper) GoParamType(ft ir.FieldType, q Qualifier) string {
	if ft.List {
		return "[]" + m.elemType(ft, q)
	}
	if ft.NonNull {
		return m.qualifiedLeaf(ft.Base, q)
	}
	if m.ParamIsOpt(ft) {
		return "param.Opt[" + m.qualifiedLeaf(ft.Base, q) + "]"
	}
	return m.qualifiedLeaf(ft.Base, q)
}

// ParamIsOpt reports whether a nullable, non-list field maps to a param.Opt[T] (a
// comparable scalar or enum) rather than a value type wrapped by "omitzero" (a nested
// input struct or the non-comparable Json scalar). Generated code uses this to choose
// between Opt.IsPresent() and param.IsOmitted(...) when deciding to send an argument.
func (m *Mapper) ParamIsOpt(ft ir.FieldType) bool {
	if ft.List || ft.NonNull {
		return false
	}
	if m.isInput(ft.Base) {
		return false
	}
	return m.leafGoType(ft.Base) != "json.RawMessage"
}
