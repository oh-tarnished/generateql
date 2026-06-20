package golang

// writeHelpers writes the root field.go, re-exporting the runtime scalar types and the
// Subscription type so generated clients read results and hold subscriptions without
// importing the runtime packages directly.
func (g *generator) writeHelpers() error {
	imports := []string{g.graphqlModule(), g.opts.RuntimeModule}
	return g.writeFile("", "field.go", g.opts.Package, imports, helpersBody)
}

// helpersBody is the body of the generated root field.go.
const helpersBody = `// Subscription is a live GraphQL subscription stream.
type Subscription = runtime.Subscription

// Int64 and Bigdecimal are precision-preserving scalar types used by model fields and
// input values; re-exported so reading results needs no extra import.
type (
	Int64      = graphql.Int64
	Bigdecimal = graphql.Bigdecimal
)

// Ptr returns a pointer to v, for the rare API that still needs one.
func Ptr[T any](v T) *T { return &v }
`
