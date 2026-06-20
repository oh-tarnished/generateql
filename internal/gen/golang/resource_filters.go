package golang

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/generateql/internal/ir"
	"github.com/oh-tarnished/generateql/internal/naming"
)

// renderPredicates renders predicates.go: a package-level field handle per filterable
// scalar column plus And/Or/Not. usesEnums reports whether the enums import is needed.
func (r *renderer) renderPredicates(rg *resGen) (body string, usesEnums bool) {
	base := r.boolExpType(rg)
	if base == "" {
		return "", false
	}
	type fv struct{ name, expr string }
	var vars []fv
	var relations []ir.Field
	for _, f := range sortedFields(r.schema.Inputs[base].Fields) {
		if f.Name == "_and" || f.Name == "_or" || f.Name == "_not" {
			continue
		}
		// Scalar comparisons have an _eq field; relations are row BoolExps without one.
		if r.hasField(f.Type.Base, "_eq") {
			expr, enum := r.fieldHandle(f)
			if enum {
				usesEnums = true
			}
			vars = append(vars, fv{naming.Export(f.Name), expr})
			continue
		}
		if r.isBoolExp(f.Type.Base) {
			relations = append(relations, f)
		}
	}
	if len(vars) == 0 && len(relations) == 0 {
		return "", false
	}
	var b strings.Builder
	if len(vars) > 0 {
		fmt.Fprintf(&b, "// Filter fields for %s. Build predicates like %s.Eq(v) and combine\n// them with And/Or/Not.\nvar (\n", rg.res.Name, vars[0].name)
		for _, v := range vars {
			fmt.Fprintf(&b, "\t%s = %s\n", v.name, v.expr)
		}
		b.WriteString(")\n\n")
	}
	for _, f := range relations {
		fmt.Fprintf(&b, "// %s filters by the %s relation, taking a predicate from that resource.\nfunc %s(p graphql.Predicate) graphql.Predicate { return graphql.Relation(%q, p) }\n\n", naming.Export(f.Name), f.Name, naming.Export(f.Name), f.Name)
	}
	b.WriteString("// And matches rows satisfying every predicate.\nfunc And(p ...graphql.Predicate) graphql.Predicate { return graphql.And(p...) }\n\n")
	b.WriteString("// Or matches rows satisfying any predicate.\nfunc Or(p ...graphql.Predicate) graphql.Predicate { return graphql.Or(p...) }\n\n")
	b.WriteString("// Not negates a predicate.\nfunc Not(p graphql.Predicate) graphql.Predicate { return graphql.Not(p) }\n")
	return b.String(), usesEnums
}

// fieldHandle returns the graphql field-handle expression for a filterable column and
// whether it references the enums package.
func (r *renderer) fieldHandle(f ir.Field) (string, bool) {
	operand := r.eqOperand(f.Type.Base)
	if operand != "" && r.mapper.IsEnum(operand) {
		return fmt.Sprintf("graphql.EnumField[enumsql.%s]{Col: %q}", operand, f.Name), true
	}
	kind := "graphql.StringField"
	switch r.mapper.LeafGoType(operand) {
	case "bool":
		kind = "graphql.BoolField"
	case "int", "int32", "graphql.Int64":
		kind = "graphql.Int64Field"
	case "float64":
		kind = "graphql.FloatField"
	case "json.RawMessage":
		kind = "graphql.JSONField"
	}
	return fmt.Sprintf("%s{Col: %q}", kind, f.Name), false
}

// eqOperand returns the operand type of a comparison input's _eq (or _in element).
func (r *renderer) eqOperand(cmp string) string {
	in, ok := r.schema.Inputs[cmp]
	if !ok {
		return ""
	}
	for _, f := range in.Fields {
		if f.Name == "_eq" || f.Name == "_in" {
			return f.Type.Base
		}
	}
	return ""
}
