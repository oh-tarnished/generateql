/**
 * Core enum definitions shared across transport types and providers.
 *
 * @remarks These enums drive runtime routing, validation, and lifecycle transitions.
 */
/**
 * Transport families exposed by this package.
 *
 * @remarks
 * Use this enum when creating a `NetworkClient` to select the concrete provider.
 */
export enum NetworkType {
	/** Plain HTTP transport (`http://...`). */
	HTTP = "http",
	/** Secure HTTP transport (`https://...`). */
	HTTPS = "https",
	/** WebSocket transport (`ws://...` or `wss://...`). */
	WEBSOCKET = "websocket",
	/** GraphQL transport over HTTP(S), with optional websocket subscriptions. */
	GRAPHQL = "graphql",
}

/**
 * URL schemes accepted by connection options.
 *
 * @remarks
 * Some transport types only allow a subset of schemes (validated at runtime).
 */
export enum URLScheme {
	/** Unencrypted HTTP scheme. */
	HTTP = "http",
	/** Encrypted HTTP scheme. */
	HTTPS = "https",
	/** Unencrypted websocket scheme. */
	WS = "ws",
	/** Encrypted websocket scheme. */
	WSS = "wss",
}

/**
 * HTTP protocol preference used during connectivity checks.
 *
 * @remarks
 * This is a preference/validation hint and does not force fetch runtime negotiation.
 */
export enum HTTPProtocol {
	/** Let runtime/network negotiation choose protocol automatically. */
	AUTO = "auto",
	/** Prefer HTTP/1.x behavior. */
	HTTP1 = "http1",
	/** Prefer HTTP/3 and validate `alt-svc` support when requested. */
	HTTP3 = "http3",
}

/**
 * Package log verbosity levels ordered from least to most verbose output.
 */
export enum LogLevel {
	/** Disable all package log output. */
	SILENT = "silent",
	/** Log only error events. */
	ERROR = "error",
	/** Log warnings and errors. */
	WARN = "warn",
	/** Log informational, warning, and error events. */
	INFO = "info",
	/** Log all events including debug details. */
	DEBUG = "debug",
}

/**
 * Lifecycle states shared across all transport providers.
 *
 * @remarks
 * Providers transition through these values during connect, close, and reconnect flows.
 */
export enum ConnectionState {
	/** Initial state before any connection attempt. */
	IDLE = "idle",
	/** A connection attempt is currently in progress. */
	CONNECTING = "connecting",
	/** Provider is connected and ready for request/message operations. */
	CONNECTED = "connected",
	/** Provider is actively shutting down resources. */
	CLOSING = "closing",
	/** Provider is closed and not currently connected. */
	CLOSED = "closed",
}
