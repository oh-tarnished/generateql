package ir

import (
	"sort"
	"strings"
)

// groupResources buckets root operations by the row object they act on, so the
// generator can emit one set of files per resource. The resource is derived from
// each operation's return type (see resourceOf).
func groupResources(s *Schema) []*Resource {
	byName := map[string]*Resource{}
	var order []string

	add := func(name string, op *Operation) {
		r, ok := byName[name]
		if !ok {
			r = &Resource{Name: name}
			byName[name] = r
			order = append(order, name)
		}
		switch op.Kind {
		case "mutation":
			r.Mutations = append(r.Mutations, op)
		case "subscription":
			r.Subscriptions = append(r.Subscriptions, op)
		default:
			r.Queries = append(r.Queries, op)
		}
	}

	for _, op := range s.Queries {
		add(resourceOf(s, op), op)
	}
	for _, op := range s.Mutations {
		add(resourceOf(s, op), op)
	}
	for _, op := range s.Subscriptions {
		add(resourceOf(s, op), op)
	}

	sort.Strings(order)
	resources := make([]*Resource, 0, len(order))
	for _, name := range order {
		resources = append(resources, byName[name])
	}
	return resources
}

// resourceOf determines the row-object name an operation belongs to. It unwraps the
// return type and, for mutation response wrappers (which expose a "returning" list of
// rows) and aggregate wrappers, maps back to the underlying row object.
func resourceOf(s *Schema, op *Operation) string {
	base := op.Return.Base
	obj, ok := s.Objects[base]
	if !ok {
		if base == "" {
			return "Root"
		}
		return base
	}

	// Mutation wrappers: {affectedRows, returning: [Row!]!} -> Row.
	for _, f := range obj.Fields {
		if f.Name == "returning" && f.Type.List {
			if _, isObj := s.Objects[f.Type.Base]; isObj {
				return f.Type.Base
			}
		}
	}

	// Aggregate wrappers: XAggExp / XAggregate -> X when the row object exists.
	for _, suffix := range []string{"AggExp", "Aggregate"} {
		if trimmed := strings.TrimSuffix(base, suffix); trimmed != base {
			if _, isObj := s.Objects[trimmed]; isObj {
				return trimmed
			}
		}
	}
	return base
}
