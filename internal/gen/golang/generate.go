package golang

import (
	"embed"
	"fmt"
	"go/format"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/oh-tarnished/generate-ql/internal/ir"
	"github.com/oh-tarnished/generate-ql/internal/naming"
	"github.com/oh-tarnished/generate-ql/internal/selection"
	"github.com/oh-tarnished/generate-ql/internal/typemap"
)

//go:embed templates/file.go.tmpl
var templatesFS embed.FS

// Type packages: each generated GraphQL type gets its own file, grouped by kind into
// separate packages so references stay cycle-free.
const (
	modelsDir = "types/schema" // object/row models package, referenced as "schema."
	inputsDir = "types/inputs"
	enumsDir  = "types/enums"
)

// Options configures Go client generation.
type Options struct {
	Schema        *ir.Schema
	OutDir        string
	Package       string            // root package name (Service + New)
	GoModule      string            // import path of the generated root package
	RuntimeModule string            // import path of the runtime facade
	MaxDepth      int               // relation inlining depth
	Scalars       map[string]string // GraphQL scalar -> Go type overrides
}

// fileData is the data passed to the file template.
type fileData struct {
	Package string
	Imports []string
	Body    string
}

// resGen is a resource within a domain, with its per-kind operations.
type resGen struct {
	res        *ir.Resource
	domain     string // e.g. "schedule"
	field      string // e.g. "AvailabilityExceptions"
	pkg        string // e.g. "availabilityexceptions"
	importPath string
	queries    []op
	mutations  []op
	subs       []op
}

// domainGen groups resources sharing a top-level name (e.g. all "schedule*").
type domainGen struct {
	name       string // package name, e.g. "schedule"
	field      string // aggregator field, e.g. "Schedule"
	importPath string
	reses      []*resGen
	usedPkg    map[string]bool // dedup for resource package names within the domain
	usedField  map[string]bool // dedup for resource field names within the domain
}

// Generate renders the full Go client into Options.OutDir.
func Generate(opts Options) error {
	tmpl, err := template.ParseFS(templatesFS, "templates/file.go.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	mapper := typemap.New(opts.Schema, opts.Scalars)
	g := &generator{
		opts: opts,
		tmpl: tmpl,
		r: &renderer{
			schema:    opts.Schema,
			mapper:    mapper,
			selection: selection.New(opts.Schema, mapper, opts.MaxDepth, qModels),
		},
	}
	g.plan()

	if err := g.writeTypes(); err != nil {
		return err
	}
	if err := g.writeResources(); err != nil {
		return err
	}
	if err := g.writeDomains(); err != nil {
		return err
	}
	return g.writeRoot()
}

type generator struct {
	opts    Options
	tmpl    *template.Template
	r       *renderer
	domains []*domainGen
}

// plan groups resources by domain and assigns packages and method names.
func (g *generator) plan() {
	byDomain := map[string]*domainGen{}
	for _, res := range g.opts.Schema.Resources {
		rawDomain, rest := splitDomain(res.Name)
		if rest == "" {
			rest = res.Name
		}
		domain := identifier(rawDomain) // keyword/identifier-safe package name
		dg, ok := byDomain[domain]
		if !ok {
			dg = &domainGen{
				name:       domain,
				field:      naming.Export(rawDomain),
				importPath: g.opts.GoModule + "/" + domain,
				usedPkg:    map[string]bool{},
				usedField:  map[string]bool{},
			}
			byDomain[domain] = dg
		}
		// Dedup within the domain so distinct resource names never collide on a
		// package directory or aggregator field (which would overwrite files or emit
		// a duplicate struct field).
		pkg := uniqueName(identifier(rest), dg.usedPkg)
		dg.reses = append(dg.reses, &resGen{
			res:        res,
			domain:     domain,
			field:      uniqueName(naming.Export(rest), dg.usedField),
			pkg:        pkg,
			importPath: g.opts.GoModule + "/" + domain + "/" + pkg,
			queries:    pairOps(res.Queries, res.Name),
			mutations:  pairOps(res.Mutations, res.Name),
			subs:       pairOps(res.Subscriptions, res.Name),
		})
	}
	for _, name := range sortedKeys(byDomain) {
		dg := byDomain[name]
		sort.Slice(dg.reses, func(i, j int) bool { return dg.reses[i].field < dg.reses[j].field })
		g.domains = append(g.domains, dg)
	}
}

func pairOps(ops []*ir.Operation, resource string) []op {
	used := map[string]bool{}
	out := make([]op, 0, len(ops))
	for _, o := range ops {
		out = append(out, op{Op: o, Name: uniqueName(opShortName(o, resource), used)})
	}
	return out
}

// ---- file writing -----------------------------------------------------------

func (g *generator) writeFile(subdir, name, pkg string, imports []string, body string) error {
	dir := filepath.Join(g.opts.OutDir, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}
	var raw strings.Builder
	if err := g.tmpl.Execute(&raw, fileData{Package: pkg, Imports: imports, Body: body}); err != nil {
		return fmt.Errorf("template exec for %s: %w", name, err)
	}
	formatted, err := format.Source([]byte(raw.String()))
	if err != nil {
		return fmt.Errorf("gofmt %s/%s: %w", subdir, name, err)
	}
	return os.WriteFile(filepath.Join(dir, name), formatted, 0o644)
}

// ---- types packages ---------------------------------------------------------

// writeTypes writes one file per GraphQL type, named after the type, into the enums,
// models, and inputs packages.
func (g *generator) writeTypes() error {
	for _, name := range sortedKeys(g.opts.Schema.Enums) {
		body := g.r.enum(g.opts.Schema.Enums[name])
		if err := g.writeFile(enumsDir, typeFile(name), "enums", g.typeImports(body), body); err != nil {
			return err
		}
	}
	for _, name := range sortedKeys(g.opts.Schema.Objects) {
		body := g.r.model(g.opts.Schema.Objects[name])
		if err := g.writeFile(modelsDir, typeFile(name), "schema", g.typeImports(body), body); err != nil {
			return err
		}
	}
	for _, name := range sortedKeys(g.opts.Schema.Inputs) {
		body := g.r.input(g.opts.Schema.Inputs[name])
		if err := g.writeFile(inputsDir, typeFile(name), "inputs", g.typeImports(body), body); err != nil {
			return err
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

// typeImports returns the imports a file needs based on the qualified type references
// and json usage in its body.
func (g *generator) typeImports(body string) []string {
	var im []string
	if strings.Contains(body, "json.RawMessage") {
		im = append(im, "encoding/json")
	}
	if strings.Contains(body, "graphql.") {
		im = append(im, g.graphqlModule())
	}
	if strings.Contains(body, "enums.") {
		im = append(im, g.opts.GoModule+"/"+enumsDir)
	}
	if strings.Contains(body, "inputs.") {
		im = append(im, g.opts.GoModule+"/"+inputsDir)
	}
	if strings.Contains(body, "schema.") {
		im = append(im, g.opts.GoModule+"/"+modelsDir)
	}
	return im
}

// graphqlModule is the import path of the runtime graphql helper package (scalar types
// and pointer constructors), a sibling of the runtime facade module.
func (g *generator) graphqlModule() string {
	return path.Dir(g.opts.RuntimeModule) + "/graphql"
}

// ---- resource packages ------------------------------------------------------

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

func (g *generator) resImports(body string) []string {
	return append([]string{"context", g.opts.RuntimeModule}, g.typeImports(body)...)
}

// ---- domain packages --------------------------------------------------------

func (g *generator) writeDomains() error {
	for _, dg := range g.domains {
		if err := g.writeDomain(dg); err != nil {
			return err
		}
	}
	return nil
}

// writeDomain writes a domain package aggregating its resources' handlers.
func (g *generator) writeDomain(dg *domainGen) error {
	var b strings.Builder
	imports := map[string]bool{g.opts.RuntimeModule: true}
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

// ---- root package -----------------------------------------------------------

func (g *generator) writeRoot() error {
	var b strings.Builder
	imports := map[string]bool{g.opts.RuntimeModule: true}

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
	b.WriteString(serviceFromURL)

	imports["net/url"] = true
	return g.writeFile("", "service.go", g.opts.Package, sortedKeys(imports), b.String())
}

// serviceFromURL is the source of the NewFromURL convenience constructor.
const serviceFromURL = `// NewFromURL connects using a full endpoint URL (e.g.
// "http://localhost:3280/graphql") and optional request headers, instead of a manual
// ConnectionOptions. Pass nil headers when none are needed.
func NewFromURL(endpoint string, headers map[string]string) (*Service, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	scheme := runtime.HTTPS
	if u.Scheme == "http" {
		scheme = runtime.HTTP
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
	return New(runtime.ConnectionOptions{
		URL:     runtime.URLOptions{Scheme: scheme, Host: u.Host, Paths: []string{path}},
		Headers: headers,
	})
}
`

// ---- kind specs & helpers ---------------------------------------------------

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

// splitDomain splits a PascalCase resource name into a lowercase domain (its first
// word) and the remainder (e.g. "ScheduleBufferSettings" -> "schedule", "BufferSettings").
func splitDomain(name string) (domain, rest string) {
	runes := []rune(name)
	i := 1
	for i < len(runes) && !unicode.IsUpper(runes[i]) {
		i++
	}
	return strings.ToLower(string(runes[:i])), string(runes[i:])
}

// opShortName shortens a root field name within its resource package.
func opShortName(operation *ir.Operation, resource string) string {
	switch operation.Kind {
	case "mutation":
		for _, verb := range []string{"insert", "update", "delete", "upsert"} {
			if strings.HasPrefix(operation.Name, verb) {
				rest := strings.TrimPrefix(operation.Name[len(verb):], resource)
				if rest == "" {
					return naming.Export(verb)
				}
				return naming.Export(verb) + naming.Export(rest)
			}
		}
		return naming.Export(operation.Name)
	case "subscription":
		return "On" + queryShort(operation.Name, lowerFirst(resource))
	default:
		return queryShort(operation.Name, lowerFirst(resource))
	}
}

func queryShort(name, resCamel string) string {
	if strings.HasPrefix(name, resCamel) {
		rest := name[len(resCamel):]
		if rest == "" {
			return "List"
		}
		return naming.Export(rest)
	}
	return naming.Export(name)
}

func uniqueName(base string, used map[string]bool) string {
	name := base
	for i := 2; used[name]; i++ {
		name = fmt.Sprintf("%s%d", base, i)
	}
	used[name] = true
	return name
}

// goKeywords are the reserved words that cannot be used as identifiers, so they cannot
// be a generated package name (e.g. a table named "Type" -> package "type").
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// identifier lowercases a name and keeps only [a-z0-9] for use as a package name,
// avoiding empty/digit-leading names and Go keywords (which are invalid as packages).
func identifier(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" || (s[0] >= '0' && s[0] <= '9') {
		s = "res" + s
	}
	if goKeywords[s] {
		s += "_"
	}
	return s
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
