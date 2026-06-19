import { describe, expect, test } from "bun:test";
import { NetworkClient } from "../lib";
import { ConnectionState, NetworkType } from "../lib/types/types";
import { unwrapResult } from "./helpers/result";

describe("WebSocket e2e", () => {
	test("connect and send return normalized websocket envelope", async () => {
		const outbound = "loom-e2e";
		const client = new NetworkClient(NetworkType.WEBSOCKET, {
			url: { host: "ws.ifelse.io", paths: ["/"] },
			timeoutMs: 10_000,
			websocketReconnectDelayMs: 2_000,
		});
		const connection = unwrapResult(await client.connect());

		const received = await new Promise<string>((resolve, reject) => {
			const timeoutId = setTimeout(
				() => reject(new Error("websocket timeout")),
				10_000,
			);
			const off = connection.listen((message) => {
				if (message.data !== outbound) return;
				off();
				clearTimeout(timeoutId);
				resolve(message.data);
			});
			connection
				.send(outbound)
				.then((sent) => {
					if (!sent.ok) {
						reject(new Error(sent.error.message));
						return;
					}
					expect(sent.value.meta.transport).toBe("websocket");
					expect(sent.value.meta.connected).toBe(true);
				})
				.catch(reject);
		});
		await connection.close();

		expect(received).toBe(outbound);
	});

	test("state listener receives close transition", async () => {
		const client = new NetworkClient(NetworkType.WEBSOCKET, {
			url: { host: "ws.ifelse.io", paths: ["/"] },
			timeoutMs: 10_000,
		});
		const connection = unwrapResult(await client.connect());

		let sawClosed = false;
		const off = connection.onStateChange((state) => {
			if (state.state === ConnectionState.CLOSED) sawClosed = true;
		});
		await connection.close();
		off();

		expect(sawClosed).toBe(true);
	});
});
