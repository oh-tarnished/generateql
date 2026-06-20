# GenerateQL

Generate a fully-typed Go client from a live GraphQL endpoint.

GenerateQL introspects a GraphQL server (Hasura, Grafbase/DDN, Prisma-backed engines,
anything that supports standard introspection), and emits idiomatic Go: typed models,
input/enum types, and one function per query, mutation, and subscription — organized
into clean per-domain packages and backed by a small transport runtime.

No hand-written GraphQL strings, no hand-written struct tags. Point it at a URL, get a
typed client.

---

## Why

If you work against a GraphQL DB layer you normally either hand-write query strings and
struct tags, or hand-maintain a client. Both drift from the schema. GenerateQL makes the
schema the source of truth:

- **Typed end to end** — every table/type becomes a Go struct; every root field becomes a
  typed function. Wrong field names or types fail at compile time.
- **Idiomatic surface** — `svc.Query.Booking.Contacts.List(ctx, ...)`, not string building.
- **Convention-agnostic** — families (`X` / `XById` / `XAggregate`, `insertX` / `updateXById`
  / `deleteXById`) are detected from introspection, not hardcoded to one engine.
- **Regenerate freely** — re-run on schema changes; the output is deterministic and
  `gofmt`-clean.

---

## How it works

```text
introspect ──> IR ──> selection + typemap ──> Go code generator ──> typed client
 (HTTP POST)   (normalized,   (depth-bounded   (per-domain packages)
               grouped)        struct shapes)
```

1. **Introspect** the endpoint (standard `__schema` query).
2. Normalize into a language-agnostic **IR**: objects, inputs, enums, scalars, and root
   operations grouped per resource.
3. Map GraphQL types to Go and build depth-bounded selection structs.
4. Render Go packages and `gofmt` them.

Generated code runs on the **runtime** in this repo (`runtime/go`), a transport-agnostic
client (GraphQL / HTTP / WebSocket) exposed through a small facade.

---

## Install

Requires Go 1.25+.

```bash
# build the CLI
go build -o generateql .

# or run without installing
go run . <command> [flags]
```

---

## Quick start

```bash
# 1. (optional) cache the schema so you don't re-query on every generate
go run . introspect --endpoint http://localhost:3280/graphql -o schema.json

# 2. generate a client
go run . generate \
  --schema schema.json \
  --out ./client \
  --package myapp \
  --go-module github.com/me/myapp/client \
  --max-depth 1
```

Then use it:

```go
package main

import (
    "context"
    "fmt"

    "github.com/me/myapp/client"
    "github.com/me/myapp/client/prisma/migrations"
    "github.com/oh-tarnished/generateql/runtime/go/graphql"
)

func main() {
    svc, err := client.NewFromURL("http://localhost:3280/graphql", nil)
    if err != nil {
        panic(err)
    }

    rows, err := svc.Query.Prisma.Migrations.List(
        context.Background(),
        migrations.ListParams{Limit: graphql.Int(5)},
    )
    if err != nil {
        panic(err)
    }
    fmt.Printf("%d rows\n", len(rows))
}
```

A complete, runnable example lives in [`examples/freebusy`](examples/freebusy) (generated
client + `demo/`).

---

## Generated code

### Layout

Resources are grouped by **domain** (the first word of the type name), so you get a handful
of top-level folders instead of dozens:

```text
client/
  service.go               Service + New + NewFromURL + aggregators
  booking/
    booking.go             domain aggregator (Query/Mutation/Subscription)
    contacts/
      handlers.go          Params structs + Query/Mutation/Subscription interfaces + constructors
      queries.go           queryHandler implementation
      mutations.go         mutationHandler implementation
      subscriptions.go     subscriptionHandler implementation
    moneys/ ...
  schedule/ resource/ promocode/ identity/ organisation/ prisma/
  types/
    schema/                row models, one file per type (BookingContacts.go, ...)
    inputs/                *BoolExp, *OrderByExp, Insert*Input, ... one file per type
    enums/                 OrderBy.go, ...
```

- **One file per schema type**, named after the type (e.g. `BookingContacts.go`).
- **Three type packages** — `schema` (output models), `inputs`, `enums` — kept separate but
  cycle-free (models inline their relations; inputs reference only inputs + enums).

### Calling operations

The `Service` exposes three aggregators, each nested `domain → resource → method`:

```go
svc.Query.Booking.Contacts.List(ctx, contacts.ListParams{...})
svc.Query.Booking.Contacts.ById(ctx, "id-123")
svc.Mutation.Booking.Contacts.Insert(ctx, objects, contacts.InsertParams{...})
svc.Mutation.Booking.Contacts.UpdateById(ctx, keyId, cols, contacts.UpdateByIdParams{...})
svc.Subscription.Booking.Contacts.OnList(ctx, contacts.OnListParams{...}) // -> *runtime.Subscription
```

Method names are shortened within the resource: `List`, `ById`, `Aggregate`, `Insert`,
`UpdateById`, `DeleteById`, and `On*` for subscriptions.

### Required vs optional arguments

Required (non-null) arguments are **positional**; nullable arguments are bundled into a
generated `<Method>Params` struct, so you never pass positional `nil`s:

```go
type ListParams struct {
    Limit   *int
    Offset  *int
    OrderBy *[]inputs.BookingContactsOrderByExp
    Where   *inputs.BookingContactsBoolExp
}

rows, _ := svc.Query.Booking.Contacts.List(ctx, contacts.ListParams{
    Where: graphql.Ptr(inputs.BookingContactsBoolExp{ /* ... */ }),
    Limit: graphql.Int(20),
})
```

Use the [`graphql`](runtime/go/graphql) helper package for the pointers:
`graphql.Int`, `graphql.String`, `graphql.Bool`, `graphql.Int32`, `graphql.Int64`,
`graphql.Float64`, `graphql.JSON`, and the generic `graphql.Ptr[T]` (for enums/inputs).

### Connecting

```go
// easy path: just a URL (+ optional headers)
svc, err := client.NewFromURL("https://api.example.com/graphql",
    map[string]string{"x-hasura-admin-secret": "secret"})

// full control
svc, err := client.New(runtime.ConnectionOptions{
    URL:     runtime.URLOptions{Scheme: runtime.HTTPS, Host: "api.example.com", Paths: []string{"/graphql"}},
    Headers: map[string]string{"Authorization": "Bearer ..."},
    Timeout: 15 * time.Second,
})
```

> `runtime.HTTP`/`HTTPS` is just the URL scheme of the GraphQL endpoint (GraphQL travels
> over HTTP) — the client itself is the GraphQL client, not a REST client.

---

## CLI reference

### `generateql introspect`

Fetch a server's introspection schema and print it (or write it to a file) for caching.

| Flag | Description |
|------|-------------|
| `--endpoint` | GraphQL endpoint URL (required), e.g. `http://localhost:3280/graphql` |
| `--admin-secret` | Shortcut for the `x-hasura-admin-secret` header |
| `--header` | Extra request header as `'Key: Value'` (repeatable) |
| `-o, --out` | Write schema JSON to this file (default: stdout) |

### `generateql generate`

Generate the Go client from a live endpoint or a cached schema.

| Flag | Default | Description |
|------|---------|-------------|
| `--endpoint` | — | GraphQL endpoint URL (used when `--schema` is not set) |
| `--schema` | — | Path to a cached introspection JSON file |
| `--go-module` | — | **Required.** Import path of the generated root package |
| `-o, --out` | `./generated` | Output directory |
| `--package` | `client` | Go package name for the generated root package |
| `--runtime-module` | `github.com/oh-tarnished/generateql/runtime/go/runtime` | Import path of the runtime facade |
| `--max-depth` | `1` | How many levels of relations to inline into models |
| `--scalar` | — | Scalar override as `GraphQLName=GoType` (repeatable) |
| `--admin-secret` | — | Shortcut for the `x-hasura-admin-secret` header |
| `--header` | — | Extra request header as `'Key: Value'` (repeatable) |

---

## Configuration

### `--go-module` (required)

The import path under which the generated code will live. The generator emits multiple
packages that import each other (resource packages import `<go-module>/types/schema`,
`/types/inputs`, `/types/enums`), so it must know the absolute import path. It should match
the module + subpath where you write the output. Example: output `--out ./client` inside
module `github.com/me/app` → `--go-module github.com/me/app/client`.

### `--runtime-module`

Import path of the runtime facade the generated client calls. Defaults to the runtime in
this repo. Point it at your own vendored/forked copy if you relocate the runtime.

### `--max-depth`

Controls how deep relationships are inlined into a model:

- `0` — scalar fields only.
- `1` (default) — scalars + each direct relation's scalars.
- `n` — relations expanded `n` levels.

Selection is expressed as nested Go structs; a per-branch visited set makes it cycle-safe,
so deeply cyclic schemas never recurse forever. Higher depth = richer single-call results,
but larger generated structs and queries.

### Scalar mapping & `--scalar`

Default GraphQL scalar → Go type mapping (unknown scalars fall back to `string`):

| GraphQL | Go |
|---------|-----|
| `ID`, `String`, `String1`, `Bigdecimal`, `Timestamp`, `Timestamptz` | `string` |
| `Boolean`, `Boolean1` | `bool` |
| `Int` | `int` |
| `Int32` | `int32` |
| `Int64` | `int64` |
| `Float`, `Float64` | `float64` |
| `Json` | `json.RawMessage` |

Override or add mappings with repeatable `--scalar`:

```bash
--scalar Timestamptz=time.Time --scalar Bigdecimal=decimal.Decimal
```

(You're responsible for the corresponding imports/marshaling in those types.)

### Authentication

Both commands accept auth via either flag:

```bash
--admin-secret "$HASURA_ADMIN_SECRET"          # sets x-hasura-admin-secret
--header "Authorization: Bearer $TOKEN"        # any header, repeatable
```

At runtime, pass headers to `NewFromURL(url, headers)` or `New(ConnectionOptions{Headers: ...})`.

---

## The runtime

The generator and runtime live in one module (`github.com/oh-tarnished/generateql`).
Generated clients import two packages from it:

- **`.../runtime/go/runtime`** — the facade the generated code targets. Re-exports the
  network client: `NewConnection`, `ConnectionOptions`, `URLOptions`, `GraphQLClient`,
  `Subscription`, scalar/scheme/client-type constants.
- **`.../runtime/go/graphql`** — pointer constructors and engine-tolerant scalar types
  (below).
- **`.../runtime/go/network`** — the underlying transport: a factory (`NewConnection`) that
  switches between GraphQL, HTTP, and WebSocket clients. The GraphQL client is built on
  [`hasura/go-graphql-client`](https://github.com/hasura/go-graphql-client).

### The `graphql` helper package

- Pointer constructors for optional args: `Ptr[T]`, `Int`, `Int32`, `Float64`, `Bool`,
  `String`, `JSON`.
- **Tolerant scalar types** for engines that vary their JSON encoding: `Int64` and
  `Bigdecimal` decode from a JSON **string or number** (engines often serialize 64-bit
  ints / decimals as strings for precision, but return aggregates as numbers).
- `Var(value, gqlType)` / `VarPtr(...)` wrap an argument with its exact GraphQL type so
  the library declares the right variable type for engine-specific scalars (e.g.
  `String1!`), not the Go-kind default (`String!`).

### Subscriptions

Subscription methods return a `*runtime.Subscription`; read decoded messages from its
`Updates()` channel and call `Stop()` to end it. The runtime uses the modern
**`graphql-transport-ws`** subprotocol (go-graphql-client's `GraphQLWS`); the ws/wss URL
is derived from the endpoint automatically. Live-query engines push the current result
set on connect and again on every change.

---

## Notes & limitations

- **Optional arguments are omitted, not nulled.** A nil optional arg is left out of the
  operation entirely rather than sent as `arg: null` — many engines reject an explicit
  null for filter/check arguments.
- **Engine-specific scalars are handled at runtime.** `Int64`/`Bigdecimal` use tolerant
  types (string-or-number); every argument is wrapped with its exact GraphQL type so
  custom scalars like `String1!` declare correctly. Override the Go mapping with
  `--scalar Name=GoType` when needed.
- **Generated files stay under 400 lines** — types are split one-per-file; large packages are
  naturally partitioned.
- **`.proto` output** is on the roadmap, not yet emitted.
- File names use the exact PascalCase type name (no underscores) on purpose: snake_case names
  ending in a GOOS/GOARCH word (e.g. `..._windows.go`) would be silently excluded by the Go
  toolchain.
- Other languages (Python, TypeScript, Rust) share the same IR and are planned; today the
  generator emits Go.

---

## Repository layout

```text
.
├── cmd/                 Cobra CLI (introspect, generate)
├── internal/
│   ├── introspect/      fetch + decode introspection JSON
│   ├── ir/              normalized schema + resource grouping
│   ├── selection/       depth-bounded model rendering
│   ├── typemap/         GraphQL -> Go type mapping
│   ├── naming/          identifier helpers
│   └── gen/golang/      the Go code generator + templates
├── runtime/go/          transport runtime (network) + facade + graphql helpers
└── examples/freebusy/   a generated client + runnable demo
```

---

## License

Copyright © 2026 oh-tarnished.

Licensed under the **Apache License, Version 2.0**. You may not use this project except in
compliance with the License; you may obtain a copy at
<http://www.apache.org/licenses/LICENSE-2.0>. See the [LICENSE](LICENSE) file for the full
terms. Unless required by applicable law or agreed to in writing, the software is
distributed on an "AS IS" BASIS, without warranties or conditions of any kind.
