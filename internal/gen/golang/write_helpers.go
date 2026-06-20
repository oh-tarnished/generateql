package golang

// writeHelpers writes the root field.go, which re-exports the runtime constructors and
// scalar types. It lets generated clients depend on a single import (the root package)
// for building inputs and reading results, instead of reaching into the runtime
// graphql/param/runtime packages directly.
func (g *generator) writeHelpers() error {
	imports := []string{g.graphqlModule(), g.paramModule(), g.opts.RuntimeModule}
	return g.writeFile("", "field.go", g.opts.Package, imports, helpersBody)
}

// helpersBody is the body of the generated root field.go. The constructors return
// param.Opt values so optional input fields are set without pointers (e.g.
// {Eq: pkg.String(id)} rather than {Eq: &s}).
const helpersBody = `// Subscription is a live GraphQL subscription stream.
type Subscription = runtime.Subscription

// Int64 and Bigdecimal are precision-preserving scalar types shared by model fields and
// input values; re-exported so reading results needs no extra import.
type (
	Int64      = graphql.Int64
	Bigdecimal = graphql.Bigdecimal
)

// String sets an optional string input field without taking a pointer.
func String(v string) param.Opt[string] { return param.NewOpt(v) }

// Int sets an optional int input field.
func Int(v int) param.Opt[int] { return param.NewOpt(v) }

// Int32 sets an optional int32 input field.
func Int32(v int32) param.Opt[int32] { return param.NewOpt(v) }

// Bool sets an optional bool input field.
func Bool(v bool) param.Opt[bool] { return param.NewOpt(v) }

// Float sets an optional float64 input field.
func Float(v float64) param.Opt[float64] { return param.NewOpt(v) }

// Opt sets an optional input field of any comparable type, such as an enum or Int64
// value: Opt(Int64(2)) or Opt(enums.OrderByAsc).
func Opt[T comparable](v T) param.Opt[T] { return param.NewOpt(v) }

// Ptr returns a pointer to v, for the rare API that still needs one.
func Ptr[T any](v T) *T { return &v }
`
