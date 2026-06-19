/**
 * GraphQL operation, response, and subscription contract definitions.
 */
import type { LoomNetworkError } from "../errors";
import type { HTTPGraphQLMeta } from "./http";
import type { JsonObject, JsonValue } from "./json";
import type { Result, TransportConnection } from "./lifecycle";

/** Structured GraphQL error detail normalized by the provider. */
export interface GraphQLErrorDetail {
	/** Human-readable GraphQL error message. */
	message: string;
	/** Path of the failing field/resolver segments as strings. */
	path: string[];
	/** Error code from extensions, or `GRAPHQL_ERROR` fallback. */
	code: string;
}

/** Normalized GraphQL response envelope. */
export interface GraphQLResult<TData extends JsonObject = JsonObject> {
	/** Unified metadata envelope including status, headers, and transport kind. */
	meta: HTTPGraphQLMeta<"graphql">;
	/** Parsed GraphQL `data` payload, or `null` when request fails. */
	data: TData | null;
	/** GraphQL execution/validation errors for this operation. */
	errors: GraphQLErrorDetail[];
	/** Transport-level failure message (fetch/socket/protocol), if any. */
	networkError: string | null;
}

/** GraphQL operation kinds accepted by `execute`. */
export type GraphQLOperationType = "query" | "mutation" | "subscription";

/** Request object used by generic GraphQL `execute`. */
export interface GraphQLRequestObject {
	/** Operation kind discriminator. */
	operation: GraphQLOperationType;
	/** GraphQL operation document string. */
	document: string;
	/** Optional variable map for the document. */
	variables?: Record<string, JsonValue>;
}

/** Callback signature for streamed GraphQL subscription results. */
export type GraphQLSubscriptionHandler<TData extends JsonObject = JsonObject> =
	(result: GraphQLResult<TData>) => void;

/** GraphQL transport contract for query, mutation, and subscription APIs. */
export interface GraphQLConnection
	extends TransportConnection<GraphQLRequestObject, GraphQLResult<JsonObject>> {
	/** Executes one query or mutation operation from a request object. */
	execute<TData extends JsonObject = JsonObject>(
		request: GraphQLRequestObject,
	): Promise<Result<GraphQLResult<TData>, LoomNetworkError>>;
	/** Executes a GraphQL query operation. */
	query<TData extends JsonObject = JsonObject>(
		query: string,
		variables?: Record<string, JsonValue>,
	): Promise<Result<GraphQLResult<TData>, LoomNetworkError>>;
	/** Executes a GraphQL mutation operation. */
	mutation<TData extends JsonObject = JsonObject>(
		mutation: string,
		variables?: Record<string, JsonValue>,
	): Promise<Result<GraphQLResult<TData>, LoomNetworkError>>;
	/** Starts a GraphQL subscription stream and returns an unsubscribe function. */
	subscription<TData extends JsonObject = JsonObject>(
		subscription: string,
		handler: GraphQLSubscriptionHandler<TData>,
		variables?: Record<string, JsonValue>,
	): Promise<Result<() => void, LoomNetworkError>>;
}
