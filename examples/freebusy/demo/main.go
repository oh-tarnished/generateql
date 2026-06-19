// Command demo shows how to use the generated freebusy client against the local
// GraphQL endpoint. Start the endpoint, then run:
//
//	go run ./demo
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/oh-tarnished/generate-ql/examples/freebusy"
	"github.com/oh-tarnished/generate-ql/examples/freebusy/prisma/migrations"
	"github.com/oh-tarnished/generate-ql/runtime/go/graphql"
)

func main() {
	// NewFromURL is the easy path: just the endpoint (+ optional headers).
	svc, err := freebusy.NewFromURL("http://localhost:3280/graphql", nil)
	if err != nil {
		fmt.Println("connect error:", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// s.Query.<Domain>.<Resource>.<Method>(ctx, Params{...}); graphql.Int for optionals.
	rows, err := svc.Query.Prisma.Migrations.List(ctx, migrations.ListParams{Limit: graphql.Int(5)})
	if err != nil {
		fmt.Println("query error:", err)
		os.Exit(1)
	}
	fmt.Printf("prismaMigrations: %d row(s)\n", len(rows))
	for _, r := range rows {
		fmt.Printf("  - %s  %s\n", r.Id, r.MigrationName)
	}
}
