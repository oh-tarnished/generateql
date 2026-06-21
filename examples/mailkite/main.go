package main

import (
	"context"
	"log"
	"net/url"

	"github.com/oh-tarnished/generateql/examples/mailkite/mailkiteql"
	"github.com/oh-tarnished/generateql/examples/mailkite/mailkiteql/bounceql/resourceql"
)

func main() {
	u, err := url.Parse("http://localhost:3280/graphql")
	if err != nil {
		log.Fatalf("failed to parse endpoint URL: %v", err)
	}

	hasura, err := mailkiteql.Connect(u)
	if err != nil {
		log.Fatalf("failed to connect to Hasura: %v", err)
	}
	resp, err := hasura.Mutation.Bounce.Resource.Create(context.Background(), resourceql.CreateInput{})
	if err != nil {
		log.Fatalf("failed to execute query: %v", err)
	}
	log.Printf("created %d resource(s)", resp.AffectedRows)
	for _, r := range resp.Returning {
		log.Printf("Resource: %s", r.Name)
	}

}
