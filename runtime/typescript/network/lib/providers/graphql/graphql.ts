/**
 * GraphQL provider public runtime API module.
 */
import type { LoomNetworkError } from "../../errors";
import type {
	GraphQLConnection,
	GraphQLRequestObject,
	GraphQLResult,
	GraphQLSubscriptionHandler,
	JsonObject,
	JsonValue,
	Result,
} from "../../types/types";
import { GraphQLBaseProvider } from "./graphql_base";
import {
	executeGraphQL,
	mutationGraphQL,
	queryGraphQL,
	subscribeGraphQL,
	toResult,
} from "./graphql_execution";

/**
 * GraphQL provider implementing query, mutation, and subscription operations.
 *
 * @remarks
 * HTTP request metadata is tracked per operation and returned in `meta`.
 * Subscriptions use `graphql-ws` and return stream envelopes with `meta.transport`.
 */
export class GraphQLProvider
	extends GraphQLBaseProvider
	implements GraphQLConnection
{
	/** Returns the typed GraphQL connection interface. */
	override connection(): GraphQLConnection {
		return this;
	}

	/**
	 * Executes a single GraphQL operation (query or mutation).
	 *
	 * @param request - GraphQL operation request payload.
	 * @returns Normalized GraphQL response envelope.
	 */
	async execute<TData extends JsonObject = JsonObject>(
		request: GraphQLRequestObject,
	): Promise<Result<GraphQLResult<TData>, LoomNetworkError>> {
		return toResult(() =>
			executeGraphQL<TData>(this.executionContext(), request),
		);
	}

	/**
	 * Runs a GraphQL query and returns normalized response envelope.
	 *
	 * @param query - GraphQL query document.
	 * @param variables - Optional query variables.
	 * @returns Normalized query result envelope.
	 */
	async query<TData extends JsonObject = JsonObject>(
		query: string,
		variables: Record<string, JsonValue> = {},
	): Promise<Result<GraphQLResult<TData>, LoomNetworkError>> {
		return toResult(() =>
			queryGraphQL<TData>(this.executionContext(), query, variables),
		);
	}

	/**
	 * Runs a GraphQL mutation and returns normalized response envelope.
	 *
	 * @param mutation - GraphQL mutation document.
	 * @param variables - Optional mutation variables.
	 * @returns Normalized mutation result envelope.
	 */
	async mutation<TData extends JsonObject = JsonObject>(
		mutation: string,
		variables: Record<string, JsonValue> = {},
	): Promise<Result<GraphQLResult<TData>, LoomNetworkError>> {
		return toResult(() =>
			mutationGraphQL<TData>(this.executionContext(), mutation, variables),
		);
	}

	/**
	 * Starts a GraphQL subscription stream and returns an unsubscribe function.
	 *
	 * @param subscription - GraphQL subscription document.
	 * @param handler - Callback invoked for each stream event envelope.
	 * @param variables - Optional subscription variables.
	 * @returns Unsubscribe callback for the active stream.
	 */
	async subscription<TData extends JsonObject = JsonObject>(
		subscription: string,
		handler: GraphQLSubscriptionHandler<TData>,
		variables: Record<string, JsonValue> = {},
	): Promise<Result<() => void, LoomNetworkError>> {
		return toResult(() =>
			subscribeGraphQL<TData>(
				this.executionContext(),
				subscription,
				handler,
				variables,
			),
		);
	}

	/** Runs the connectivity probe query used by `connect`. */
	protected async probeConnectivity(
		query: string,
	): Promise<GraphQLResult<JsonObject>> {
		return queryGraphQL<JsonObject>(this.executionContext(), query, {});
	}
}
