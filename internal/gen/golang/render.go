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

	"github.com/oh-tarnished/generate-ql/internal/ir"
	"github.com/oh-tarnished/generate-ql/internal/naming"
	"github.com/oh-tarnished/generate-ql/internal/selection"
	"github.com/oh-tarnished/generate-ql/internal/typemap"
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

// paramsType renders the <Method>Params struct for an operation's nullable arguments,
// or "" when the operation has none.
func (r *renderer) paramsType(o op) string {
	opt := optionalArgs(o.Op)
	if len(opt) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "// %sParams holds the optional arguments for %s.\ntype %sParams struct {\n", o.Name, o.Name, o.Name)
	for _, a := range opt {
		fmt.Fprintf(&b, "\t%s %s\n", naming.Export(a.Name), r.mapper.GoArgType(a.Type, qHandler))
	}
	b.WriteString("}\n")
	return b.String()
}

// iface renders an interface declaration with one method signature per op.
func (r *renderer) iface(name, doc string, ops []op) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// %s\ntype %s interface {\n", doc, name)
	for _, o := range ops {
		fmt.Fprintf(&b, "\t// %s runs the %q %s.\n\t%s\n", o.Name, o.Op.Name, o.Op.Kind, r.signature(o))
	}
	b.WriteString("}\n")
	return b.String()
}

// handler renders the unexported handler struct plus a method per op.
func (r *renderer) handler(recv string, ops []op) string {
	var b strings.Builder
	fmt.Fprintf(&b, "type %s struct {\n\tgql *runtime.GraphQLClient\n}\n\n", recv)
	for _, o := range ops {
		b.WriteString(r.method(recv, o))
		b.WriteByte('\n')
	}
	return b.String()
}

// signature renders a method signature (no receiver): required args are positional,
// nullable args are collapsed into a trailing params struct.
func (r *renderer) signature(o op) string {
	parts := []string{"ctx context.Context"}
	for _, a := range requiredArgs(o.Op) {
		parts = append(parts, fmt.Sprintf("%s %s", paramName(a.Name), r.mapper.GoType(a.Type, qHandler)))
	}
	if len(optionalArgs(o.Op)) > 0 {
		parts = append(parts, "params "+o.Name+"Params")
	}
	join := strings.Join(parts, ", ")
	if o.Op.Kind == "subscription" {
		return fmt.Sprintf("%s(%s) (*runtime.Subscription, error)", o.Name, join)
	}
	return fmt.Sprintf("%s(%s) (%s, error)", o.Name, join, r.mapper.GoType(o.Op.Return, qHandler))
}

// method renders the concrete handler method implementing op. Optional (nullable)
// arguments are only added to the variables map when non-nil, so the runtime omits
// them from the operation entirely rather than sending an explicit null.
func (r *renderer) method(recv string, o op) string {
	retGo := r.mapper.GoType(o.Op.Return, qHandler)

	var b strings.Builder
	fmt.Fprintf(&b, "func (h *%s) %s {\n", recv, r.signature(o))
	fmt.Fprintf(&b, "\tvar out %s\n", retGo)
	b.WriteString(r.argsBlock(o.Op))

	if o.Op.Kind == "subscription" {
		fmt.Fprintf(&b, "\treturn h.gql.SubscribeFields(%q, &out, args)\n}\n", o.Op.Name)
		return b.String()
	}
	verb := "QueryFields"
	if o.Op.Kind == "mutation" {
		verb = "MutateFields"
	}
	fmt.Fprintf(&b, "\tres := <-h.gql.%s(%q, &out, args)\n", verb, o.Op.Name)
	fmt.Fprintf(&b, "\treturn out, res.Error\n}\n")
	return b.String()
}

// argsBlock renders the variables-map construction: required args are set
// unconditionally; optional args are added only when non-nil.
func (r *renderer) argsBlock(operation *ir.Operation) string {
	var b strings.Builder
	req := requiredArgs(operation)
	if len(req) == 0 {
		b.WriteString("\targs := map[string]any{}\n")
	} else {
		b.WriteString("\targs := map[string]any{\n")
		for _, a := range req {
			fmt.Fprintf(&b, "\t\t%q: %s,\n", a.Name, paramName(a.Name))
		}
		b.WriteString("\t}\n")
	}
	for _, a := range optionalArgs(operation) {
		field := naming.Export(a.Name)
		fmt.Fprintf(&b, "\tif params.%s != nil {\n\t\targs[%q] = params.%s\n\t}\n", field, a.Name, field)
	}
	return b.String()
}

// requiredArgs returns the non-null arguments (positional parameters).
func requiredArgs(operation *ir.Operation) []ir.Arg {
	var out []ir.Arg
	for _, a := range operation.Args {
		if a.Type.NonNull {
			out = append(out, a)
		}
	}
	return out
}

// optionalArgs returns the nullable arguments (collected into the params struct).
func optionalArgs(operation *ir.Operation) []ir.Arg {
	var out []ir.Arg
	for _, a := range operation.Args {
		if !a.Type.NonNull {
			out = append(out, a)
		}
	}
	return out
}

// paramName converts a GraphQL argument name to a lowerCamel Go parameter.
func paramName(name string) string {
	parts := strings.Split(strings.TrimLeft(name, "_"), "_")
	for i := 1; i < len(parts); i++ {
		parts[i] = naming.Export(parts[i])
	}
	out := strings.Join(parts, "")
	if out == "" {
		return "arg"
	}
	return out
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
