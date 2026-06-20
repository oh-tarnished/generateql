package golang

import (
	"path"
	"strings"

	"github.com/oh-tarnished/generateql/internal/ir"
)

// enumsDir is the shared enums package, referenced as "enums.". Models are written into a
// per-domain "<domain>/schema" package instead of one global package, and aliased back into
// the domain aggregator (see writeDomain).
const enumsDir = "enumsql"

// domainObjects returns, per domain, the model objects that the domain's operations return
// (rows, responses, aggregates). Because every object inlines its relations, this reachable
// set is self-contained — no cross-domain references — so each domain's schema package
// compiles on its own.
func (g *generator) domainObjects() map[string][]*ir.Object {
	out := map[string][]*ir.Object{}
	for _, dg := range g.domains {
		seen := map[string]*ir.Object{}
		for _, rg := range dg.reses {
			for _, set := range [][]op{rg.queries, rg.mutations, rg.subs} {
				for _, o := range set {
					if obj, ok := g.opts.Schema.Objects[o.Op.Return.Base]; ok {
						seen[obj.Name] = obj
					}
				}
			}
		}
		for _, name := range sortedKeys(seen) {
			out[dg.name] = append(out[dg.name], seen[name])
		}
	}
	return out
}

// writeTypes writes the shared enums package and, per domain, that domain's model objects
// into "<domain>/schema".
func (g *generator) writeTypes() error {
	for _, name := range sortedKeys(g.opts.Schema.Enums) {
		body := g.r.enum(g.opts.Schema.Enums[name])
		if err := g.writeFile(enumsDir, typeFile(name), "enumsql", g.typeImports(body), body); err != nil {
			return err
		}
	}
	for _, domain := range sortedKeys(g.domSchema) {
		dir := domain + "/schemaql"
		for _, obj := range g.domSchema[domain] {
			body := g.r.model(obj)
			if err := g.writeFile(dir, typeFile(obj.Name), "schemaql", g.typeImports(body), body); err != nil {
				return err
			}
		}
	}
	return nil
}

// typeFile returns the file name for a generated type: the schema type name itself,
// kept PascalCase so it contains no underscores. Underscores would let Go misread a
// trailing word as a GOOS/GOARCH build constraint (e.g. "..._windows.go").
func typeFile(name string) string {
	return name + ".go"
}

// typeImports returns the non-schema imports a body needs (json, the runtime graphql
// helpers, and the shared enums package). The per-domain schema import is resolved by the
// caller, which knows the domain.
func (g *generator) typeImports(body string) []string {
	var im []string
	if strings.Contains(body, "json.RawMessage") {
		im = append(im, "encoding/json")
	}
	if strings.Contains(body, "graphql.") {
		im = append(im, g.graphqlModule())
	}
	if strings.Contains(body, "enumsql.") {
		im = append(im, g.opts.GoModule+"/"+enumsDir)
	}
	return im
}

// schemaModule is the import path of a domain's model package.
func (g *generator) schemaModule(domain string) string {
	return g.opts.GoModule + "/" + domain + "/schemaql"
}

// graphqlModule is the import path of the runtime graphql helper package (scalar types,
// predicate DSL, and column helpers), a sibling of the runtime facade module.
func (g *generator) graphqlModule() string {
	return path.Dir(g.opts.RuntimeModule) + "/graphql"
}
