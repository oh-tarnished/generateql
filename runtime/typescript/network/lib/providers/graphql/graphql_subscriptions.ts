/**
 * GraphQL websocket subscription transport helper utilities.
 */
import { type Client, createClient } from "graphql-ws";
import type {
	GraphQLErrorDetail,
	GraphQLResult,
	JsonObject,
} from "../../types/types";

/** Converts a GraphQL HTTP(S) URL to a WebSocket subscription URL. */
function toSubscriptionURL(httpUrl: string): string {
	const parsed = new URL(httpUrl);
	parsed.protocol = parsed.protocol === "https:" ? "wss:" : "ws:";
	return parsed.toString();
}

/** Creates a lazy `graphql-ws` client used for subscriptions. */
function createSubscriptionClient(
	httpUrl: string,
	headers: Record<string, string>,
): Client {
	return createClient({
		url: toSubscriptionURL(httpUrl),
		connectionParams: headers,
		webSocketImpl: WebSocket,
		lazy: true,
	});
}

/** Normalizes streamed GraphQL errors emitted by subscription payloads. */
function extractStreamErrors(
	errors:
		| readonly {
				message: string;
				path?: readonly (string | number)[];
				extensions?: object;
		  }[]
		| undefined,
): GraphQLErrorDetail[] {
	if (!errors || errors.length === 0) return [];
	return errors.map((errorItem) => ({
		// `extensions` is loosely typed across GraphQL runtimes, parse only known key.
		...(() => {
			const extension = errorItem.extensions as
				| { code?: string | number | boolean | null }
				| undefined;
			const code = extension?.code;
			return { code: typeof code === "string" ? code : "GRAPHQL_ERROR" };
		})(),
		message: errorItem.message,
		path: (errorItem.path ?? []).map((part) => String(part)),
	}));
}

/** Converts subscription transport errors into a display-safe message. */
function stringifySubscriptionError(error: unknown): string {
	if (Array.isArray(error)) {
		return error
			.map((item) =>
				item instanceof Error ? item.message : JSON.stringify(item),
			)
			.join(", ");
	}
	if (error instanceof Error) return error.message;
	return JSON.stringify(error);
}

/** Builds normalized envelope for GraphQL subscription events. */
function buildStreamEnvelope<TData extends JsonObject = JsonObject>(
	data: TData | null,
	errors: GraphQLErrorDetail[],
	networkError: string | null,
): GraphQLResult<TData> {
	return {
		meta: { status: 0, headers: {}, transport: "graphql" },
		data,
		errors,
		networkError,
	};
}

export {
	buildStreamEnvelope,
	createSubscriptionClient,
	extractStreamErrors,
	stringifySubscriptionError,
	toSubscriptionURL,
};
