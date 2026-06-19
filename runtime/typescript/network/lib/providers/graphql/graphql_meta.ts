/**
 * GraphQL HTTP request/response metadata correlation utilities.
 */
import { ApolloLink } from "@apollo/client/core";

/** Captured network metadata for one GraphQL HTTP operation. */
type GraphQLResponseMeta = {
	status: number;
	headers: Record<string, string>;
};

type HeaderValue = string | number | boolean;
type HeaderObject = Record<string, HeaderValue | readonly string[]>;
type HeaderList = Array<string[]>;
type HeaderInput = Headers | HeaderObject | HeaderList | undefined;

/**
 * Tracks request-scoped HTTP status/headers for Apollo operations.
 *
 * @remarks
 * Apollo does not expose response metadata per operation by default, so this
 * helper injects a request id and stores the corresponding fetch response meta.
 */
class GraphQLHTTPMetaTracker {
	/** Per-request metadata store keyed by generated request id. */
	private readonly responseMetaByRequestId: Map<string, GraphQLResponseMeta> =
		new Map<string, GraphQLResponseMeta>();

	/** Creates a fetch wrapper that records status/headers by request id. */
	createTrackedFetch(): typeof fetch {
		return (async (
			input: Parameters<typeof fetch>[0],
			init?: Parameters<typeof fetch>[1],
		): Promise<Response> => {
			const response = await fetch(input, init);
			const requestId = this.extractRequestIdFromInit(init);
			if (requestId) {
				this.responseMetaByRequestId.set(requestId, {
					status: response.status,
					headers: this.responseHeadersToObject(response.headers),
				});
			}
			return response;
		}) as typeof fetch;
	}

	/** Creates an Apollo link that injects the request id header. */
	createRequestIdLink(): ApolloLink {
		return new ApolloLink((operation, forward) => {
			const requestId = operation.getContext().loomRequestId as
				| string
				| undefined;
			if (!requestId || !forward) return forward(operation);
			const context = operation.getContext() as { headers?: HeaderInput };
			const existingHeaders = this.headersToObject(context.headers);
			operation.setContext({
				headers: {
					...existingHeaders,
					"x-loom-request-id": requestId,
				},
			});
			return forward(operation);
		});
	}

	/** Returns and removes metadata for a request id. */
	popResponseMeta(requestId: string): GraphQLResponseMeta {
		const matched = this.responseMetaByRequestId.get(requestId);
		if (!matched) return { status: 0, headers: {} };
		this.responseMetaByRequestId.delete(requestId);
		return matched;
	}

	/** Generates a stable request id used for meta correlation. */
	createRequestId(): string {
		const scope = globalThis as { crypto?: { randomUUID?: () => string } };
		if (scope.crypto?.randomUUID) return scope.crypto.randomUUID();
		return `${Date.now()}-${Math.floor(Math.random() * 1_000_000)}`;
	}

	/** Clears all tracked request metadata. */
	clear(): boolean {
		this.responseMetaByRequestId.clear();
		return true;
	}

	/** Converts `Headers` into a plain object map. */
	private responseHeadersToObject(headers: Headers): Record<string, string> {
		const output: Record<string, string> = {};
		for (const [key, value] of headers.entries()) output[key] = value;
		return output;
	}

	/** Normalizes all supported header input forms into string map. */
	private headersToObject(headers: HeaderInput): Record<string, string> {
		if (!headers) return {};
		if (headers instanceof Headers)
			return this.responseHeadersToObject(headers);
		if (Array.isArray(headers)) {
			const output: Record<string, string> = {};
			for (const item of headers) {
				const key = item[0];
				const value = item[1];
				if (typeof key === "string" && typeof value !== "undefined") {
					output[key] = String(value);
				}
			}
			return output;
		}
		if (typeof headers === "object") {
			const output: Record<string, string> = {};
			for (const [key, value] of Object.entries(headers)) {
				output[key] = Array.isArray(value) ? value.join(", ") : String(value);
			}
			return output;
		}
		return {};
	}

	/** Reads the tracking request id from fetch init headers. */
	private extractRequestIdFromInit(
		init: Parameters<typeof fetch>[1] | undefined,
	): string | null {
		if (!init) return null;
		return this.headersToObject(init.headers)["x-loom-request-id"] ?? null;
	}
}

export type { GraphQLResponseMeta };
export { GraphQLHTTPMetaTracker };
