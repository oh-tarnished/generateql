/**
 * WebSocket message metadata and connection contract definitions.
 */
import type { LoomNetworkError } from "../errors";
import type {
	ConnectionStateHandler,
	ConnectionStateResult,
	Result,
	TransportConnection,
} from "./lifecycle";

/** WebSocket transport metadata envelope. */
export interface WebSocketMeta {
	/** Transport discriminator used for narrowing. */
	transport: "websocket";
	/** Indicates whether the socket is currently open. */
	connected: boolean;
	/** Negotiated WebSocket subprotocol, if any. */
	protocol: string;
	/** Close event code from the last socket close event. */
	closeCode: number | null;
	/** Close event reason from the last socket close event. */
	closeReason: string | null;
}

/** Normalized WebSocket message envelope. */
export interface WebSocketMessage {
	/** WebSocket connection metadata at message time. */
	meta: WebSocketMeta;
	/** Raw message string payload. */
	data: string;
	/** Envelope-level processing errors. */
	errors: string[];
}

/** Callback signature for inbound WebSocket messages. */
export type MessageHandler = (message: WebSocketMessage) => void;

/** Request object accepted by generic websocket `execute`. */
export interface WebSocketSendRequest {
	/** Message string to send over the socket. */
	message: string;
}

/** WebSocket transport contract for send/listen/state APIs. */
export interface WebSocketConnection
	extends TransportConnection<WebSocketSendRequest, WebSocketMessage> {
	/** Executes one send request. */
	execute(
		request: WebSocketSendRequest,
	): Promise<Result<WebSocketMessage, LoomNetworkError>>;
	/** Sends one string payload through the open socket. */
	send(message: string): Promise<Result<WebSocketMessage, LoomNetworkError>>;
	/** Subscribes to inbound messages and returns an unsubscribe function. */
	listen(handler: MessageHandler): () => void;
	/** Subscribes to lifecycle transitions and returns an unsubscribe function. */
	onStateChange(handler: ConnectionStateHandler): () => void;
	/** Enables/disables automatic reconnect behavior. */
	setAutoReconnect(enabled: boolean, delayMs?: number): ConnectionStateResult;
}
