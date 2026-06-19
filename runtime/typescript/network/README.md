# @machanirobotics/loom-network

Strongly typed transport client for HTTP, GraphQL, and WebSocket with:

- one factory (`NetworkClient`)
- one consistent response style (`meta`, `data`, `errors`)
- both throwing and non-throwing APIs (`*Result`)
- runtime option validation through Zod
- structured package logging suitable for debugging distributed systems

This package is designed to feel predictable when you switch transports. You do not need to re-learn separate connection models for HTTP, GraphQL, and WebSocket.

## Installation

```bash
bun add @machanirobotics/loom-network
```

Use runtime API from `@machanirobotics/loom-network` and types/enums from `@machanirobotics/loom-network/types`.

## Mental model

There are three steps for every transport:

1. Create a client with `NetworkClient`.
2. Connect to get a typed connection object.
3. Execute requests/messages and close when done.

```ts
import { NetworkClient } from "@machanirobotics/loom-network";
import { NetworkType } from "@machanirobotics/loom-network/types";

const client = new NetworkClient(NetworkType.HTTPS, {
  url: { host: "jsonplaceholder.typicode.com", paths: ["/todos/1"] },
});

const connection = await client.connect();
const response = await connection.request({ method: "GET" });
console.log(response.meta.status, response.data);
await connection.close();
```

## Why this package exists

Most projects end up mixing `fetch`, GraphQL clients, and WebSocket code with inconsistent return shapes and error handling. This package normalizes that by giving you:

- a single factory surface
- transport-specific connections with stable method names
- normalized metadata envelopes
- explicit lifecycle states (`idle`, `connecting`, `connected`, `closing`, `closed`)
- optional non-throwing flows to avoid pervasive `try/catch` blocks

## Imports

```ts
import { NetworkClient } from "@machanirobotics/loom-network";
import {
  NetworkType,
  HTTPProtocol,
  type TransportResponse,
} from "@machanirobotics/loom-network/types";
```

`NetworkClient` is your runtime entrypoint. Everything else (types/enums/contracts) should be imported from `@machanirobotics/loom-network/types`.

## Connection options (explained)

The constructor accepts `NetworkClientOptions`:

```ts
{
	url: {
		host: string;        // example: "api.example.com"
		paths: string[];     // example: ["/v1/ping", "/v1/users"]
		scheme?: URLScheme;  // optional; inferred from NetworkType when omitted
		params?: Record<string, string>; // applied as query params
	},
	timeoutMs?: number;                 // default 10_000
	headers?: Record<string, string>;   // default {}
	retries?: number;                   // default 0
	retryDelayMs?: number;              // default 2_000
	skipConnectivityCheck?: boolean;    // default false
	graphQLConnectivityQuery?: string;  // default "query { __typename }"
	websocketReconnectDelayMs?: number; // default 5_000
	httpProtocol?: HTTPProtocol;        // default auto
	logLevel?: LogLevel;                // default info
}
```

### Scheme inference rules

If `url.scheme` is omitted, this package infers:

- `NetworkType.HTTP` -> `http`
- `NetworkType.HTTPS` -> `https`
- `NetworkType.GRAPHQL` -> `https`
- `NetworkType.WEBSOCKET` -> `wss`

If you explicitly pass an incompatible scheme, connect fails early with a protocol error.

## Response shapes

All transport responses include a top-level `meta`. Most also include `data` and `errors`.

### HTTP response

```ts
type HTTPResponse<T> = {
  meta: { status: number; headers: Record<string, string>; transport: "http" };
  status: number; // convenience duplicate of meta.status
  ok: boolean;
  headers: Record<string, string>; // convenience duplicate of meta.headers
  data: T;
  protocol: HTTPProtocol;
  contentType: string;
  errors: string[];
};
```

### GraphQL response

```ts
type GraphQLResult<T> = {
  meta: {
    status: number;
    headers: Record<string, string>;
    transport: "graphql";
  };
  data: T | null;
  errors: Array<{ message: string; path: string[]; code: string }>;
  networkError: string | null;
};
```

### WebSocket message envelope

```ts
type WebSocketMessage = {
  meta: {
    transport: "websocket";
    connected: boolean;
    protocol: string;
    closeCode: number | null;
    closeReason: string | null;
  };
  data: string;
  errors: string[];
};
```

## Throwing vs non-throwing API styles

Every transport method returns a non-throwing `Result` envelope (`connect`, `request`, `query`, `send`, `subscription`, ...).

Non-throwing returns:

```ts
type Result<T, E> = { ok: true; value: T } | { ok: false; error: E };
```

Example:

```ts
const connected = await client.connect();
if (!connected.ok) {
  console.error(connected.error.message);
  return;
}

const response = await connected.value.request<{ id: number }>({
  method: "GET",
  path: "/todos/1",
});

if (response.ok) console.log(response.value.data.id);
```

## HTTP usage

```ts
import { NetworkClient } from "@machanirobotics/loom-network";
import { HTTPProtocol, NetworkType } from "@machanirobotics/loom-network/types";

const httpClient = new NetworkClient(NetworkType.HTTPS, {
  url: { host: "jsonplaceholder.typicode.com", paths: ["/todos/1"] },
  timeoutMs: 8_000,
  retries: 1,
  httpProtocol: HTTPProtocol.HTTP1,
});

const http = await httpClient.connect();
const todo = await http.request<{ id: number; title: string }>({
  method: "GET",
  path: "/todos/1",
});
console.log(todo.meta.status, todo.headers["content-type"], todo.data.title);
await http.close();
```

## GraphQL usage

```ts
import { NetworkClient } from "@machanirobotics/loom-network";
import { NetworkType } from "@machanirobotics/loom-network/types";

const gqlClient = new NetworkClient(NetworkType.GRAPHQL, {
  url: { host: "rickandmortyapi.com", paths: ["/graphql"] },
  timeoutMs: 10_000,
});

const gql = await gqlClient.connect();
const result = await gql.query<{ character: { id: string; name: string } }>(
  `query CharacterById($id: ID!) { character(id: $id) { id name } }`,
  { id: "1" },
);

if (result.networkError) console.error(result.networkError);
else if (result.errors.length) console.error(result.errors);
else console.log(result.data?.character.name);

await gql.close();
```

## GraphQL subscriptions

```ts
const client = new NetworkClient(NetworkType.GRAPHQL, {
  url: { host: "localhost:4000", paths: ["/graphql"] },
  skipConnectivityCheck: true,
});

const connection = await client.connect();
const stop = await connection.subscription("subscription { ping }", (event) => {
  console.log(event.meta.transport, event.data, event.errors);
});

// later
stop();
await connection.close();
```

Important: keep GraphQL scheme as HTTP/HTTPS. The provider derives websocket subscription URL internally.

## WebSocket usage

```ts
const wsClient = new NetworkClient(NetworkType.WEBSOCKET, {
  url: { host: "ws.ifelse.io", paths: ["/"] },
  timeoutMs: 10_000,
  websocketReconnectDelayMs: 2_000,
});

const ws = await wsClient.connect();

const offState = ws.onStateChange((state) => {
  console.log(state.previousState, "->", state.state, state.message);
});

const offMessage = ws.listen((message) => {
  console.log(message.meta.connected, message.data);
});

ws.setAutoReconnect(true, 2_000);
await ws.send("hello");

// later
offMessage();
offState();
await ws.close();
```

## Logging

The package emits structured logs with:

- timestamp
- level
- source file and line
- package name/version/environment
- provider/client tag and message

Environment controls:

- `NODE_ENV` -> displayed environment label
- `LOOM_COLOR=always|never|auto` -> ANSI color mode
- `LOOM_PACKAGE_NAME` -> override package name in log output
- `LOOM_PACKAGE_VERSION` -> override package version in log output

## API docs generation

Generate HTML docs:

```bash
bun run docs
```

Generate JSON docs:

```bash
bun run docs:json
```

Output:

- `docs/` (HTML site)
- `docs/api.json` (JSON model)

## Runtime notes and constraints

- WebSocket handshake headers are not exposed by browser/Bun WebSocket APIs.
- HTTP/3 mode is a preference/validation hint; runtime protocol negotiation still depends on platform and server.
- GraphQL subscriptions require a server that supports websocket GraphQL transport.

## Troubleshooting

- **Connect fails with scheme error**: Ensure `NetworkType` and `url.scheme` are compatible, or omit scheme and let client infer it.
- **GraphQL connect fails during probe**: set `skipConnectivityCheck: true` temporarily and verify endpoint/query separately.
- **WebSocket does not echo payload**: some public endpoints send banners or non-echo traffic first; filter by expected message content.
- **No logs visible**: check `logLevel` option and `LOOM_COLOR` behavior in your terminal/runtime.
