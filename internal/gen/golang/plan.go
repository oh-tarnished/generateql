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
		// Dedup within the domain so distinct resource names never collide on a package
		// directory or aggregator field (which would overwrite files or emit a duplicate
		// struct field).
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

// opShortName shortens a root field name within its resource package (e.g.
// "bookingContacts" -> "List", "insertBookingContacts" -> "Insert"). Subscriptions
// get an "On" prefix.
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
