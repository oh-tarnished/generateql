package golang

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/generateql/internal/ir"
)

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

// signature renders a method signature (no receiver): required args are positional;
// optional args collapse into a trailing variadic request builder.
func (r *renderer) signature(o op) string {
	specs := r.classify(o)
	parts := []string{"ctx context.Context"}
	// Scalar keys (e.g. id) come before the object/patch body for natural call order.
	for _, s := range specs {
		if s.role == roleScalar {
			parts = append(parts, s.goName+" "+s.goType)
		}
	}
	for _, s := range specs {
		if s.role == roleCreate || s.role == roleUpdate {
			parts = append(parts, s.goName+" "+s.goType)
		}
	}
	if len(optionalSpecs(specs)) > 0 {
		parts = append(parts, "req ...*"+o.Name+"Request")
	}
	join := strings.Join(parts, ", ")
	if o.Op.Kind == "subscription" {
		return fmt.Sprintf("%s(%s) (*runtime.Subscription, error)", o.Name, join)
	}
	return fmt.Sprintf("%s(%s) (%s, error)", o.Name, join, r.mapper.GoType(o.Op.Return, qResource))
}

// method renders the concrete handler method implementing op.
func (r *renderer) method(recv string, o op) string {
	retGo := r.mapper.GoType(o.Op.Return, qResource)
	var b strings.Builder
	fmt.Fprintf(&b, "func (h *%s) %s {\n", recv, r.signature(o))
	fmt.Fprintf(&b, "\tvar out %s\n", retGo)
	b.WriteString(r.argsBlock(o))
	switch o.Op.Kind {
	case "subscription":
		fmt.Fprintf(&b, "\treturn h.gql.SubscribeFields(%q, &out, args)\n}\n", o.Op.Name)
		return b.String()
	case "mutation":
		fmt.Fprintf(&b, "\tres := <-h.gql.MutateFields(%q, &out, args)\n", o.Op.Name)
	default:
		fmt.Fprintf(&b, "\tres := <-h.gql.QueryFields(%q, &out, args)\n", o.Op.Name)
	}
	b.WriteString("\treturn out, res.Error\n}\n")
	return b.String()
}

// argsBlock renders the variables-map construction. Required args are set unconditionally
// (objects wrapped into a one-element slice, an update patch flattened to set-columns);
// optional args are added only when non-zero, so the runtime omits them entirely.
func (r *renderer) argsBlock(o op) string {
	specs := r.classify(o)
	var b strings.Builder
	if len(optionalSpecs(specs)) > 0 {
		fmt.Fprintf(&b, "\tvar r %sRequest\n\tif len(req) > 0 && req[0] != nil {\n\t\tr = *req[0]\n\t}\n", o.Name)
	}
	b.WriteString("\targs := map[string]any{}\n")
	for _, s := range specs {
		if s.parent != "" {
			continue
		}
		switch s.role {
		case roleScalar:
			fmt.Fprintf(&b, "\targs[%q] = graphql.Var(%s, %q)\n", s.argName, s.goName, s.gqlType)
		case roleCreate:
			fmt.Fprintf(&b, "\targs[%q] = graphql.Var([]CreateInput{%s}, %q)\n", s.argName, s.goName, s.gqlType)
		case roleUpdate:
			fmt.Fprintf(&b, "\targs[%q] = graphql.Var(graphql.SetColumns(%s), %q)\n", s.argName, s.goName, s.gqlType)
		default: // rolePredicate, roleOrder, roleInt
			fmt.Fprintf(&b, "\tif !graphql.IsOmitted(r.%s) {\n\t\targs[%q] = graphql.VarPtr(r.%s, %q)\n\t}\n", s.goName, s.argName, s.goName, s.gqlType)
		}
	}
	b.WriteString(r.nestedArgs(specs))
	return b.String()
}

// nestedArgs renders the construction of wrapper variables (e.g. an aggregate
// filter_input) from their flattened child specs, including each child only when set.
func (r *renderer) nestedArgs(specs []argSpec) string {
	var b strings.Builder
	seen := map[string]bool{}
	for _, p := range specs {
		if p.parent == "" || seen[p.parent] {
			continue
		}
		seen[p.parent] = true
		local := paramName(p.parent)
		fmt.Fprintf(&b, "\t%s := map[string]any{}\n", local)
		for _, c := range specs {
			if c.parent != p.parent {
				continue
			}
			fmt.Fprintf(&b, "\tif !graphql.IsOmitted(r.%s) {\n\t\t%s[%q] = r.%s\n\t}\n", c.goName, local, c.argName, c.goName)
		}
		fmt.Fprintf(&b, "\tif len(%s) > 0 {\n\t\targs[%q] = graphql.VarPtr(%s, %q)\n\t}\n", local, p.parent, local, p.parentType)
	}
	return b.String()
}

// gqlType renders an argument's GraphQL type WITHOUT the outer non-null "!" (which
// go-graphql-client appends for non-pointer Var values and omits for VarPtr).
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

// paramName converts a GraphQL argument name to a lowerCamel Go parameter.
func paramName(name string) string {
	parts := strings.Split(strings.TrimLeft(name, "_"), "_")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	out := strings.Join(parts, "")
	if out == "" {
		return "arg"
	}
	if goKeywords[out] {
		out += "_"
	}
	return out
}
