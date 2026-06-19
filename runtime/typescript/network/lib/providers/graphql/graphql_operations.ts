/**
 * GraphQL query/mutation/subscription execution primitives.
 */
import { type ApolloClient, gql } from "@apollo/client/core";
import type { Client, SubscribePayload } from "graphql-ws";
import { LoomConnectError, LoomProtocolError } from "../../errors";
import {
	ConnectionState,
	type GraphQLResult,
	type GraphQLSubscriptionHandler,
	type JsonObject,
	type JsonValue,
} from "../../types/types";
import {
	buildGraphQLResult,
	buildGraphQLTransportFailure,
	toGraphQLErrorDetails,
} from "./graphql_helpers";
import type { GraphQLHTTPMetaTracker } from "./graphql_meta";
import {
	buildStreamEnvelope,
	extractStreamErrors,
	stringifySubscriptionError,
} from "./graphql_subscriptions";

/** Lazy accessor for the active Apollo GraphQL client. */
type RequireClient = () => ApolloClient;
/** Lazy accessor for the active GraphQL subscription client. */
type RequireSubscriptionClient = () => Client;
/** Read-only connection state getter. */
type StateReader = () => ConnectionState;

/**
 * Executes one GraphQL query and returns normalized result envelope.
 *
 * @param state - Current provider state reader.
 * @param requireClient - Lazy Apollo client accessor.
 * @param metaTracker - Request/response metadata tracker.
 * @param query - GraphQL query document string.
 * @param variables - Query variables map.
 * @returns Normalized GraphQL result with transport metadata.
 */
async function runGraphQLQuery<TData extends JsonObject = JsonObject>(
	state: StateReader,
	requireClient: RequireClient,
	metaTracker: GraphQLHTTPMetaTracker,
	query: string,
	variables: Record<string, JsonValue>,
): Promise<GraphQLResult<TData>> {
	if (state() !== ConnectionState.CONNECTED) {
		throw new LoomProtocolError("query requires connected state");
	}
	const client = requireClient();
	const requestId = metaTracker.createRequestId();
	try {
		const result = await client.query<TData>({
			query: gql(query),
			variables,
			context: { loomRequestId: requestId },
		});
		const errors = toGraphQLErrorDetails(
			result.error?.message ? [result.error.message] : [],
		);
		const meta = metaTracker.popResponseMeta(requestId);
		return buildGraphQLResult<TData>(result.data ?? null, errors, null, meta);
	} catch (error) {
		const message =
			error instanceof Error ? error.message : "graphql query failed";
		const meta = metaTracker.popResponseMeta(requestId);
		return buildGraphQLTransportFailure<TData>(message, meta);
	}
}

/**
 * Executes one GraphQL mutation and returns normalized result envelope.
 *
 * @param state - Current provider state reader.
 * @param requireClient - Lazy Apollo client accessor.
 * @param metaTracker - Request/response metadata tracker.
 * @param mutation - GraphQL mutation document string.
 * @param variables - Mutation variables map.
 * @returns Normalized GraphQL result with transport metadata.
 */
async function runGraphQLMutation<TData extends JsonObject = JsonObject>(
	state: StateReader,
	requireClient: RequireClient,
	metaTracker: GraphQLHTTPMetaTracker,
	mutation: string,
	variables: Record<string, JsonValue>,
): Promise<GraphQLResult<TData>> {
	if (state() !== ConnectionState.CONNECTED) {
		throw new LoomProtocolError("mutation requires connected state");
	}
	const client = requireClient();
	const requestId = metaTracker.createRequestId();
	try {
		const result = await client.mutate<TData>({
			mutation: gql(mutation),
			variables,
			context: { loomRequestId: requestId },
		});
		const errors = toGraphQLErrorDetails(
			result.error?.message ? [result.error.message] : [],
		);
		const meta = metaTracker.popResponseMeta(requestId);
		return buildGraphQLResult<TData>(result.data ?? null, errors, null, meta);
	} catch (error) {
		const message =
			error instanceof Error ? error.message : "graphql mutation failed";
		const meta = metaTracker.popResponseMeta(requestId);
		return buildGraphQLTransportFailure<TData>(message, meta);
	}
}

/**
 * Starts a GraphQL subscription stream and returns an unsubscribe function.
 *
 * @param state - Current provider state reader.
 * @param requireSubscriptionClient - Lazy subscription client accessor.
 * @param subscription - GraphQL subscription document string.
 * @param handler - Event callback for each streamed payload.
 * @param variables - Subscription variables map.
 * @returns Unsubscribe callback that stops the active stream.
 */
async function runGraphQLSubscription<TData extends JsonObject = JsonObject>(
	state: StateReader,
	requireSubscriptionClient: RequireSubscriptionClient,
	subscription: string,
	handler: GraphQLSubscriptionHandler<TData>,
	variables: Record<string, JsonValue>,
): Promise<() => void> {
	if (state() !== ConnectionState.CONNECTED) {
		throw new LoomProtocolError("subscription requires connected state");
	}
	const client = requireSubscriptionClient();
	const payload: SubscribePayload = { query: subscription, variables };
	let unsubscribe: () => void = () => false;
	unsubscribe = client.subscribe(payload, {
		next: (result) => {
			const details = extractStreamErrors(result.errors);
			handler(
				buildStreamEnvelope<TData>(
					(result.data as TData | null) ?? null,
					details,
					null,
				),
			);
		},
		error: (error) => {
			const message = stringifySubscriptionError(error);
			handler(buildStreamEnvelope<TData>(null, [], message));
		},
		complete: () => true,
	});
	return () => {
		unsubscribe();
		return true;
	};
}

/**
 * Validates a connectivity probe result before marking provider connected.
 *
 * @param probeResult - GraphQL probe response to validate.
 * @returns True when probe confirms GraphQL endpoint reachability.
 */
async function assertGraphQLConnectivity(
	probeResult: GraphQLResult<JsonObject>,
): Promise<boolean> {
	if (probeResult.networkError) {
		throw new LoomConnectError(probeResult.networkError);
	}
	if (probeResult.errors.length > 0) {
		throw new LoomConnectError(
			probeResult.errors[0]?.message ?? "graphql connectivity check failed",
		);
	}
	if (!probeResult.data) {
		throw new LoomConnectError(
			"graphql connectivity check returned empty data",
		);
	}
	return true;
}

export {
	assertGraphQLConnectivity,
	runGraphQLMutation,
	runGraphQLQuery,
	runGraphQLSubscription,
};
