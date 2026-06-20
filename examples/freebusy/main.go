package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/oh-tarnished/generateql/examples/freebusy/freebusyql"
	"github.com/oh-tarnished/generateql/examples/freebusy/freebusyql/organisationql/resourceql"
)

// Config holds the application configuration
type Config struct {
	Endpoint string
}

func main() {
	// 1. Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. Parse configuration
	var cfg Config
	flag.StringVar(&cfg.Endpoint, "endpoint", "http://localhost:3280/graphql", "GraphQL endpoint URL")
	flag.Parse()

	// 3. Execute main logic and handle top-level errors gracefully
	if err := run(context.Background(), cfg); err != nil {
		slog.Error("Demo execution failed", "error", err)
		os.Exit(1)
	}
}

// run contains the core business logic, allowing defers to execute properly before exiting.
func run(ctx context.Context, cfg Config) error {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse endpoint URL: %w", err)
	}

	svc, err := freebusyql.Connect(u)
	if err != nil {
		return fmt.Errorf("failed to connect to GraphQL endpoint: %w", err)
	}
	l, err := svc.Query.Resource.Entity.Find(ctx)
	if err != nil {
		return fmt.Errorf("failed to find resource entity: %w", err)
	}
	slog.Info("Resource entity found", "entityCount", len(l))
	for _, e := range l {
		slog.Info("Resource entity", "id", e.Id, "name", e.Name)
	}

	// Setup client aliases for brevity
	q := svc.Query.Organisation.Resource
	m := svc.Mutation.Organisation.Resource
	id := uuid.New().String()

	slog.Info("Starting demo", "id", id, "endpoint", cfg.Endpoint)

	// ── SUBSCRIBE (live query filtered to this row) ──────────────────────────
	sub, err := svc.Subscription.Organisation.Resource.OnFind(ctx, resourceql.OnFind().Where(resourceql.Id.Eq(id)))
	if err != nil {
		slog.Warn("Subscription skipped or unavailable", "error", err)
	} else {
		defer sub.Stop() // stop the stream when run returns, not now
		slog.Info("Subscription established, observing live updates for this resource")
		go observe(sub)

		// Allow socket to connect and fetch the initial snapshot
		time.Sleep(1500 * time.Millisecond)
	}

	// ── CREATE (one row, native fields) ──────────────────────────────────────
	ins, err := m.Create(ctx, resourceql.CreateInput{
		Id:           id,
		DisplayName:  "BoB the Builder",
		Name:         "organisations/" + id,
		BillingEmail: "bob@construction.com",
		MemberCount:  2,
		UpdateTime:   time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("create operation failed: %w", err)
	}
	slog.Info("CREATE successful", "affectedRows", ins.AffectedRows, "id", id)
	time.Sleep(700 * time.Millisecond) // Demo pacing

	row, err := q.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("get operation failed: %w", err)
	}
	slog.Info("GET successful", "displayName", row.DisplayName, "memberCount", int64Of(row.MemberCount))

	list, err := q.Find(ctx, resourceql.Find().Where(resourceql.Id.Eq(id)).Limit(10))
	if err != nil {
		return fmt.Errorf("find operation failed: %w", err)
	}
	slog.Info("FIND successful", "matchedCount", len(list))

	// ── QUERY: Aggregate (filtered via filter_input) ─────────────────────────
	agg, err := q.Aggregate(ctx, resourceql.Aggregate().Where(resourceql.Id.Eq(id)))
	if err != nil {
		return fmt.Errorf("aggregate operation failed: %w", err)
	}
	slog.Info("AGGREGATE successful", "matchedRows", int64(agg.Count))

	upd, err := m.Update(ctx, id, resourceql.UpdateInput{DisplayName: "BoB (updated)"})
	if err != nil {
		return fmt.Errorf("update operation failed: %w", err)
	}

	var newName string
	if len(upd.Returning) > 0 {
		newName = upd.Returning[0].DisplayName
	}
	slog.Info("UPDATE successful", "affectedRows", upd.AffectedRows, "newName", newName)
	time.Sleep(700 * time.Millisecond)

	del, err := m.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete operation failed: %w", err)
	}
	slog.Info("DELETE successful", "affectedRows", del.AffectedRows)
	time.Sleep(700 * time.Millisecond) // Allow subscription to observe the removal

	return nil
}

// observe prints each subscription message until the stream closes or errors.
func observe(sub *freebusyql.Subscription) {
	for res := range sub.Updates() {
		if res.Error != nil {
			slog.Error("Subscription stream error", "error", res.Error)
			return
		}
		if rows, ok := res.Response.(*[]resourceql.OrganisationResource); ok {
			slog.Info("SUBSCRIBE event received", "liveResultCount", len(*rows))
		}
	}
}

// int64Of safely dereferences an int64 pointer, returning 0 if nil.
func int64Of(p *freebusyql.Int64) int64 {
	if p == nil {
		return 0
	}
	return int64(*p)
}
