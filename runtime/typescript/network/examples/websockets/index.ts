import { NetworkClient } from "@machanirobotics/loom-network";
import {
	type ConnectionStateResult,
	NetworkType,
} from "@machanirobotics/loom-network/types";
import { logResponse } from "../shared/printing";

export function runWebSocketExample(): Promise<boolean> {
	const client = new NetworkClient(NetworkType.WEBSOCKET, {
		url: { host: "ws.ifelse.io", paths: ["/"] },
		timeoutMs: 10_000,
		websocketReconnectDelayMs: 2_000,
	});

	return client
		.connect()
		.then((connected) => {
			if (!connected.ok) {
				console.warn("WebSocket connect failed:", connected.error.message);
				return false;
			}
			const connection = connected.value;
			const autoReconnect: ConnectionStateResult = connection.setAutoReconnect(
				true,
				2_000,
			);
			console.log("WebSocket state:", autoReconnect.state);
			const stopListening = connection.listen((response) => {
				console.log("WebSocket listen event");
				logResponse(response);
			});
			return connection
				.send("Hello from loom TypeScript provider package")
				.then((sent) => {
					if (!sent.ok) {
						console.warn("WebSocket send failed:", sent.error.message);
						return false;
					}
					console.log("WebSocket send event");
					logResponse(sent.value);
					return Bun.sleep(1_000).then(() => {
						stopListening();
						return true;
					});
				})
				.finally(() => {
					connection.closeResult().then(() => true);
				});
		})
		.catch((error) => {
			const message =
				error instanceof Error ? error.message : "websocket unreachable";
			console.warn("WebSocket example skipped:", message);
			return true;
		});
}

if (import.meta.main) {
	runWebSocketExample().then((ok) => {
		if (!ok) process.exitCode = 1;
	});
}
