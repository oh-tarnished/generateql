/**
 * Shared lifecycle/result contracts for all transport providers.
 */
import type { LoomNetworkError } from "../errors";
import type { ConnectionState } from "./enums";
import type { ConnectionOptions } from "./schemas";

/** Normalized state transition result payload. */
export interface ConnectionStateResult {
	/** State after the transition attempt. */
	state: ConnectionState;
	/** State before the transition attempt. */
	previousState: ConnectionState;
	/** True when `state` differs from `previousState`. */
	changed: boolean;
	/** ISO timestamp captured at transition time. */
	timestamp: string;
	/** Human-readable transition detail. */
	message: string;
}

/** Callback signature for connection state change events. */
export type ConnectionStateHandler = (result: ConnectionStateResult) => void;

/** Non-throwing result wrapper for API calls. */
export type Result<TValue, TError extends LoomNetworkError> =
	| { ok: true; value: TValue }
	| { ok: false; error: TError };

/** Lifecycle operations implemented by all transport providers. */
export interface ConnectionLifecycle {
	/** Connects the provider and transitions state to connected. */
	connect(options?: ConnectionOptions): Promise<ConnectionStateResult>;
	/** Non-throwing wrapper for `connect`. */
	connectResult(
		options?: ConnectionOptions,
	): Promise<Result<ConnectionStateResult, LoomNetworkError>>;
	/** Closes provider resources and transitions state to closed. */
	close(): Promise<ConnectionStateResult>;
	/** Non-throwing wrapper for `close`. */
	closeResult(): Promise<Result<ConnectionStateResult, LoomNetworkError>>;
	/** Re-establishes connectivity after a close. */
	reconnect(): Promise<ConnectionStateResult>;
	/** Non-throwing wrapper for `reconnect`. */
	reconnectResult(): Promise<Result<ConnectionStateResult, LoomNetworkError>>;
	/** Returns current lifecycle state. */
	state(): ConnectionState;
}

/** Generic execute contract for any transport request/response pair. */
export interface TransportConnection<TRequest, TResponse>
	extends ConnectionLifecycle {
	/** Executes one request and returns the transport-specific response envelope. */
	execute(request: TRequest): Promise<Result<TResponse, LoomNetworkError>>;
}
