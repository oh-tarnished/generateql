import { afterEach, beforeEach, describe, expect, test } from "bun:test";
import { GraphQLProvider } from "../lib/providers/graphql/graphql";
import { URLScheme } from "../lib/types/types";
import { unwrapResult } from "./helpers/result";

type FetchMock = typeof fetch;

const baseOptions = {
	url: {
		scheme: URLScheme.HTTPS,
		host: "example.com",
		paths: ["/graphql"],
	},
	skipConnectivityCheck: true,
	timeoutMs: 2_000,
};

function jsonResponse(
	body: object,
	status: number,
	headers: Record<string, string>,
): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: {
			"content-type": "application/json",
			...headers,
		},
	});
}

function parseBody(init: RequestInit | undefined): {
	query: string;
	variables: Record<string, string>;
} {
	if (!init?.body || typeof init.body !== "string") {
		return { query: "", variables: {} };
	}
	const parsed = JSON.parse(init.body) as {
		query?: string;
		variables?: Record<string, string>;
	};
	return {
		query: parsed.query ?? "",
		variables: parsed.variables ?? {},
	};
}

describe("GraphQLProvider concurrency and subscription normalization", () => {
	let originalFetch: FetchMock;

	beforeEach(() => {
		originalFetch = globalThis.fetch;
	});

	afterEach(() => {
		globalThis.fetch = originalFetch;
	});

	test("concurrent queries keep per-request meta attached", async () => {
		globalThis.fetch = (async (_input, init) => {
			const { variables } = parseBody(init);
			const id = variables.id ?? "0";
			const status = id === "1" ? 201 : 202;
			const delay = id === "1" ? 50 : 5;
			await Bun.sleep(delay);
			return jsonResponse({ data: { item: { id } } }, status, {
				"x-case-id": id,
			});
		}) as FetchMock;

		const provider = new GraphQLProvider(baseOptions);
		await provider.connect();
		const [first, second] = await Promise.all([
			provider.query<{ item: { id: string } }>(
				"query One($id: ID!) { item(id: $id) { id } }",
				{ id: "1" },
			),
			provider.query<{ item: { id: string } }>(
				"query Two($id: ID!) { item(id: $id) { id } }",
				{ id: "2" },
			),
		]);
		await provider.close();

		const firstValue = unwrapResult(first);
		const secondValue = unwrapResult(second);
		expect(firstValue.data?.item.id).toBe("1");
		expect(firstValue.meta.status).toBe(201);
		expect(firstValue.meta.headers["x-case-id"]).toBe("1");

		expect(secondValue.data?.item.id).toBe("2");
		expect(secondValue.meta.status).toBe(202);
		expect(secondValue.meta.headers["x-case-id"]).toBe("2");
	});

	test("parallel query and mutation keep isolated response meta", async () => {
		globalThis.fetch = (async (_input, init) => {
			const { query, variables } = parseBody(init);
			const isMutation = query.includes("mutation");
			const id = variables.id ?? "0";
			const status = isMutation ? 212 : 211;
			const delay = isMutation ? 5 : 50;
			await Bun.sleep(delay);
			return jsonResponse({ data: { item: { id } } }, status, {
				"x-op-kind": isMutation ? "mutation" : "query",
			});
		}) as FetchMock;

		const provider = new GraphQLProvider(baseOptions);
		await provider.connect();
		const [queryResult, mutationResult] = await Promise.all([
			provider.query<{ item: { id: string } }>(
				"query Q($id: ID!) { item(id: $id) { id } }",
				{ id: "q" },
			),
			provider.mutation<{ item: { id: string } }>(
				"mutation M($id: ID!) { item(id: $id) { id } }",
				{ id: "m" },
			),
		]);
		await provider.close();

		const queryValue = unwrapResult(queryResult);
		const mutationValue = unwrapResult(mutationResult);
		expect(queryValue.data?.item.id).toBe("q");
		expect(queryValue.meta.status).toBe(211);
		expect(queryValue.meta.headers["x-op-kind"]).toBe("query");

		expect(mutationValue.data?.item.id).toBe("m");
		expect(mutationValue.meta.status).toBe(212);
		expect(mutationValue.meta.headers["x-op-kind"]).toBe("mutation");
	});

	test("subscription transport errors return normalized envelope", async () => {
		const provider = new GraphQLProvider(baseOptions);
		await provider.connect();
		(
			provider as unknown as {
				subscriptionClient: {
					subscribe: (
						payload: object,
						sink: {
							next?: (value: object) => void;
							error?: (error: unknown) => void;
							complete?: () => void;
						},
					) => () => void;
					dispose: () => void;
				};
			}
		).subscriptionClient = {
			subscribe: (_payload, sink) => {
				sink.error?.(new Error("subscription transport failed"));
				return () => true;
			},
			dispose: () => true,
		};

		let event: {
			networkError: string | null;
			errorsCount: number;
			status: number;
			transport: string;
		} | null = null;

		const stream = await provider.subscription<{ ping: string }>(
			"subscription Ping { ping }",
			(result) => {
				event = {
					networkError: result.networkError,
					errorsCount: result.errors.length,
					status: result.meta.status,
					transport: result.meta.transport,
				};
			},
		);
		unwrapResult(stream)();
		await provider.close();

		const captured = event as unknown as {
			networkError: string | null;
			errorsCount: number;
			status: number;
			transport: string;
		};
		expect(captured.networkError).toBe("subscription transport failed");
		expect(captured.errorsCount).toBe(0);
		expect(captured.status).toBe(0);
		expect(captured.transport).toBe("graphql");
	});
});
