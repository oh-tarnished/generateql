// Command generateql is the GenerateQL CLI: it introspects a GraphQL endpoint (or a cached
// schema) and generates a typed Go client library.
package main

import "github.com/oh-tarnished/generateql/cmd"

// Build metadata, injected via -ldflags at release time (see .github/release/goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Execute(version, commit, date)
}
