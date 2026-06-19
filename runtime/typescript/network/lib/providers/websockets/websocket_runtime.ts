/**
 * Low-level websocket runtime helpers for open/listen/message conversion flows.
 */
import { LoomConnectError, LoomTimeoutError } from "../../errors";
import type { WebSocketMessage } from "../../types/types";

/** Message callback invoked for each incoming string payload. */
type IncomingMessageHandler = (message: string) => void;

/** Close callback invoked when socket closes. */
type CloseHandler = (event: CloseEvent) => void;

/**
 * Opens a websocket with timeout handling and resolves the connected socket.
 *
 * @param url - Fully qualified websocket URL.
 * @param timeoutMs - Maximum wait time for `onopen`.
 * @returns Connected websocket instance.
 */
async function openWebSocket(
	url: string,
	timeoutMs: number,
): Promise<WebSocket> {
	return new Promise<WebSocket>((resolve, reject) => {
		const socket = new WebSocket(url);
		const timeoutHandle = setTimeout(() => {
			socket.close();
			reject(new LoomTimeoutError(`websocket timeout after ${timeoutMs}ms`));
		}, timeoutMs);
		socket.onopen = () => {
			clearTimeout(timeoutHandle);
			resolve(socket);
		};
		socket.onerror = () => {
			clearTimeout(timeoutHandle);
			reject(new LoomConnectError("websocket connection failed"));
		};
	});
}

/**
 * Binds message and close listeners to a websocket instance.
 *
 * @param socket - Connected websocket object.
 * @param onMessage - Handler for incoming payload text.
 * @param onClose - Handler for close events.
 * @returns True once listeners are attached.
 */
function bindWebSocketListeners(
	socket: WebSocket,
	onMessage: IncomingMessageHandler,
	onClose: CloseHandler,
): boolean {
	socket.onmessage = (event) => {
		const value =
			typeof event.data === "string" ? event.data : String(event.data);
		onMessage(value);
	};
	socket.onclose = onClose;
	return true;
}

/**
 * Converts payload and socket state into normalized websocket envelope.
 *
 * @param socket - Active websocket instance or null.
 * @param lastCloseCode - Last observed close code.
 * @param lastCloseReason - Last observed close reason.
 * @param message - Message payload string.
 * @returns Normalized websocket message envelope.
 */
function toWebSocketMessage(
	socket: WebSocket | null,
	lastCloseCode: number | null,
	lastCloseReason: string | null,
	message: string,
): WebSocketMessage {
	return {
		meta: {
			transport: "websocket",
			connected: socket?.readyState === WebSocket.OPEN,
			protocol: socket?.protocol ?? "",
			closeCode: lastCloseCode,
			closeReason: lastCloseReason,
		},
		data: message,
		errors: [],
	};
}

export { bindWebSocketListeners, openWebSocket, toWebSocketMessage };
