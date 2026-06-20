// Package golang renders a Go client from the IR using an interface/handler
// architecture grouped by domain. Each resource is its own package exposing
// Query/Mutation/Subscription interfaces backed by unexported handlers; a shared
// types package holds models, inputs, and enums; domain and root packages aggregate
// the handlers so callers write s.Query.<Domain>.<Resource>.<Method>(...).
//
// Required (non-null) arguments are positional; nullable arguments are bundled into a
// generated <Method>Params struct so callers never pass positional nils.
package golang

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/generateql/internal/ir"
	"github.com/oh-tarnished/generateql/internal/naming"
	"github.com/oh-tarnished/generateql/internal/selection"
	"github.com/oh-tarnished/generateql/internal/typemap"
)

// Qualifiers for referencing generated types from each writing context. Models inline
// their relations (so they only reference enums); inputs reference enums and other
// inputs (same package); handler code references all three type packages.
var (
	qModels  = typemap.Qualifier{Enums: "enums."}
	qInputs  = typemap.Qualifier{Enums: "enums."}
	qHandler = typemap.Qualifier{Models: "schema.", Inputs: "inputs.", Enums: "enums."}
)

// op pairs an operation with its de-duplicated exported method name.
type op struct {
	Op   *ir.Operation
	Name string
}

// renderer turns IR elements into Go source fragments.
type renderer struct {
	schema    *ir.Schema
	mapper    *typemap.Mapper
	selection *selection.Renderer
}

// enum renders a Go enum: a named string type plus a typed constant per value.
func (r *renderer) enum(e *ir.Enum) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// %s is the %s enum.\ntype %s string\n\n", e.Name, e.Name, e.Name)
	if len(e.Values) > 0 {
		b.WriteString("const (\n")
		for _, v := range e.Values {
			fmt.Fprintf(&b, "\t%s%s %s = %q\n", e.Name, naming.Export(v), e.Name, v)
		}
		b.WriteString(")\n")
	}
	return b.String()
}

// input renders an input-object struct plus a pointer-receiver GetGraphQLType.
func (r *renderer) input(in *ir.Input) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// %s is the %s input type.\ntype %s struct {\n", in.Name, in.Name, in.Name)
	for _, f := range in.Fields {
		fmt.Fprintf(&b, "\t%s %s `json:%q`\n", naming.Export(f.Name), r.mapper.GoType(f.Type, qInputs), f.Name+",omitempty")
	}
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "func (*%s) GetGraphQLType() string { return %q }\n", in.Name, in.Name)
	return b.String()
}

// model renders an object's model struct, with relations inlined to max depth.
func (r *renderer) model(o *ir.Object) string {
	body := r.selection.ModelBody(o)
	var b strings.Builder
	fmt.Fprintf(&b, "// %s is the %s model.\ntype %s struct {\n%s}\n", o.Name, o.Name, o.Name, body)
	return b.String()
}

// sortedKeys returns the keys of m in deterministic order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
