/**
 * GraphQL operation dispatch and `Result` helper utilities.
 */
import type { ApolloClient } from "@apollo/client/core";
import type { Client } from "graphql-ws";
import {
	type LoomNetworkError,
	LoomProtocolError,
	toLoomNetworkError,
} from "../../errors";
import type {
	GraphQLRequestObject,
	GraphQLResult,
	GraphQLSubscriptionHandler,
	JsonObject,
	JsonValue,
	Result,
} from "../../types/types";
import type { GraphQLHTTPMetaTracker } from "./graphql_meta";
import {
	runGraphQLMutation,
	runGraphQLQuery,
	runGraphQLSubscription,
} from "./graphql_operations";

/** Runtime execution context passed from `GraphQLProvider` into helper functions. */
interface GraphQLExecutionContext {
	/** Provider state getter used by operation guards. */
	state: () => import("../../types/types").ConnectionState;
	/** Connected Apollo client accessor. */
	requireClient: () => ApolloClient;
	/** Connected `graphql-ws` client accessor. */
	requireSubscriptionClient: () => Client;
	/** Per-request HTTP metadata tracker for GraphQL operations. */
	metaTracker: GraphQLHTTPMetaTracker;
	/** Structured debug logger callback. */
	logDebug: (action: string, detail?: string) => boolean;
	/** Structured info logger callback. */
	logInfo: (action: string, detail?: string) => boolean;
}

/** Converts thrown errors into package `Result` envelopes. */
async function toResult<TValue>(
	task: () => Promise<TValue>,
): Promise<Result<TValue, LoomNetworkError>> {
	try {
		const value = await task();
		return { ok: true, value };
	} catch (error) {
		return { ok: false, error: toLoomNetworkError(error as Error | object) };
	}
}

/** Executes one GraphQL operation using query/mutation dispatch rules. */
async function executeGraphQL<TData extends JsonObject = JsonObject>(
	context: GraphQLExecutionContext,
	request: GraphQLRequestObject,
): Promise<GraphQLResult<TData>> {
	if (request.operation === "query") {
		return queryGraphQL<TData>(
			context,
			request.document,
			request.variables ?? {},
		);
	}
	if (request.operation === "subscription") {
		throw new LoomProtocolError(
			"subscription operation is streaming; use subscription() instead",
		);
	}
	return mutationGraphQL<TData>(
		context,
		request.document,
		request.variables ?? {},
	);
}

/** Executes one GraphQL query and records operation metadata. */
async function queryGraphQL<TData extends JsonObject = JsonObject>(
	context: GraphQLExecutionContext,
	query: string,
	variables: Record<string, JsonValue> = {},
): Promise<GraphQLResult<TData>> {
	context.logDebug("query", `variables=${Object.keys(variables).length}`);
	return runGraphQLQuery<TData>(
		context.state,
		context.requireClient,
		context.metaTracker,
		query,
		variables,
	);
}

/** Executes one GraphQL mutation and records operation metadata. */
async function mutationGraphQL<TData extends JsonObject = JsonObject>(
	context: GraphQLExecutionContext,
	mutation: string,
	variables: Record<string, JsonValue> = {},
): Promise<GraphQLResult<TData>> {
	context.logDebug("mutation", `variables=${Object.keys(variables).length}`);
	return runGraphQLMutation<TData>(
		context.state,
		context.requireClient,
		context.metaTracker,
		mutation,
		variables,
	);
}

/** Starts a GraphQL subscription stream and returns an unsubscribe callback. */
async function subscribeGraphQL<TData extends JsonObject = JsonObject>(
	context: GraphQLExecutionContext,
	subscription: string,
	handler: GraphQLSubscriptionHandler<TData>,
	variables: Record<string, JsonValue> = {},
): Promise<() => void> {
	context.logInfo(
		"subscription start",
		`variables=${Object.keys(variables).length}`,
	);
	const unsubscribe = await runGraphQLSubscription<TData>(
		context.state,
		context.requireSubscriptionClient,
		subscription,
		handler,
		variables,
	);
	return () => {
		unsubscribe();
		context.logInfo("subscription stop");
	};
}

export type { GraphQLExecutionContext };
export {
	executeGraphQL,
	mutationGraphQL,
	queryGraphQL,
	subscribeGraphQL,
	toResult,
};
