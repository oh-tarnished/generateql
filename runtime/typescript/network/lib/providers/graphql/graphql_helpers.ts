/**
 * GraphQL response normalization helpers for query/mutation transport envelopes.
 */
import type {
	GraphQLErrorDetail,
	GraphQLResult,
	JsonObject,
} from "../../types/types";
import type { GraphQLResponseMeta } from "./graphql_meta";

/** Converts raw GraphQL error messages into typed error details. */
function toGraphQLErrorDetails(messages: string[]): GraphQLErrorDetail[] {
	return messages.map((message) => ({
		message,
		path: [],
		code: "GRAPHQL_ERROR",
	}));
}

/** Builds the normalized GraphQL response envelope used by this package. */
function buildGraphQLResult<TData extends JsonObject = JsonObject>(
	data: TData | null,
	errors: GraphQLErrorDetail[],
	networkError: string | null,
	meta: GraphQLResponseMeta,
): GraphQLResult<TData> {
	return {
		meta: {
			status: meta.status,
			headers: meta.headers,
			transport: "graphql",
		},
		data,
		errors,
		networkError,
	};
}

/**
 * Builds a normalized response for transport-level GraphQL failures.
 *
 * @remarks
 * For pure transport failures `errors` stays empty and `networkError` is set.
 */
function buildGraphQLTransportFailure<TData extends JsonObject = JsonObject>(
	message: string,
	meta: GraphQLResponseMeta,
): GraphQLResult<TData> {
	// For pure transport failures keep GraphQL errors empty.
	return buildGraphQLResult<TData>(null, [], message, meta);
}

export {
	buildGraphQLResult,
	buildGraphQLTransportFailure,
	toGraphQLErrorDetails,
};
