// Command demo exercises the full generated OrganisationResource API against the
// local GraphQL endpoint: a live subscription, insert, query (by id / list /
// aggregate), update, and delete. Start the endpoint, then run:
//
//	go run ./demo
//
// The subscription is filtered to the row this demo creates, so its result set goes
// 0 -> 1 (insert) -> 1 (update) -> 0 (delete) as the operations run.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/oh-tarnished/generateql/examples/freebusy"
	"github.com/oh-tarnished/generateql/examples/freebusy/organisation/resource"
	"github.com/oh-tarnished/generateql/examples/freebusy/types/inputs"
	"github.com/oh-tarnished/generateql/examples/freebusy/types/schema"
	"github.com/oh-tarnished/generateql/runtime/go/graphql"
	"github.com/oh-tarnished/generateql/runtime/go/runtime"
)

func main() {
	svc, err := freebusy.NewFromURL("http://localhost:3280/graphql", nil)
	must("connect", err)

	ctx := context.Background()
	q := svc.Query.Organisation.Resource
	m := svc.Mutation.Organisation.Resource

	id := uuid.New().String()
	whereID := func() *inputs.OrganisationResourceBoolExp {
		return graphql.Ptr(inputs.OrganisationResourceBoolExp{
			Id: &inputs.TextBoolExp{Eq: graphql.String(id)},
		})
	}

	// ── SUBSCRIBE (live query filtered to this row) ──────────────────────────
	if sub, err := svc.Subscription.Organisation.Resource.OnList(ctx, resource.OnListParams{Where: whereID()}); err != nil {
		fmt.Println("SUBSCRIBE  skipped:", err)
	} else {
		defer sub.Stop()
		go observe(sub)
		time.Sleep(1500 * time.Millisecond) // let the socket connect + initial snapshot
	}

	// ── INSERT ───────────────────────────────────────────────────────────────
	ins, err := m.Insert(ctx, []inputs.InsertOrganisationResourceObjectInput{{
		Id:           id,
		DisplayName:  "BoB the Builder",
		Name:         "organisations/" + id,
		BillingEmail: graphql.String("bob@construction.com"),
		MemberCount:  graphql.Ptr(graphql.Int64(2)),
		UpdateTime:   time.Now().UTC().Format(time.RFC3339),
	}}, resource.InsertParams{})
	must("insert", err)
	fmt.Printf("INSERT     affected=%d  id=%s\n", ins.AffectedRows, id)
	time.Sleep(700 * time.Millisecond)

	// ── QUERY: ById ──────────────────────────────────────────────────────────
	row, err := q.ById(ctx, id)
	must("byId", err)
	fmt.Printf("BY_ID      displayName=%q  memberCount=%d\n", row.DisplayName, int64Of(row.MemberCount))

	// ── QUERY: List (filtered) ───────────────────────────────────────────────
	list, err := q.List(ctx, resource.ListParams{Where: whereID(), Limit: graphql.Int(10)})
	must("list", err)
	fmt.Printf("LIST       matched=%d\n", len(list))

	// ── QUERY: Aggregate ─────────────────────────────────────────────────────
	agg, err := q.Aggregate(ctx, resource.AggregateParams{})
	must("aggregate", err)
	fmt.Printf("AGGREGATE  totalRows=%d\n", int64(agg.Count))

	// ── UPDATE ───────────────────────────────────────────────────────────────
	upd, err := m.UpdateById(ctx, id, inputs.UpdateOrganisationResourceByIdUpdateColumnsInput{
		DisplayName: &inputs.UpdateColumnOrganisationResourceDisplayNameInput{Set: "BoB (updated)"},
	}, resource.UpdateByIdParams{})
	must("update", err)
	newName := ""
	if len(upd.Returning) > 0 {
		newName = upd.Returning[0].DisplayName
	}
	fmt.Printf("UPDATE     affected=%d  -> %q\n", upd.AffectedRows, newName)
	time.Sleep(700 * time.Millisecond)

	// ── DELETE ───────────────────────────────────────────────────────────────
	del, err := m.DeleteById(ctx, id, resource.DeleteByIdParams{})
	must("delete", err)
	fmt.Printf("DELETE     affected=%d\n", del.AffectedRows)
	time.Sleep(700 * time.Millisecond) // let the subscription observe the removal
}

// observe prints each subscription message (the live result-set size) until the
// stream closes or errors.
func observe(sub *runtime.Subscription) {
	for res := range sub.Updates() {
		if res.Error != nil {
			fmt.Println("SUBSCRIBE  error:", res.Error)
			return
		}
		if rows, ok := res.Response.(*[]schema.OrganisationResource); ok {
			fmt.Printf("SUBSCRIBE  live result: %d row(s)\n", len(*rows))
		}
	}
}

func int64Of(p *graphql.Int64) int64 {
	if p == nil {
		return 0
	}
	return int64(*p)
}

func must(label string, err error) {
	if err != nil {
		fmt.Printf("%s error: %v\n", label, err)
		os.Exit(1)
	}
}
