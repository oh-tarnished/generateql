import { NetworkClient } from "@machanirobotics/loom-network";
import type { HTTPRequestObject } from "@machanirobotics/loom-network/types";
import {
	HTTPProtocol,
	NetworkType,
	type TransportMeta,
} from "@machanirobotics/loom-network/types";

export type DemoResult = {
	name: string;
	ok: boolean;
	meta: TransportMeta | null;
	request: unknown;
	response: unknown;
	error: string | null;
};

export async function runHttpDemo(): Promise<DemoResult> {
	const request: HTTPRequestObject = { method: "GET", path: "/todos/1" };
	const client = new NetworkClient(NetworkType.HTTPS, {
		url: { host: "jsonplaceholder.typicode.com", paths: ["/todos/1"] },
		timeoutMs: 8_000,
		retries: 1,
		httpProtocol: HTTPProtocol.HTTP1,
	});
	const connected = await client.connect();
	if (!connected.ok) {
		return {
			name: "HTTP",
			ok: false,
			meta: null,
			request,
			response: null,
			error: connected.error.message,
		};
	}
	const connection = connected.value;
	try {
		const result = await connection.request(request);
		if (!result.ok) {
			return {
				name: "HTTP",
				ok: false,
				meta: null,
				request,
				response: null,
				error: result.error.message,
			};
		}
		return {
			name: "HTTP",
			ok: true,
			meta: result.value.meta,
			request,
			response: result.value.data,
			error: null,
		};
	} catch (error) {
		return {
			name: "HTTP",
			ok: false,
			meta: null,
			request,
			response: null,
			error: error instanceof Error ? error.message : "HTTP request failed",
		};
	} finally {
		await connection.closeResult();
	}
}

export async function runGraphqlDemo(): Promise<DemoResult> {
	const request = {
		query: "CharacterById",
		variables: { id: "1" },
	};
	const client = new NetworkClient(NetworkType.GRAPHQL, {
		url: { host: "rickandmortyapi.com", paths: ["/graphql"] },
		timeoutMs: 10_000,
	});
	const connected = await client.connect();
	if (!connected.ok) {
		return {
			name: "GraphQL",
			ok: false,
			meta: null,
			request,
			response: null,
			error: connected.error.message,
		};
	}
	const connection = connected.value;
	try {
		const result = await connection.query<{
			character: { id: string; name: string; status: string };
		}>(
			"query CharacterById($id: ID!) { character(id: $id) { id name status } }",
			{ id: "1" },
		);
		if (!result.ok) {
			return {
				name: "GraphQL",
				ok: false,
				meta: null,
				request,
				response: null,
				error: result.error.message,
			};
		}
		return {
			name: "GraphQL",
			ok: !result.value.networkError && result.value.errors.length === 0,
			meta: result.value.meta,
			request,
			response: result.value,
			error: result.value.networkError,
		};
	} catch (error) {
		return {
			name: "GraphQL",
			ok: false,
			meta: null,
			request,
			response: null,
			error: error instanceof Error ? error.message : "GraphQL request failed",
		};
	} finally {
		await connection.closeResult();
	}
}

export async function runWebsocketDemo(): Promise<DemoResult> {
	const request = { message: "hello from svelte demo" };
	const client = new NetworkClient(NetworkType.WEBSOCKET, {
		url: { host: "ws.ifelse.io", paths: ["/"] },
		timeoutMs: 10_000,
	});
	const connected = await client.connect();
	if (!connected.ok) {
		return {
			name: "WebSocket",
			ok: false,
			meta: null,
			request,
			response: null,
			error: connected.error.message,
		};
	}
	const connection = connected.value;
	try {
		const sent = await connection.send(request.message);
		if (!sent.ok) {
			return {
				name: "WebSocket",
				ok: false,
				meta: null,
				request,
				response: null,
				error: sent.error.message,
			};
		}
		const received = await new Promise<string>((resolve) => {
			const off = connection.listen((event) => {
				if (event.data !== request.message) return;
				off();
				resolve(event.data);
			});
			setTimeout(() => {
				off();
				resolve("No echo message received before timeout");
			}, 4_000);
		});
		return {
			name: "WebSocket",
			ok: true,
			meta: sent.value.meta,
			request,
			response: { sent: sent.value.data, received },
			error: null,
		};
	} catch (error) {
		return {
			name: "WebSocket",
			ok: false,
			meta: null,
			request,
			response: null,
			error:
				error instanceof Error ? error.message : "WebSocket request failed",
		};
	} finally {
		await connection.closeResult();
	}
}
