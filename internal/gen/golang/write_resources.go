package golang

import (
	"fmt"
	"path/filepath"
	"strings"
)

// writeResources writes one package per resource (handlers + implementations).
func (g *generator) writeResources() error {
	for _, dg := range g.domains {
		for _, rg := range dg.reses {
			if err := g.writeResource(rg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *generator) writeResource(rg *resGen) error {
	subdir := filepath.Join(rg.domain, rg.pkg)
	if err := g.writeHandlers(subdir, rg); err != nil {
		return err
	}
	if err := g.writeHandlerImpl(subdir, rg.pkg, "queries.go", "queryHandler", rg.queries); err != nil {
		return err
	}
	if err := g.writeHandlerImpl(subdir, rg.pkg, "mutations.go", "mutationHandler", rg.mutations); err != nil {
		return err
	}
	return g.writeHandlerImpl(subdir, rg.pkg, "subscriptions.go", "subscriptionHandler", rg.subs)
}

// writeHandlers writes Params structs, interfaces, and constructors for a resource.
func (g *generator) writeHandlers(subdir string, rg *resGen) error {
	var b strings.Builder
	for _, set := range [][]op{rg.queries, rg.mutations, rg.subs} {
		for _, o := range set {
			if p := g.r.paramsType(o); p != "" {
				b.WriteString(p)
				b.WriteByte('\n')
			}
		}
	}
	g.ifaceBlock(&b, "QueryHandler", "Query", rg.res.Name, "NewQuery", "queryHandler", rg.queries)
	g.ifaceBlock(&b, "MutationHandler", "Mutation", rg.res.Name, "NewMutation", "mutationHandler", rg.mutations)
	g.ifaceBlock(&b, "SubscriptionHandler", "Subscription", rg.res.Name, "NewSubscription", "subscriptionHandler", rg.subs)
	return g.writeFile(subdir, "handlers.go", rg.pkg, g.resImports(b.String()), b.String())
}

// ifaceBlock appends an interface plus its constructor when ops is non-empty.
func (g *generator) ifaceBlock(b *strings.Builder, iface, verb, resource, ctor, recv string, ops []op) {
	if len(ops) == 0 {
		return
	}
	b.WriteString(g.r.iface(iface, fmt.Sprintf("%s runs %s %s operations.", iface, resource, strings.ToLower(verb)), ops))
	fmt.Fprintf(b, "\n// %s returns a %s bound to gql.\nfunc %s(gql *runtime.GraphQLClient) %s { return &%s{gql: gql} }\n\n", ctor, iface, ctor, iface, recv)
}

func (g *generator) writeHandlerImpl(subdir, pkg, file, recv string, ops []op) error {
	if len(ops) == 0 {
		return nil
	}
	body := g.r.handler(recv, ops)
	return g.writeFile(subdir, file, pkg, g.resImports(body), body)
}

// resImports returns the imports for a resource file: context and the runtime facade
// plus any referenced type packages.
func (g *generator) resImports(body string) []string {
	return append([]string{"context", g.opts.RuntimeModule}, g.typeImports(body)...)
}
