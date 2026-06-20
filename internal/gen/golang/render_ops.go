package golang

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/generateql/internal/ir"
	"github.com/oh-tarnished/generateql/internal/naming"
)

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
		fmt.Fprintf(&b, "\t%s %s\n", naming.Export(a.Name), r.mapper.GoParamType(a.Type, qHandler))
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

// argsBlock renders the variables-map construction. Each value is wrapped with
// graphql.Var/VarPtr so go-graphql-client declares the exact GraphQL type (engine
// scalars like String1!/Int64 are not inferable from the Go kind). Required args are
// set unconditionally; optional args are added only when non-nil.
func (r *renderer) argsBlock(operation *ir.Operation) string {
	var b strings.Builder
	req := requiredArgs(operation)
	if len(req) == 0 {
		b.WriteString("\targs := map[string]any{}\n")
	} else {
		b.WriteString("\targs := map[string]any{\n")
		for _, a := range req {
			fmt.Fprintf(&b, "\t\t%q: graphql.Var(%s, %q),\n", a.Name, paramName(a.Name), gqlType(a.Type))
		}
		b.WriteString("\t}\n")
	}
	for _, a := range optionalArgs(operation) {
		field := naming.Export(a.Name)
		if r.mapper.ParamIsOpt(a.Type) {
			// param.Opt[T]: send the unwrapped value only when the caller set it.
			fmt.Fprintf(&b, "\tif params.%s.IsPresent() {\n\t\targs[%q] = graphql.VarPtr(params.%s.Value(), %q)\n\t}\n", field, a.Name, field, gqlType(a.Type))
			continue
		}
		// Value type (nested input or slice): omit it while it is still the zero value.
		fmt.Fprintf(&b, "\tif !param.IsOmitted(params.%s) {\n\t\targs[%q] = graphql.VarPtr(params.%s, %q)\n\t}\n", field, a.Name, field, gqlType(a.Type))
	}
	return b.String()
}

// gqlType renders the GraphQL type string for an argument WITHOUT the outer non-null
// "!": go-graphql-client appends "!" for non-pointer (Var) values, while nullable
// (VarPtr) values keep it off. List and element-nullability are included
// (e.g. "String1", "[UsersOrderByExp!]").
func gqlType(ft ir.FieldType) string {
	if ft.List {
		elem := ft.Base
		if ft.ElemNonNull {
			elem += "!"
		}
		return "[" + elem + "]"
	}
	return ft.Base
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
