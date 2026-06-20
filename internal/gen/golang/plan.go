package golang

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/oh-tarnished/generateql/internal/ir"
	"github.com/oh-tarnished/generateql/internal/naming"
)

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

// plan groups resources by domain and assigns packages and method names.
func (g *generator) plan() {
	byDomain := map[string]*domainGen{}
	for _, res := range g.opts.Schema.Resources {
		rawDomain, rest := splitDomain(res.Name)
		if rest == "" {
			rest = res.Name
		}
		// Generated packages carry a "ql" suffix (protobuf-style: foldername == package name
		// == import segment, e.g. organisationql, resourceql), keeping them visually distinct
		// from hand-written packages.
		domain := identifier(rawDomain) + "ql"
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
		// Dedup within the domain so distinct resource names never collide on a package
		// directory or aggregator field (which would overwrite files or emit a duplicate
		// struct field).
		pkg := uniqueName(identifier(rest), dg.usedPkg) + "ql"
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

// pairOps assigns each operation a short, unique method name within its kind.
func pairOps(ops []*ir.Operation, resource string) []op {
	used := map[string]bool{}
	out := make([]op, 0, len(ops))
	for _, o := range ops {
		out = append(out, op{Op: o, Name: uniqueName(opShortName(o, resource), used)})
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

// opShortName maps a root field name to a friendly CRUD method within its resource
// package: list -> "Find", byId -> "Get", insertX -> "Create", updateXById -> "Update",
// deleteXById -> "Delete". Subscriptions get an "On" prefix (OnFind/OnGet).
func opShortName(operation *ir.Operation, resource string) string {
	switch operation.Kind {
	case "mutation":
		return mutationShort(operation.Name, resource)
	case "subscription":
		return "On" + queryShort(operation.Name, lowerFirst(resource))
	default:
		return queryShort(operation.Name, lowerFirst(resource))
	}
}

func queryShort(name, resCamel string) string {
	if strings.HasPrefix(name, resCamel) {
		switch rest := name[len(resCamel):]; rest {
		case "":
			return "Find"
		case "ById":
			return "Get"
		default:
			return naming.Export(rest)
		}
	}
	return naming.Export(name)
}

// mutationShort maps insert/update/delete root fields to CRUD verbs, dropping the trailing
// key suffix (e.g. "updateXById" -> "Update", "insertX" -> "Create").
func mutationShort(name, resource string) string {
	verbs := []struct{ prefix, friendly string }{
		{"insert", "Create"}, {"update", "Update"}, {"delete", "Delete"}, {"upsert", "Upsert"},
	}
	for _, v := range verbs {
		if strings.HasPrefix(name, v.prefix) {
			rest := strings.TrimPrefix(name[len(v.prefix):], resource)
			rest = strings.TrimPrefix(rest, "ById")
			if rest == "" {
				return v.friendly
			}
			return v.friendly + naming.Export(rest)
		}
	}
	return naming.Export(name)
}

// uniqueName ensures a name is unique within used, appending a counter on collision.
func uniqueName(base string, used map[string]bool) string {
	name := base
	for i := 2; used[name]; i++ {
		name = fmt.Sprintf("%s%d", base, i)
	}
	used[name] = true
	return name
}

// goKeywords are reserved words that cannot be a generated package name (e.g. a table
// named "Type" -> package "type").
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// identifier lowercases a name and keeps only [a-z0-9] for use as a package name,
// avoiding empty/digit-leading names and Go keywords (invalid as packages).
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
