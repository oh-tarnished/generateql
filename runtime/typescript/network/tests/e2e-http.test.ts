import { describe, expect, test } from "bun:test";
import { z } from "zod";
import { NetworkClient } from "../lib";
import { HTTPProtocol, NetworkType } from "../lib/types/types";
import { unwrapResult } from "./helpers/result";

describe("HTTP e2e", () => {
	test("connect and request return normalized HTTP envelope", async () => {
		const client = new NetworkClient(NetworkType.HTTPS, {
			url: { host: "jsonplaceholder.typicode.com", paths: ["/todos/1"] },
			httpProtocol: HTTPProtocol.HTTP1,
			timeoutMs: 8_000,
			retries: 1,
		});
		const connection = unwrapResult(await client.connect());

		const result = await connection.request<{ id: number; title: string }>({
			method: "GET",
			path: "/todos/1",
		});
		await connection.close();

		const response = unwrapResult(result);
		expect(response.meta.transport).toBe("http");
		expect(response.meta.status).toBe(200);
		expect(response.status).toBe(200);
		expect(response.ok).toBe(true);
		expect(response.data.id).toBe(1);
		expect(typeof response.headers["content-type"]).toBe("string");
	});

	test("request returns Result error for schema validation mismatch", async () => {
		const client = new NetworkClient(NetworkType.HTTPS, {
			url: { host: "jsonplaceholder.typicode.com", paths: ["/todos/1"] },
			timeoutMs: 8_000,
		});
		const connection = unwrapResult(await client.connect());

		const result = await connection.request({
			method: "GET",
			path: "/todos/1",
			responseSchema: z.object({ impossible: z.string() }),
		});
		await connection.close();

		expect(result.ok).toBe(false);
		if (result.ok) return;
		expect(
			["LoomNetworkError", "LoomConnectError", "LoomProtocolError"].includes(
				result.error.name,
			),
		).toBe(true);
	});
});
