/**
 * WebSocket provider implementation module.
 */

import {
	type LoomNetworkError,
	LoomProtocolError,
	toLoomNetworkError,
} from "../../errors";
import { Provider } from "../../provider";
import {
	type ConnectionOptions,
	ConnectionState,
	type ConnectionStateHandler,
	type ConnectionStateResult,
	type MessageHandler,
	type NetworkType,
	type Result,
	URLScheme,
	type WebSocketConnection,
	type WebSocketMessage,
	type WebSocketSendRequest,
} from "../../types/types";
import {
	bindWebSocketListeners,
	openWebSocket,
	toWebSocketMessage,
} from "./websocket_runtime";

/**
 * WebSocket transport provider with listen/send, auto-reconnect, and state events.
 */
export class WebSocketProvider
	extends Provider<NetworkType.WEBSOCKET>
	implements WebSocketConnection
{
	private socket: WebSocket | null = null;
	private listeners: Set<MessageHandler> = new Set<MessageHandler>();
	private autoReconnect = false;
	private reconnecting = false;
	private lastCloseCode: number | null = null;
	private lastCloseReason: string | null = null;
	private stateListeners: Set<ConnectionStateHandler> =
		new Set<ConnectionStateHandler>();

	/** Opens a websocket connection and transitions to connected state. */
	override async connect(
		options?: ConnectionOptions,
	): Promise<ConnectionStateResult> {
		this.emitState(this.ensureStateAllowed(ConnectionState.CONNECTING));
		if (options) this.setOptions(options);
		const current = this.getOptions();
		this.logInfo(
			"connect start",
			`host=${current.url.host} scheme=${current.url.scheme}`,
		);
		if (![URLScheme.WS, URLScheme.WSS].includes(current.url.scheme)) {
			this.logError("connect failed", `invalid scheme=${current.url.scheme}`);
			throw new LoomProtocolError(
				`invalid WebSocket scheme: ${current.url.scheme}`,
			);
		}
		const target = this.buildURL(0);
		this.socket = await openWebSocket(target, current.timeoutMs);
		this.lastCloseCode = null;
		this.lastCloseReason = null;
		this.bindSocketEvents();
		this.logDebug("socket open");
		this.logInfo("connect success", `url=${target}`);
		return this.transition(ConnectionState.CONNECTED, "connected");
	}

	/** Closes socket connection and clears listeners. */
	override async close(): Promise<ConnectionStateResult> {
		this.emitState(this.ensureStateAllowed(ConnectionState.CLOSING));
		this.logDebug("close");
		this.autoReconnect = false;
		if (this.socket) this.socket.close();
		this.socket = null;
		this.listeners.clear();
		return this.transition(ConnectionState.CLOSED, "closed");
	}

	/** Reconnects by closing and opening a new socket session. */
	override async reconnect(): Promise<ConnectionStateResult> {
		this.logInfo("reconnect");
		await this.close();
		return this.connect(this.getOptions());
	}

	/** Returns the concrete WebSocket connection interface. */
	override connection(): WebSocketConnection {
		return this;
	}
	/** Generic execute alias for send operation. */
	async execute(
		request: WebSocketSendRequest,
	): Promise<Result<WebSocketMessage, LoomNetworkError>> {
		return this.send(request.message);
	}
	/** Sends one message over the open websocket connection. */
	async send(
		message: string,
	): Promise<Result<WebSocketMessage, LoomNetworkError>> {
		try {
			if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
				throw new LoomProtocolError("websocket is not open");
			}
			this.logDebug("send", `size=${message.length}`);
			this.socket.send(message);
			const value = toWebSocketMessage(
				this.socket,
				this.lastCloseCode,
				this.lastCloseReason,
				message,
			);
			return { ok: true, value };
		} catch (error) {
			return { ok: false, error: toLoomNetworkError(error as Error | object) };
		}
	}
	/** Registers a message listener and returns unsubscribe callback. */
	listen(handler: MessageHandler): () => void {
		this.logDebug("listen add");
		this.listeners.add(handler);
		return () => {
			this.listeners.delete(handler);
		};
	}
	/** Registers lifecycle state listener and returns unsubscribe callback. */
	onStateChange(handler: ConnectionStateHandler): () => void {
		this.stateListeners.add(handler);
		return () => {
			this.stateListeners.delete(handler);
		};
	}
	/** Enables/disables auto reconnect and optionally overrides reconnect delay. */
	setAutoReconnect(enabled: boolean, delayMs?: number): ConnectionStateResult {
		this.logInfo(
			"auto reconnect",
			`enabled=${enabled} delayMs=${String(delayMs ?? this.getOptions().websocketReconnectDelayMs)}`,
		);
		this.autoReconnect = enabled;
		if (delayMs && delayMs > 0) {
			const current = this.getOptions();
			this.setOptions({ ...current, websocketReconnectDelayMs: delayMs });
		}
		return {
			state: this.state(),
			previousState: this.state(),
			changed: false,
			timestamp: new Date().toISOString(),
			message: "auto reconnect updated",
		};
	}
	/** Attaches socket message/close handlers for listeners and reconnect flow. */
	private bindSocketEvents(): boolean {
		if (!this.socket) return false;
		return bindWebSocketListeners(
			this.socket,
			(value) => {
				this.logDebug("socket message", `size=${value.length}`);
				const payload = toWebSocketMessage(
					this.socket,
					this.lastCloseCode,
					this.lastCloseReason,
					value,
				);
				for (const listener of this.listeners) listener(payload);
			},
			(event) => {
				this.lastCloseCode = event.code;
				this.lastCloseReason = event.reason || null;
				this.socket = null;
				this.transition(ConnectionState.CLOSED, "socket closed");
				this.logWarn("socket close");
				this.handleAutoReconnect().catch(() => false);
			},
		);
	}
	/** Performs delayed reconnect when auto-reconnect is enabled. */
	private async handleAutoReconnect(): Promise<boolean> {
		if (!this.autoReconnect || this.reconnecting) return false;
		this.reconnecting = true;
		const delay = this.getOptions().websocketReconnectDelayMs;
		this.transition(ConnectionState.CONNECTING, "auto reconnect waiting");
		await Bun.sleep(delay);
		try {
			await this.connect(this.getOptions());
			return true;
		} finally {
			this.reconnecting = false;
		}
	}
	/** Applies state transition and publishes it to observers. */
	private transition(
		nextState: ConnectionState,
		message: string,
	): ConnectionStateResult {
		const result = this.setState(nextState, message);
		this.emitState(result);
		return result;
	}
	/** Emits one lifecycle transition to all subscribed handlers. */
	private emitState(result: ConnectionStateResult): boolean {
		for (const handler of this.stateListeners) handler(result);
		return true;
	}
}
