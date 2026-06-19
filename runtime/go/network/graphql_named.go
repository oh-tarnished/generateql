// Dynamic-field GraphQL operations.
//
// QueryFields/MutateFields build an operation from a field name, a result value, and
// a variables map, declaring ONLY the arguments present in the map. Generated clients
// use these so that optional (nil) arguments are omitted entirely rather than sent as
// explicit nulls — many engines reject `arg: null` for filters.
package network

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// QueryFields runs a query selecting result under field with the given arguments.
// result must be a pointer to the typed result; it is filled on success.
func (g *GraphQLClient) QueryFields(field string, result any, args map[string]interface{}) <-chan GraphQLResult {
	return g.execFields(false, field, result, args)
}

// MutateFields runs a mutation selecting result under field with the given arguments.
func (g *GraphQLClient) MutateFields(field string, result any, args map[string]interface{}) <-chan GraphQLResult {
	return g.execFields(true, field, result, args)
}

// execFields builds a single-field operation struct with a dynamic graphql tag, runs
// it, and copies the decoded field back into result.
func (g *GraphQLClient) execFields(mutation bool, field string, result any, args map[string]interface{}) <-chan GraphQLResult {
	resultChan := make(chan GraphQLResult, 1)
	go func() {
		defer close(resultChan)
		if g.client == nil {
			resultChan <- GraphQLResult{Error: fmt.Errorf("GraphQL client is not initialized")}
			return
		}
		rv := reflect.ValueOf(result)
		if rv.Kind() != reflect.Ptr {
			resultChan <- GraphQLResult{Error: fmt.Errorf("result must be a pointer")}
			return
		}
		wrapper := newOpStruct(field, rv.Type().Elem(), args)

		ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
		defer cancel()

		var err error
		if mutation {
			err = g.client.Mutate(ctx, wrapper.Interface(), args)
		} else {
			err = g.client.Query(ctx, wrapper.Interface(), args)
		}
		if err != nil {
			resultChan <- GraphQLResult{Error: fmt.Errorf("failed to execute operation: %w", err)}
			return
		}
		rv.Elem().Set(wrapper.Elem().Field(0))
		resultChan <- GraphQLResult{Response: result}
	}()
	return resultChan
}

// newOpStruct returns a pointer to a freshly built struct{ Result T `graphql:"field(...)"` },
// where the argument list contains only the keys present in args.
func newOpStruct(field string, resultType reflect.Type, args map[string]interface{}) reflect.Value {
	st := reflect.StructOf([]reflect.StructField{{
		Name: "Result",
		Type: resultType,
		Tag:  reflect.StructTag(fmt.Sprintf("graphql:%q", buildFieldTag(field, args))),
	}})
	return reflect.New(st)
}

// buildFieldTag renders `field(name: $name, ...)` for the arguments present (sorted for
// determinism), or just `field` when there are none.
func buildFieldTag(field string, args map[string]interface{}) string {
	if len(args) == 0 {
		return field
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: $%s", k, k))
	}
	return fmt.Sprintf("%s(%s)", field, strings.Join(parts, ", "))
}
