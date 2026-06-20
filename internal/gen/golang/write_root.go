package golang

import (
	"fmt"
	"strings"
)

// writeDomains writes one package per domain aggregating its resources' handlers.
func (g *generator) writeDomains() error {
	for _, dg := range g.domains {
		if err := g.writeDomain(dg); err != nil {
			return err
		}
	}
	return nil
}

// writeDomain writes a domain package aggregating its resources' handlers and aliasing the
// domain's model types (so callers read results via <domain>.<Model> with one import).
func (g *generator) writeDomain(dg *domainGen) error {
	var b strings.Builder
	imports := map[string]bool{g.opts.RuntimeModule: true}
	if objs := g.domSchema[dg.name]; len(objs) > 0 {
		imports[g.schemaModule(dg.name)] = true
		b.WriteString("// Model type aliases for this domain, re-exported from its schema package.\n")
		for _, obj := range objs {
			fmt.Fprintf(&b, "type %s = schemaql.%s\n", obj.Name, obj.Name)
		}
		b.WriteByte('\n')
	}
	for _, spec := range kindSpecs() {
		members := domainMembers(dg, spec.pick)
		if len(members) == 0 {
			continue
		}
		fmt.Fprintf(&b, "// %s aggregates %s handlers for the %s domain.\ntype %s struct {\n", spec.iface, spec.verb, dg.name, spec.iface)
		for _, m := range members {
			fmt.Fprintf(&b, "\t%s %s.%s\n", m.field, m.pkg, spec.iface)
			imports[m.importPath] = true
		}
		b.WriteString("}\n\n")
		fmt.Fprintf(&b, "// %s wires every %s handler in the domain.\nfunc %s(gql *runtime.GraphQLClient) %s {\n\treturn %s{\n", spec.ctor, spec.verb, spec.ctor, spec.iface, spec.iface)
		for _, m := range members {
			fmt.Fprintf(&b, "\t\t%s: %s.%s(gql),\n", m.field, m.pkg, spec.ctor)
		}
		b.WriteString("\t}\n}\n\n")
	}
	return g.writeFile(dg.name, dg.name+".go", dg.name, sortedKeys(imports), b.String())
}

// writeRoot writes the root Service, its domain aggregators, and the New/NewFromURL
// constructors.
func (g *generator) writeRoot() error {
	var b strings.Builder
	imports := map[string]bool{g.opts.RuntimeModule: true, "net/url": true}

	b.WriteString("// Service is a typed GraphQL client. Access operations via\n")
	b.WriteString("// s.Query.<Domain>.<Resource>, s.Mutation..., and s.Subscription....\n")
	b.WriteString("type Service struct {\n\tQuery QueryHandler\n\tMutation MutationHandler\n\tSubscription SubscriptionHandler\n}\n\n")

	for _, spec := range kindSpecs() {
		fmt.Fprintf(&b, "// %s groups every domain's %s handlers.\ntype %s struct {\n", spec.iface, spec.verb, spec.iface)
		for _, dg := range g.domains {
			if len(domainMembers(dg, spec.pick)) == 0 {
				continue
			}
			fmt.Fprintf(&b, "\t%s %s.%s\n", dg.field, dg.name, spec.iface)
			imports[dg.importPath] = true
		}
		b.WriteString("}\n\n")
	}

	b.WriteString("// New connects to the endpoint described by opts and returns a Service.\n")
	b.WriteString("func New(opts runtime.ConnectionOptions) (*Service, error) {\n")
	b.WriteString("\tconn, err := runtime.NewConnection(runtime.GraphQLConnClient)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n")
	b.WriteString("\tif _, err := conn.WithOpts(opts); err != nil {\n\t\treturn nil, err\n\t}\n")
	b.WriteString("\tgql, err := conn.AsGraphQLConnectionType()\n\tif err != nil {\n\t\treturn nil, err\n\t}\n")
	b.WriteString("\treturn &Service{\n")
	for _, spec := range kindSpecs() {
		fmt.Fprintf(&b, "\t\t%s: %s{\n", spec.field, spec.iface)
		for _, dg := range g.domains {
			if len(domainMembers(dg, spec.pick)) == 0 {
				continue
			}
			fmt.Fprintf(&b, "\t\t\t%s: %s.%s(gql),\n", dg.field, dg.name, spec.ctor)
		}
		b.WriteString("\t\t},\n")
	}
	b.WriteString("\t}, nil\n}\n\n")
	b.WriteString(serviceConnect)

	return g.writeFile("", "service.go", g.opts.Package, sortedKeys(imports), b.String())
}

// serviceConnect is the source of the Connect convenience constructor.
const serviceConnect = `// Connect dials the GraphQL endpoint at u (e.g. url.Parse("http://localhost:3280/graphql")).
// Request headers are optional; pass one map only if the endpoint needs them.
func Connect(u *url.URL, headers ...map[string]string) (*Service, error) {
	var h map[string]string
	if len(headers) > 0 {
		h = headers[0]
	}
	return New(runtime.ConnectionOptions{URL: runtime.URLFromStd(u), Headers: h})
}
`

// kindSpec describes one operation kind's aggregator naming.
type kindSpec struct {
	field string // Service field: "Query"
	iface string // aggregator/interface type: "QueryHandler"
	verb  string // "query"
	ctor  string // "NewQuery"
	pick  func(*resGen) []op
}

func kindSpecs() []kindSpec {
	return []kindSpec{
		{"Query", "QueryHandler", "query", "NewQuery", func(r *resGen) []op { return r.queries }},
		{"Mutation", "MutationHandler", "mutation", "NewMutation", func(r *resGen) []op { return r.mutations }},
		{"Subscription", "SubscriptionHandler", "subscription", "NewSubscription", func(r *resGen) []op { return r.subs }},
	}
}

// member is a resource field within a domain aggregator.
type member struct {
	field      string
	pkg        string
	importPath string
}

// domainMembers returns the resources in dg that have ops for the picked kind.
func domainMembers(dg *domainGen, pick func(*resGen) []op) []member {
	var out []member
	for _, rg := range dg.reses {
		if len(pick(rg)) > 0 {
			out = append(out, member{field: rg.field, pkg: rg.pkg, importPath: rg.importPath})
		}
	}
	return out
}
